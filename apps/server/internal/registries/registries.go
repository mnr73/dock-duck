// Package registries shows which Docker registries the CLI is signed in to and
// signs in/out through `docker login` / `docker logout`.
//
// Secrets: the password goes to docker over stdin only (never argv, where `ps`
// shows it), is never logged and is never returned. Reading the state returns
// names and usernames; the stored secret is never read, not even to mask it.
package registries

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"dockduck/internal/files"
)

// Hub is the canonical Docker Hub key in config.json.
const Hub = "https://index.docker.io/v1/"

// Entry is one registry the CLI holds credentials for.
type Entry struct {
	Server   string `json:"server"` // as stored: "https://index.docker.io/v1/", "ghcr.io", ...
	Name     string `json:"name"`   // friendly name
	Hub      bool   `json:"hub"`
	Username string `json:"username,omitempty"` // empty when unknown (helper without `list`, identity token)
	Store    string `json:"store"`              // "file" (plain text in config.json) or the helper name
}

// State is what the UI shows.
type State struct {
	ConfigPath string  `json:"config_path"`
	Store      string  `json:"store"` // default store for new logins: "file" or a helper
	Plaintext  bool    `json:"plaintext"`
	Entries    []Entry `json:"entries"`
}

var (
	ErrInvalid = errors.New("invalid input")
	// ErrFailed wraps docker's own reason (wrong password, unreachable, ...); use
	// Reason to show it without the wrapper.
	ErrFailed = errors.New("docker refused")
)

// Service reads and changes the CLI's registry credentials.
type Service struct {
	Docker files.StdinRunner
	// ConfigDir overrides $DOCKER_CONFIG / ~/.docker (tests).
	ConfigDir string
	// Helper runs `docker-credential-<name> list`; nil uses the real binary.
	Helper func(ctx context.Context, name string) (map[string]string, error)
}

func (s *Service) dir() string {
	if s.ConfigDir != "" {
		return s.ConfigDir
	}
	if d := os.Getenv("DOCKER_CONFIG"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker")
}

type dockerConfig struct {
	Auths map[string]struct {
		Auth          string `json:"auth"`
		IdentityToken string `json:"identitytoken"`
		Username      string `json:"username"`
	} `json:"auths"`
	CredsStore  string            `json:"credsStore"`
	CredHelpers map[string]string `json:"credHelpers"`
}

var helperName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,40}$`)

// Read returns the registries the CLI is signed in to.
func (s *Service) Read(ctx context.Context) (State, error) {
	path := filepath.Join(s.dir(), "config.json")
	st := State{ConfigPath: path, Store: "file", Entries: []Entry{}}
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}
	var c dockerConfig
	if len(b) > 0 {
		if err := json.Unmarshal(b, &c); err != nil {
			return st, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if c.CredsStore != "" {
		st.Store = c.CredsStore
	}
	st.Plaintext = st.Store == "file"

	byServer := map[string]*Entry{}
	add := func(server, store string) *Entry {
		key := canonical(server)
		if e, ok := byServer[key]; ok {
			return e
		}
		e := &Entry{Server: key, Name: friendly(key), Hub: key == Hub, Store: store}
		byServer[key] = e
		return e
	}
	for server, a := range c.Auths {
		if isHubSubkey(server) {
			continue // Docker Hub's access-/refresh-token slots belong to the Hub entry
		}
		e := add(server, "file")
		switch {
		case a.Username != "":
			e.Username = a.Username
		case a.Auth != "":
			e.Username = usernameOf(a.Auth) // the part before ':' only; the secret is dropped here
		}
		if a.Auth == "" && a.IdentityToken == "" && c.CredsStore != "" {
			e.Store = c.CredsStore // an empty `auths` slot means "ask the helper"
		}
	}
	for _, store := range append([]string{c.CredsStore}, helpers(c.CredHelpers)...) {
		if store == "" || !helperName.MatchString(store) {
			continue
		}
		list, err := s.list(ctx, store)
		if err != nil {
			continue // helper missing or locked: the entries from config.json still show
		}
		for server, user := range list {
			if isHubSubkey(server) {
				continue
			}
			e := add(server, store)
			e.Store = store
			if e.Username == "" {
				e.Username = user
			}
		}
	}
	for server, h := range c.CredHelpers {
		if helperName.MatchString(h) {
			add(server, h).Store = h
		}
	}
	for _, e := range byServer {
		st.Entries = append(st.Entries, *e)
	}
	sort.Slice(st.Entries, func(i, j int) bool {
		if st.Entries[i].Hub != st.Entries[j].Hub {
			return st.Entries[i].Hub
		}
		return st.Entries[i].Server < st.Entries[j].Server
	})
	return st, nil
}

func helpers(m map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range m {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Service) list(ctx context.Context, name string) (map[string]string, error) {
	if s.Helper != nil {
		return s.Helper(ctx, name)
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	// `list` returns {server: username} and never a secret
	out, err := exec.CommandContext(ctx, "docker-credential-"+name, "list").Output()
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func usernameOf(authB64 string) string {
	raw, err := base64.StdEncoding.DecodeString(authB64)
	if err != nil {
		return ""
	}
	user, _, ok := strings.Cut(string(raw), ":")
	if !ok {
		return ""
	}
	return user
}

func isHubSubkey(server string) bool {
	return strings.HasPrefix(server, Hub) && server != Hub
}

// canonical maps every spelling of Docker Hub to Hub, and strips a scheme and
// trailing slash from the rest, so one registry is one row.
func canonical(server string) string {
	s := strings.TrimSpace(server)
	l := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://"), "/"))
	switch l {
	case "", "docker.io", "index.docker.io", "index.docker.io/v1", "registry-1.docker.io", "hub.docker.com":
		return Hub
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return strings.TrimSuffix(s, "/")
	}
	return s
}

func friendly(server string) string {
	switch {
	case server == Hub:
		return "Docker Hub"
	case server == "ghcr.io":
		return "GitHub Container Registry"
	case server == "quay.io":
		return "Quay"
	case server == "registry.gitlab.com" || strings.HasPrefix(server, "registry.gitlab."):
		return "GitLab Container Registry"
	case strings.HasSuffix(server, ".azurecr.io"):
		return "Azure Container Registry"
	case strings.HasSuffix(server, ".pkg.dev") || server == "gcr.io" || strings.HasSuffix(server, ".gcr.io"):
		return "Google Artifact Registry"
	case strings.Contains(server, ".dkr.ecr.") && strings.HasSuffix(server, ".amazonaws.com"):
		return "Amazon ECR"
	}
	return strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://")
}

// ---- input validation ----------------------------------------------------

// A registry host[:port][/path], no scheme, no spaces, never starting with '-'
// (it would become a docker flag).
var serverRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?(?::[0-9]{1,5})?(?:/[A-Za-z0-9._~/-]*)?$`)

