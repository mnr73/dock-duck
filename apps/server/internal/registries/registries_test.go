package registries

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"dockduck/internal/dockercli"
)

type fakeDocker struct {
	args   []string
	stdin  string
	stderr string
	err    error
}

func (f *fakeDocker) OutputStdin(_ context.Context, stdin string, args ...string) ([]byte, []byte, error) {
	f.args, f.stdin = args, stdin
	if f.err != nil {
		return nil, []byte(f.stderr), &dockercli.ExitError{Args: args, Stderr: f.stderr, Err: f.err}
	}
	return nil, []byte(f.stderr), nil
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func writeCfg(t *testing.T, v any) string {
	t.Helper()
	dir := t.TempDir()
	b, _ := json.Marshal(v)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The shape of a real ~/.docker/config.json: plain `auths`, Hub with its two token slots.
func TestReadRealWorldFileConfig(t *testing.T) {
	dir := writeCfg(t, map[string]any{"auths": map[string]any{
		"https://index.docker.io/v1/":              map[string]any{"auth": b64("sinadev:SECRET-PASSWORD-1")},
		"https://index.docker.io/v1/access-token":  map[string]any{"auth": b64("x:ACCESS-TOKEN-2")},
		"https://index.docker.io/v1/refresh-token": map[string]any{"auth": b64("x:REFRESH-TOKEN-3")},
		"ghcr.io":                     map[string]any{"auth": b64("sina-k:ghp_SECRETTOKEN4")},
		"registry.gitlab.company.dev": map[string]any{"auth": b64("sina:glpat-SECRET5")},
	}})
	s := &Service{ConfigDir: dir}
	st, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range st.Entries {
		names = append(names, e.Server+"="+e.Username)
	}
	want := []string{Hub + "=sinadev", "ghcr.io=sina-k", "registry.gitlab.company.dev=sina"}
	if !slices.Equal(names, want) {
		t.Errorf("entries (Hub first, its token slots folded away):\n got  %v\n want %v", names, want)
	}
	if !st.Plaintext || st.Store != "file" {
		t.Errorf("secrets sit in the file unencrypted, the UI must say so: %+v", st)
	}
	out, _ := json.Marshal(st)
	for _, secret := range []string{"SECRET-PASSWORD-1", "ACCESS-TOKEN-2", "REFRESH-TOKEN-3", "ghp_SECRETTOKEN4", "glpat-SECRET5", b64("sinadev:SECRET-PASSWORD-1")} {
		if strings.Contains(string(out), secret) {
			t.Fatalf("the state returned to the browser contains a secret: %s", secret)
		}
	}
	if !st.Entries[0].Hub || st.Entries[0].Name != "Docker Hub" || st.Entries[1].Name != "GitHub Container Registry" {
		t.Errorf("names: %+v", st.Entries)
	}
}

func TestReadMissingAndBrokenConfig(t *testing.T) {
	st, err := (&Service{ConfigDir: t.TempDir()}).Read(context.Background())
	if err != nil || len(st.Entries) != 0 || st.Entries == nil {
		t.Errorf("no config file = signed out of everything, not an error: %+v %v", st, err)
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o600)
	if _, err := (&Service{ConfigDir: dir}).Read(context.Background()); err == nil {
		t.Error("a corrupt config must be reported, not hidden")
	}
	// auth blobs that are not base64 / not user:pass must not crash or leak
	dir = writeCfg(t, map[string]any{"auths": map[string]any{"a.io": map[string]any{"auth": "%%%"}, "b.io": map[string]any{"auth": b64("nocolon")}, "c.io": map[string]any{"identitytoken": "IDTOKEN-SECRET"}}})
	st, err = (&Service{ConfigDir: dir}).Read(context.Background())
	if err != nil || len(st.Entries) != 3 {
		t.Fatalf("%+v %v", st, err)
	}
	out, _ := json.Marshal(st)
	if strings.Contains(string(out), "IDTOKEN-SECRET") || strings.Contains(string(out), "nocolon") {
		t.Errorf("leak: %s", out)
	}
}

func TestReadWithCredentialHelper(t *testing.T) {
	dir := writeCfg(t, map[string]any{
		"credsStore":  "secretservice",
		"credHelpers": map[string]string{"123.dkr.ecr.eu-west-1.amazonaws.com": "ecr-login"},
		"auths":       map[string]any{"ghcr.io": map[string]any{}},
	})
	called := []string{}
	s := &Service{ConfigDir: dir, Helper: func(_ context.Context, name string) (map[string]string, error) {
		called = append(called, name)
		if name == "secretservice" {
			return map[string]string{"https://index.docker.io/v1/": "sinadev", "ghcr.io": "sina-k", "https://index.docker.io/v1/access-token": "x"}, nil
		}
		return nil, errors.New("helper not installed")
	}}
	st, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Store != "secretservice" || st.Plaintext {
		t.Errorf("a helper means the secrets are not in the file: %+v", st)
	}
	got := map[string]string{}
	for _, e := range st.Entries {
		got[e.Server] = e.Username + "/" + e.Store
	}
	if got[Hub] != "sinadev/secretservice" || got["ghcr.io"] != "sina-k/secretservice" {
		t.Errorf("%v", got)
	}
	if _, ok := got["123.dkr.ecr.eu-west-1.amazonaws.com"]; !ok {
		t.Errorf("a registry with its own helper is listed even when the helper cannot be queried: %v", got)
	}
	if len(st.Entries) != 3 {
		t.Errorf("token slots of Hub are not separate rows: %v", got)
	}
	// helper names come from a file: only sane names are ever executed
	dir = writeCfg(t, map[string]any{"credsStore": "../../bin/evil; id"})
	called = nil
	s = &Service{ConfigDir: dir, Helper: func(_ context.Context, name string) (map[string]string, error) {
		called = append(called, name)
		return nil, nil
	}}
	_, _ = s.Read(context.Background())
	if len(called) != 0 {
		t.Errorf("a hostile helper name must never be run: %v", called)
	}
}

func TestCanonical(t *testing.T) {
	for in, want := range map[string]string{
		"": Hub, "docker.io": Hub, "index.docker.io": Hub, "https://index.docker.io/v1/": Hub, "registry-1.docker.io": Hub, "Docker.IO": Hub,
		"ghcr.io": "ghcr.io", "https://ghcr.io/": "https://ghcr.io", "localhost:5000": "localhost:5000",
	} {
		if got := canonical(in); got != want {
			t.Errorf("canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateServer(t *testing.T) {
	ok := map[string]string{
		"": "", "docker.io": "", "https://index.docker.io/v1/": "",
		"ghcr.io": "ghcr.io", "https://ghcr.io/": "ghcr.io", "registry.example.com:5000": "registry.example.com:5000",
		"localhost:5000": "localhost:5000", "192.168.1.5:5000": "192.168.1.5:5000", "reg.example.com/team": "reg.example.com/team",
	}
	for in, want := range ok {
		got, err := ValidateServer(in)
		if err != nil || got != want {
			t.Errorf("%q -> %q, %v (want %q)", in, got, err, want)
		}
	}
	for _, bad := range []string{"-v", "--config=/x", "a b", "ghcr.io;id", "$(id)", "a\nb", "ghcr.io:99999999", "..", ".evil", "evil.", "host/../x/ y", strings.Repeat("a", 300), "ftp://x.io", "x.io\x00"} {
		if _, err := ValidateServer(bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestLoginSendsTheSecretOnlyOnStdin(t *testing.T) {
	const secret = "s3cr3t-T0KEN-with spaces & $ymbols"
	f := &fakeDocker{stderr: "WARNING! Your credentials are stored unencrypted in '/home/u/.docker/config.json'.\nConfigure a credential helper to remove this warning. See\nhttps://docs.docker.com/go/credential-store/\n\nLogin Succeeded\n"}
	r, err := (&Service{Docker: f}).Login(context.Background(), "ghcr.io", "sina-k", secret)
	if err != nil {
		t.Fatal(err)
	}
	if f.stdin != secret+"\n" {
		t.Errorf("stdin %q", f.stdin)
	}
	want := []string{"login", "--username", "sina-k", "--password-stdin", "ghcr.io"}
	if !slices.Equal(f.args, want) {
		t.Errorf("argv %q, want %q", f.args, want)
	}
	for _, a := range f.args {
		if strings.Contains(a, "s3cr3t") {
			t.Errorf("the secret is in argv (visible to `ps`): %q", f.args)
		}
	}
	if r.Server != "ghcr.io" || !r.Plaintext {
		t.Errorf("result: %+v", r)
	}
	// Docker Hub: no server argument at all
	f2 := &fakeDocker{}
	r, _ = (&Service{Docker: f2}).Login(context.Background(), "", "sinadev", "pw-long-enough")
	if slices.Contains(f2.args, "") || f2.args[len(f2.args)-1] != "--password-stdin" || r.Server != Hub || r.Plaintext {
		t.Errorf("hub: %q %+v", f2.args, r)
	}
}

func TestLoginErrorsAreHelpfulAndNeverContainTheSecret(t *testing.T) {
	const secret = "my-very-secret-token-123"
	f := &fakeDocker{err: errors.New("exit status 1"), stderr: "Error response from daemon: Get \"https://registry.example.com/v2/\": unauthorized: HTTP Basic: Access denied. password " + secret + " rejected\n"}
	_, err := (&Service{Docker: f}).Login(context.Background(), "registry.example.com", "u", secret)
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("%v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("a registry that echoes the password back must not leak it: %v", err)
	}
	if !strings.Contains(err.Error(), "unauthorized") || !strings.Contains(err.Error(), "Access denied") {
		t.Errorf("docker's own reason must reach the user: %v", err)
	}
	if strings.Contains(err.Error(), "Error response from daemon") {
		t.Errorf("noise prefix: %v", err)
	}
	// only warnings on stderr + failure: still an error, with a sane message
	f = &fakeDocker{err: errors.New("exit status 1"), stderr: "WARNING! Using --password via the CLI is insecure.\n"}
	_, err = (&Service{Docker: f}).Login(context.Background(), "", "u-name-probe", "pw")
	if err == nil || strings.Contains(err.Error(), "docker login") || strings.Contains(err.Error(), "u-name-probe") || strings.Contains(err.Error(), "--password-stdin") {
		t.Errorf("the raw ExitError (command line, username) must not reach the user: %v", err)
	}
	if !strings.Contains(err.Error(), "did not say why") {
		t.Errorf("a fixed, helpful fallback: %v", err)
	}
}

func TestLoginRejectsBadInputBeforeRunningDocker(t *testing.T) {
	f := &fakeDocker{}
	s := &Service{Docker: f}
	cases := []struct{ server, user, pw string }{
		{"-v", "u", "p"}, {"ghcr.io", "", "p"}, {"ghcr.io", "-u", "p"}, {"ghcr.io", "a\nb", "p"}, {"ghcr.io", "u", ""},
		{"ghcr.io", "u", "line1\nline2"}, {"ghcr.io", "u", "a\rb"}, {"ghcr.io", "u", "a\x00b"}, {"ghcr.io", "u", strings.Repeat("a", 9000)},
		{"a b", "u", "p"},
	}
	for _, c := range cases {
		if _, err := s.Login(context.Background(), c.server, c.user, c.pw); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", c, err)
		}
	}
	if f.args != nil {
		t.Errorf("docker must not run for invalid input: %v", f.args)
	}
}

func TestLogout(t *testing.T) {
	f := &fakeDocker{}
	if err := (&Service{Docker: f}).Logout(context.Background(), "ghcr.io"); err != nil || !slices.Equal(f.args, []string{"logout", "ghcr.io"}) {
		t.Errorf("%v %q", err, f.args)
	}
	if err := (&Service{Docker: f}).Logout(context.Background(), Hub); err != nil || !slices.Equal(f.args, []string{"logout"}) {
		t.Errorf("hub logout takes no argument: %v %q", err, f.args)
	}
	if err := (&Service{Docker: f}).Logout(context.Background(), "--all"); !errors.Is(err, ErrInvalid) {
		t.Errorf("flag injection: %v", err)
	}
	f = &fakeDocker{err: errors.New("exit 1"), stderr: "Error: something broke\n"}
	if err := (&Service{Docker: f}).Logout(context.Background(), "ghcr.io"); !errors.Is(err, ErrFailed) || !strings.Contains(err.Error(), "something broke") {
		t.Errorf("%v", err)
	}
}

func TestReasonHasNoInternalPrefix(t *testing.T) {
	f := &fakeDocker{err: errors.New("exit 1"), stderr: "Error response from daemon: login attempt to http://x/v2/ failed with status: 401 Unauthorized\n"}
	_, err := (&Service{Docker: f}).Login(context.Background(), "localhost:15099", "u", "pw")
	if got := Reason(err); got != "login attempt to http://x/v2/ failed with status: 401 Unauthorized" {
		t.Errorf("%q", got)
	}
	_, err = (&Service{Docker: f}).Login(context.Background(), "-v", "u", "pw")
	if got := Reason(err); strings.HasPrefix(got, "invalid input") || got == "" {
		t.Errorf("%q", got)
	}
}
