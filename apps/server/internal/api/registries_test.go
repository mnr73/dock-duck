package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
	"dockduck/internal/registries"
)

type regDocker struct {
	args   []string
	stdin  string
	stderr string
	err    error
}

func (f *regDocker) OutputStdin(_ context.Context, stdin string, args ...string) ([]byte, []byte, error) {
	f.args, f.stdin = args, stdin
	if f.err != nil {
		return nil, []byte(f.stderr), &dockercli.ExitError{Args: args, Stderr: f.stderr, Err: f.err}
	}
	return nil, []byte(f.stderr), nil
}

func newRegEnv(t *testing.T, fd *regDocker, cfg string) http.Handler {
	t.Helper()
	dir := t.TempDir()
	if cfg != "" {
		_ = os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfg), 0o600)
	}
	st, _ := config.Open(filepath.Join(t.TempDir(), "c.json"))
	d := dockercli.RunnerFunc(func(context.Context, ...string) ([]byte, error) { return []byte("[]"), nil })
	return New(Deps{UI: http.NotFoundHandler(), Store: st,
		Contexts:   &contexts.Service{Docker: fakeDocker{d}, Store: st},
		Projects:   &projects.Service{Docker: d, Store: st},
		Registries: &registries.Service{Docker: fd, ConfigDir: dir}})
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	r := localReq("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestRegistryStateNeverContainsSecrets(t *testing.T) {
	cfg := `{"auths":{"https://index.docker.io/v1/":{"auth":"` + base64.StdEncoding.EncodeToString([]byte("sinadev:HUB-SECRET-77")) + `"},"ghcr.io":{"auth":"` + base64.StdEncoding.EncodeToString([]byte("sina-k:GH-SECRET-88")) + `"}}}`
	h := newRegEnv(t, &regDocker{}, cfg)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, localReq("GET", "/api/v1/registries", nil))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	for _, s := range []string{"HUB-SECRET-77", "GH-SECRET-88"} {
		if strings.Contains(rec.Body.String(), s) {
			t.Fatalf("secret in the response: %s", rec.Body)
		}
	}
	var st registries.State
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if len(st.Entries) != 2 || st.Entries[0].Username != "sinadev" || !st.Plaintext {
		t.Errorf("%+v", st)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("no-store")
	}
}

func TestLoginSecretNeverReachesLogsOrResponses(t *testing.T) {
	const secret = "ghp_SUPER-SECRET-TOKEN-VALUE"
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(old) })

	// success
	fd := &regDocker{stderr: "Login Succeeded\n"}
	h := newRegEnv(t, fd, "")
	rec := post(h, "/api/v1/registries/login", `{"server":"ghcr.io","username":"sina-k","password":"`+secret+`"}`)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if fd.stdin != secret+"\n" || strings.Contains(strings.Join(fd.args, " "), secret) {
		t.Errorf("secret must be on stdin only: stdin=%q argv=%q", fd.stdin, fd.args)
	}
	// failure where the registry echoes the secret back
	fd2 := &regDocker{err: errors.New("exit 1"), stderr: "Error response from daemon: login attempt to https://ghcr.io/v2/ failed: unauthorized: bad token " + secret + "\n"}
	h2 := newRegEnv(t, fd2, "")
	rec2 := post(h2, "/api/v1/registries/login", `{"server":"ghcr.io","username":"sina-k","password":"`+secret+`"}`)
	if rec2.Code != http.StatusUnprocessableEntity || !strings.Contains(rec2.Body.String(), "unauthorized") {
		t.Errorf("%d %s", rec2.Code, rec2.Body)
	}
	// invalid input
	rec3 := post(h, "/api/v1/registries/login", `{"server":"-v","username":"u","password":"`+secret+`"}`)
	if rec3.Code != http.StatusBadRequest {
		t.Errorf("%d", rec3.Code)
	}
	rec4 := post(h, "/api/v1/registries/login", `not json `+secret)
	if rec4.Code != http.StatusBadRequest {
		t.Errorf("%d", rec4.Code)
	}
	for name, body := range map[string]string{"success": rec.Body.String(), "echoing registry": rec2.Body.String(), "bad input": rec3.Body.String(), "bad json": rec4.Body.String(), "logs": logs.String()} {
		if strings.Contains(body, secret) {
			t.Errorf("%s contains the secret:\n%s", name, body)
		}
	}
	if !strings.Contains(logs.String(), "registry login") {
		t.Errorf("the action itself is audited: %s", logs.String())
	}
}

func TestLogoutEndpoint(t *testing.T) {
	fd := &regDocker{}
	h := newRegEnv(t, fd, "")
	if rec := post(h, "/api/v1/registries/logout", `{"server":"ghcr.io"}`); rec.Code != http.StatusNoContent || strings.Join(fd.args, " ") != "logout ghcr.io" {
		t.Errorf("%d %q", rec.Code, fd.args)
	}
	if rec := post(h, "/api/v1/registries/logout", `{"server":"--all"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("flag injection: %d", rec.Code)
	}
	// cross-origin logout (a foreign page making you sign out of registries) is refused by the guard
	r := localReq("POST", "/api/v1/registries/logout", strings.NewReader(`{"server":"ghcr.io"}`))
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-origin: %d", rec.Code)
	}
}