// ValidateServer accepts "" (Docker Hub) or host[:port][/path], with an optional
// http(s):// prefix that is dropped (docker takes a bare host).
func ValidateServer(server string) (string, error) {
	s := strings.TrimSpace(server)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	s = strings.TrimSuffix(s, "/")
	if canonical(s) == Hub {
		return "", nil
	}
	if len(s) > 255 || !serverRe.MatchString(s) {
		return "", fmt.Errorf("%w: enter a registry address like ghcr.io or registry.example.com:5000", ErrInvalid)
	}
	return s, nil
}

func validateUser(u string) error {
	if u == "" || len(u) > 255 || strings.HasPrefix(u, "-") || strings.ContainsAny(u, "\r\n\x00") {
		return fmt.Errorf("%w: username is empty or has invalid characters", ErrInvalid)
	}
	return nil
}

func validateSecret(p string) error {
	if p == "" {
		return fmt.Errorf("%w: password or token is empty", ErrInvalid)
	}
	// a newline would end the stdin line early: docker would sign in with a truncated secret
	if len(p) > 8192 || strings.ContainsAny(p, "\r\n\x00") {
		return fmt.Errorf("%w: password or token has invalid characters (no line breaks)", ErrInvalid)
	}
	return nil
}

// Result of a successful login.
type Result struct {
	Server    string `json:"server"`
	Plaintext bool   `json:"plaintext"` // docker stored the secret unencrypted in config.json
}

// Login signs in with `docker login --username U --password-stdin [SERVER]`.
func (s *Service) Login(ctx context.Context, server, username, password string) (Result, error) {
	host, err := ValidateServer(server)
	if err != nil {
		return Result{}, err
	}
	if err := validateUser(username); err != nil {
		return Result{}, err
	}
	if err := validateSecret(password); err != nil {
		return Result{}, err
	}
	args := []string{"login", "--username", username, "--password-stdin"}
	if host != "" {
		args = append(args, host)
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, stderr, err := s.Docker.OutputStdin(ctx, password+"\n", args...)
	msg := scrub(string(stderr), password)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %s", ErrFailed, reason(msg, err))
	}
	if host == "" {
		host = Hub
	}
	return Result{Server: host, Plaintext: strings.Contains(strings.ToLower(msg), "unencrypted")}, nil
}

// Logout removes the stored credentials of one registry.
func (s *Service) Logout(ctx context.Context, server string) error {
	host, err := ValidateServer(server)
	if err != nil {
		return err
	}
	args := []string{"logout"}
	if host != "" {
		args = append(args, host)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, stderr, err := s.Docker.OutputStdin(ctx, "", args...)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailed, reason(scrub(string(stderr), ""), err))
	}
	return nil
}

// scrub removes the secret from any text that could be shown or logged, in case
// a registry echoes it back in an error body.
func scrub(s, secret string) string {
	if len(secret) >= 4 {
		s = strings.ReplaceAll(s, secret, "••••")
	}
	return s
}

// reason picks the line of docker's output that explains the failure. When there
// is none it returns a fixed message: err.Error() of an ExitError is the whole
// command line plus raw stderr, which must never reach the browser.
func reason(stderr string, _ error) string {
	for _, line := range strings.Split(stderr, "\n") {
		l := strings.TrimSpace(line)
		l = strings.TrimPrefix(l, "Error response from daemon: ")
		l = strings.TrimPrefix(l, "Error: ")
		if l == "" || strings.HasPrefix(l, "WARNING!") || strings.HasPrefix(l, "Configure a credential helper") || strings.HasPrefix(l, "https://docs.docker.com") || strings.HasPrefix(l, "See ") {
			continue
		}
		if len(l) > 300 {
			l = l[:300] + "…"
		}
		return l
	}
	return "docker did not say why; check the address and the credentials"
}

// Reason is the message to show a user for an error from Login/Logout: docker's own
// words for ErrFailed, the validation text for ErrInvalid.
func Reason(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, ErrFailed.Error()+": ")
	msg = strings.TrimPrefix(msg, ErrInvalid.Error()+": ")
	return msg
}
