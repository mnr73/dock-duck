package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "0.1.0", true},
		{"v0.1.1", "v0.1.0", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.10.0", "v0.9.0", true}, // numeric, not text, comparison
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v1.0.0", "v1.0.0-rc1", true},
		{"v1.0.0-rc1", "v1.0.0", false},
		{"v0.2.0", "dev", false},
		{"v0.2.0", "b8992ce-dirty", false},
		{"latest", "v0.1.0", false},
		{"", "v0.1.0", false},
		{"v0.2.0", "", false},
	}
	for _, c := range cases {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func tarGz(t *testing.T, files map[string][]byte, types map[string]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		typ := byte(tar.TypeReg)
		if tp, ok := types[name]; ok {
			typ = tp
		}
		h := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: typ}
		if typ == tar.TypeSymlink {
			h.Linkname, h.Size = "/etc/passwd", 0
			body = nil
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write(body)
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

type fake struct {
	srv     *httptest.Server
	archive []byte
	tag     string
	sums    string // override; "" = correct
	noSums  bool
	pre     bool
}

func newFake(t *testing.T, archive []byte) *fake {
	f := &fake{archive: archive, tag: "v0.2.0"}
	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		name := "dockduck_" + strings.TrimPrefix(f.tag, "v") + "_linux_amd64.tar.gz"
		assets := []Asset{{Name: name, URL: base + "/dl/" + name, Size: int64(len(f.archive))}}
		if !f.noSums {
			assets = append(assets, Asset{Name: "checksums.txt", URL: base + "/dl/checksums.txt"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": f.tag, "html_url": "https://example.test/r", "body": "notes", "published_at": "2026-01-01T00:00:00Z", "prerelease": f.pre, "assets": assets})
	})
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		if f.sums != "" {
			_, _ = w.Write([]byte(f.sums))
			return
		}
		sum := sha256.Sum256(f.archive)
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  dockduck_" + strings.TrimPrefix(f.tag, "v") + "_linux_amd64.tar.gz\n"))
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(f.archive) })
	f.srv = httptest.NewServer(mux)
	base = f.srv.URL
	t.Cleanup(f.srv.Close)
	return f
}

func setup(t *testing.T, f *fake, version string) (*Updater, string) {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "dockduck")
	if err := os.WriteFile(exe, []byte("OLD-BINARY"), 0o750); err != nil {
		t.Fatal(err)
	}
	return &Updater{Version: version, API: f.srv.URL, Exe: exe, OS: "linux", Arch: "amd64"}, exe
}

func TestCheck(t *testing.T) {
	f := newFake(t, tarGz(t, map[string][]byte{"dockduck": []byte("NEW")}, nil))
	u, _ := setup(t, f, "v0.1.0")
	st, err := u.Check(context.Background())
	if err != nil || !st.Available || st.Latest != "v0.2.0" || !st.CanInstall || st.Notes != "notes" {
		t.Fatalf("%+v %v", st, err)
	}
	u.Version = "v0.2.0"
	if st, _ := u.Check(context.Background()); st.Available {
		t.Errorf("same version must not be offered: %+v", st)
	}
	u.Version = "dev"
	st, _ = u.Check(context.Background())
	if st.Available || st.CanInstall || st.Reason == "" {
		t.Errorf("a dev build can neither compare nor install: %+v", st)
	}
	f.pre = true
	u.Version = "v0.1.0"
	if _, err := u.Check(context.Background()); err == nil {
		t.Error("a prerelease must not count as the latest release")
	}
}

