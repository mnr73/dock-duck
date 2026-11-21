// Package updater checks GitHub Releases for a newer DockDuck and replaces the
// running binary with it.
//
// Trust: the archive is only installed when its SHA-256 matches the line in the
// release's checksums.txt. Both come from the same GitHub release over HTTPS, so
// this catches corruption and a half-uploaded release, not a compromised
// repository; that is the same level of trust as `curl | sh` from the same repo.
//
// Nothing here runs by itself: a check or an update happens only when the user
// asks (Settings, or `dockduck update`). DockDuck never phones home on its own.
package updater

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository releases are read from.
const Repo = "mnr73/dock-duck"

var (
	ErrDevBuild    = errors.New("this is a development build, it has no version to compare")
	ErrNotWritable = errors.New("the binary's folder is not writable by this user")
	ErrNoAsset     = errors.New("this release has no build for this system")
	ErrChecksum    = errors.New("checksum mismatch: the download is corrupt, nothing was installed")
)

// Release is what the latest GitHub release says.
type Release struct {
	Tag    string  `json:"tag"`   // "v0.2.0"
	URL    string  `json:"url"`   // release page
	Notes  string  `json:"notes"` // release body (markdown), trimmed
	Date   string  `json:"date"`  // published_at
	Assets []Asset `json:"-"`
}

// Asset is one downloadable file of a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Status is the answer to "is there a newer version".
type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest,omitempty"`
	Available bool   `json:"available"`
	URL       string `json:"url,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Date      string `json:"date,omitempty"`
	// CanInstall is false when this build cannot replace itself (dev build, or the
	// binary sits in a folder this user cannot write); Reason says why.
	CanInstall bool   `json:"can_install"`
	Reason     string `json:"reason,omitempty"`
	// Managed is true when a service manager (systemd) will restart us after an update.
	Managed bool `json:"managed"`
}

// Updater talks to GitHub and to the file system.
type Updater struct {
	Version string       // running version ("dev" for local builds)
	API     string       // GitHub API base, default https://api.github.com (tests)
	HTTP    *http.Client // default: 30 s timeout, no redirects to other schemes
	// Exe is the path of the running binary; default os.Executable().
	Exe  string
	OS   string // default runtime.GOOS
	Arch string // default runtime.GOARCH
}

// EnvAPI points the updater at another GitHub-compatible API (a mirror, or a test
// server). It is read once at start-up from the process environment, so only
// whoever launches DockDuck can set it; nothing in the web UI can.
const EnvAPI = "DOCKDUCK_UPDATE_API"

func (u *Updater) api() string {
	if u.API != "" {
		return strings.TrimRight(u.API, "/")
	}
	if v := os.Getenv(EnvAPI); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://api.github.com"
}

func (u *Updater) client() *http.Client {
	if u.HTTP != nil {
		return u.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (u *Updater) goos() string {
	if u.OS != "" {
		return u.OS
	}
	return runtime.GOOS
}

func (u *Updater) goarch() string {
	if u.Arch != "" {
		return u.Arch
	}
	return runtime.GOARCH
}

func (u *Updater) exe() (string, error) {
	if u.Exe != "" {
		return u.Exe, nil
	}
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	// a symlink (/usr/local/bin/dockduck -> /opt/...) must be replaced at its target
	return filepath.EvalSymlinks(p)
}

func (u *Updater) get(ctx context.Context, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "dockduck-updater")
	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no release found (the repository has none yet)")
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("GitHub refused the request (rate limit); try again in a few minutes")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("response is larger than expected")
	}
	return b, nil
}

// Latest fetches the newest published, non-prerelease release.
func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	b, err := u.get(ctx, u.api()+"/repos/"+Repo+"/releases/latest", 4<<20)
	if err != nil {
		return nil, err
	}
	var r struct {
		Tag    string  `json:"tag_name"`
		URL    string  `json:"html_url"`
		Body   string  `json:"body"`
		Date   string  `json:"published_at"`
		Draft  bool    `json:"draft"`
		Pre    bool    `json:"prerelease"`
		Assets []Asset `json:"assets"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("unexpected answer from GitHub: %w", err)
	}
	if r.Tag == "" || r.Draft || r.Pre {
		return nil, errors.New("no stable release found")
	}
	notes := strings.TrimSpace(r.Body)
	if len(notes) > 4000 {
		notes = notes[:4000] + "…"
	}
	return &Release{Tag: r.Tag, URL: r.URL, Notes: notes, Date: r.Date, Assets: r.Assets}, nil
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+.*)?$`)

type semver struct {
	n   [3]int
	pre string
}

func parse(s string) (semver, bool) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	var v semver
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return semver{}, false
		}
		v.n[i] = n
	}
	v.pre = m[4]
	return v, true
}

// Newer reports whether latest is a higher version than current. Unparseable
// input is never "newer": an update must not be offered on a guess.
func Newer(latest, current string) bool {
	l, ok1 := parse(latest)
	c, ok2 := parse(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l.n[i] != c.n[i] {
			return l.n[i] > c.n[i]
		}
	}
	// 1.0.0 > 1.0.0-rc1; two prereleases compare as text (good enough, never installed)
	switch {
	case l.pre == c.pre:
		return false
	case l.pre == "":
		return true
	case c.pre == "":
		return false
	}
	return l.pre > c.pre
}

// canInstall says whether this process may replace its own binary.
func (u *Updater) canInstall() (bool, string) {
	if _, ok := parse(u.Version); !ok {
		return false, ErrDevBuild.Error()
	}
	exe, err := u.exe()
	if err != nil {
		return false, "cannot locate the running binary"
	}
	if err := writableDir(filepath.Dir(exe)); err != nil {
		return false, ErrNotWritable.Error() + ": run `sudo dockduck update`, or the install script again"
	}
	return true, ""
}

func writableDir(dir string) error {
	f, err := os.CreateTemp(dir, ".dockduck-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// Managed reports whether a service manager restarts DockDuck after it exits.
// systemd sets INVOCATION_ID for every unit it starts.
func Managed() bool { return os.Getenv("INVOCATION_ID") != "" }

// Check compares the running version with the latest release.
func (u *Updater) Check(ctx context.Context) (Status, error) {
	st := Status{Current: u.Version, Managed: Managed()}
	st.CanInstall, st.Reason = u.canInstall()
	rel, err := u.Latest(ctx)
	if err != nil {
		return st, err
	}
	st.Latest, st.URL, st.Notes, st.Date = rel.Tag, rel.URL, rel.Notes, rel.Date
	st.Available = Newer(rel.Tag, u.Version)
	return st, nil
}

func (u *Updater) assetName(tag string) string {
	return fmt.Sprintf("dockduck_%s_%s_%s.tar.gz", strings.TrimPrefix(tag, "v"), u.goos(), u.goarch())
}

// Install downloads the release, verifies it and replaces the running binary.
// The new file is written next to the old one and renamed over it, so a failure
// at any point leaves the working binary untouched. The process keeps running the
// old code until it is restarted. Returns the installed tag.
func (u *Updater) Install(ctx context.Context) (string, error) {
	if ok, why := u.canInstall(); !ok {
		return "", errors.New(why)
	}
	rel, err := u.Latest(ctx)
	if err != nil {
		return "", err
	}
	if !Newer(rel.Tag, u.Version) {
		return rel.Tag, fmt.Errorf("already up to date (%s)", u.Version)
	}
	want := u.assetName(rel.Tag)
	var archive, sums *Asset
	for i := range rel.Assets {
		switch rel.Assets[i].Name {
		case want:
			archive = &rel.Assets[i]
		case "checksums.txt":
			sums = &rel.Assets[i]
		}
	}
	if archive == nil {
		return "", fmt.Errorf("%w (%s/%s)", ErrNoAsset, u.goos(), u.goarch())
	}
	if sums == nil {
		return "", errors.New("the release has no checksums.txt, refusing to install an unverified download")
	}
	if archive.Size > 200<<20 {
		return "", errors.New("the archive is unexpectedly large, refusing")
	}
	sumText, err := u.get(ctx, sums.URL, 1<<20)
	if err != nil {
		return "", err
	}
	wantSum, err := checksumFor(string(sumText), want)
	if err != nil {
		return "", err
	}
	data, err := u.get(ctx, archive.URL, 200<<20)
	if err != nil {
		return "", err
	}
	got := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(got[:]), wantSum) {
		return "", ErrChecksum
	}
	bin, err := extract(data)
	if err != nil {
		return "", err
	}
	exe, err := u.exe()
	if err != nil {
		return "", err
	}
	if err := replace(exe, bin); err != nil {
		return "", err
	}
	return rel.Tag, nil
}

// checksumFor finds the SHA-256 of name in a `sha256sum`-style file.
func checksumFor(text, name string) (string, error) {
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name && len(f[0]) == 64 {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("checksums.txt has no entry for %s", name)
}

// extract reads the `dockduck` file out of the tar.gz. Only that one regular
// file is read; any other entry (and any path trick) is ignored.
func extract(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("the archive is not valid: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the archive has no dockduck binary")
		}
		if err != nil {
			return nil, fmt.Errorf("the archive is not valid: %w", err)
		}
		if h.Typeflag != tar.TypeReg || h.Name != "dockduck" {
			continue
		}
		if h.Size <= 0 || h.Size > 150<<20 {
			return nil, errors.New("the binary in the archive has an implausible size")
		}
		b, err := io.ReadAll(io.LimitReader(tr, h.Size))
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}

// replace atomically swaps the file at exe for content, keeping its permissions.
func replace(exe string, content []byte) error {
	mode := os.FileMode(0o755)
	if st, err := os.Stat(exe); err == nil {
		mode = st.Mode().Perm() | 0o111
	}
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".dockduck-new-*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	// Linux and macOS let a running binary be replaced by rename: the old inode
	// stays alive for the running process.
	if err := os.Rename(name, exe); err != nil {
		cleanup()
		return err
	}
	return nil
}