func TestInstallReplacesTheBinaryAtomically(t *testing.T) {
	f := newFake(t, tarGz(t, map[string][]byte{"dockduck": []byte("NEW-BINARY"), "README.md": []byte("x")}, nil))
	u, exe := setup(t, f, "v0.1.0")
	tag, err := u.Install(context.Background())
	if err != nil || tag != "v0.2.0" {
		t.Fatalf("%q %v", tag, err)
	}
	b, _ := os.ReadFile(exe)
	if string(b) != "NEW-BINARY" {
		t.Errorf("content %q", b)
	}
	if st, _ := os.Stat(exe); st.Mode().Perm()&0o111 == 0 {
		t.Errorf("must stay executable: %v", st.Mode())
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), ".dockduck-*"))
	if len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestInstallRefusesBadDownloads(t *testing.T) {
	good := tarGz(t, map[string][]byte{"dockduck": []byte("NEW")}, nil)
	t.Run("wrong checksum", func(t *testing.T) {
		f := newFake(t, good)
		f.sums = strings.Repeat("0", 64) + "  dockduck_0.2.0_linux_amd64.tar.gz\n"
		u, exe := setup(t, f, "v0.1.0")
		if _, err := u.Install(context.Background()); err == nil || !strings.Contains(err.Error(), "checksum") {
			t.Fatalf("%v", err)
		}
		if b, _ := os.ReadFile(exe); string(b) != "OLD-BINARY" {
			t.Error("the old binary must be untouched after a failed update")
		}
	})
	t.Run("no checksums file", func(t *testing.T) {
		f := newFake(t, good)
		f.noSums = true
		u, exe := setup(t, f, "v0.1.0")
		if _, err := u.Install(context.Background()); err == nil {
			t.Fatal("an unverifiable download must be refused")
		}
		if b, _ := os.ReadFile(exe); string(b) != "OLD-BINARY" {
			t.Error("untouched")
		}
	})
	t.Run("checksum line for another file", func(t *testing.T) {
		f := newFake(t, good)
		f.sums = strings.Repeat("a", 64) + "  something_else.tar.gz\n"
		u, _ := setup(t, f, "v0.1.0")
		if _, err := u.Install(context.Background()); err == nil {
			t.Fatal("no entry for our archive")
		}
	})
	t.Run("no build for this system", func(t *testing.T) {
		f := newFake(t, good)
		u, _ := setup(t, f, "v0.1.0")
		u.OS, u.Arch = "darwin", "arm64"
		if _, err := u.Install(context.Background()); err == nil || !strings.Contains(err.Error(), "darwin/arm64") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("already up to date", func(t *testing.T) {
		f := newFake(t, good)
		u, _ := setup(t, f, "v0.2.0")
		if _, err := u.Install(context.Background()); err == nil {
			t.Fatal("nothing to install")
		}
	})
	t.Run("dev build", func(t *testing.T) {
		f := newFake(t, good)
		u, _ := setup(t, f, "dev")
		if _, err := u.Install(context.Background()); err == nil {
			t.Fatal("a dev build must not replace itself")
		}
	})
}

func TestExtractIgnoresEverythingButTheBinary(t *testing.T) {
	// the entry is a symlink, or lives in a folder, or is named like a path trick
	for name, files := range map[string]map[string][]byte{
		"path trick":    {"../dockduck": []byte("EVIL")},
		"nested":        {"bin/dockduck": []byte("EVIL")},
		"absolute":      {"/usr/bin/dockduck": []byte("EVIL")},
		"only a readme": {"README.md": []byte("x")},
	} {
		if _, err := extract(tarGz(t, files, nil)); err == nil {
			t.Errorf("%s: must not be accepted as the binary", name)
		}
	}
	if _, err := extract(tarGz(t, map[string][]byte{"dockduck": nil}, map[string]byte{"dockduck": tar.TypeSymlink})); err == nil {
		t.Error("a symlink named dockduck must not be accepted")
	}
	if _, err := extract([]byte("not a gzip")); err == nil {
		t.Error("garbage")
	}
}

func TestUnwritableFolder(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere")
	}
	f := newFake(t, tarGz(t, map[string][]byte{"dockduck": []byte("NEW")}, nil))
	u, exe := setup(t, f, "v0.1.0")
	if err := os.Chmod(filepath.Dir(exe), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(exe), 0o755) })
	st, _ := u.Check(context.Background())
	if st.CanInstall || !strings.Contains(st.Reason, "sudo") {
		t.Errorf("should explain how to fix it: %+v", st)
	}
	if _, err := u.Install(context.Background()); err == nil {
		t.Error("must fail, not panic")
	}
}

func TestGitHubErrors(t *testing.T) {
	for code, want := range map[int]string{404: "no release", 403: "rate limit", 500: "500"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }))
		u := &Updater{Version: "v0.1.0", API: srv.URL, Exe: filepath.Join(t.TempDir(), "x")}
		_, err := u.Latest(context.Background())
		srv.Close()
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Errorf("%d: %v", code, err)
		}
	}
}
