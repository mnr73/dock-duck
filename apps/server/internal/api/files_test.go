package api

import (
	"context"
	"encoding/json"
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
)

type stdinFake struct{ dockercli.Runner }

func (stdinFake) OutputStdin(_ context.Context, stdin string, _ ...string) ([]byte, []byte, error) {
	if strings.Contains(stdin, "BROKEN") {
		return nil, []byte("yaml: line 3: did not find expected key\n"), &dockercli.ExitError{Stderr: "yaml: line 3: did not find expected key\n", Err: os.ErrInvalid}
	}
	return nil, nil, nil
}

type fileEnv struct {
	h    http.Handler
	dir  string
	pid  string
	proj string
}

func newFileEnv(t *testing.T) *fileEnv {
	t.Helper()
	root := t.TempDir()
	proj := filepath.Join(root, "shop")
	_ = os.MkdirAll(proj, 0o755)
	_ = os.WriteFile(filepath.Join(proj, "compose.yaml"), []byte("services:\n  app:\n    image: x\n    env_file: .env\n"), 0o644)
	_ = os.WriteFile(filepath.Join(proj, ".env"), []byte("APP_NAME=Shop\nDB_PASSWORD=hunter2\nAPP_KEY=base64:abc\nEMPTY_TOKEN=\n"), 0o600)
	_ = os.WriteFile(filepath.Join(proj, ".env.example"), []byte("APP_NAME=x\nDB_PASSWORD=\nMAIL_HOST=smtp\nSENTRY_DSN=https://real@sentry/1\n"), 0o644)

	st, _ := config.Open(filepath.Join(t.TempDir(), "c.json"))
	_ = st.Update(func(c *config.Config) error { c.Roots = []config.Root{{ID: "r", Path: root}}; return nil })
	d := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		j := strings.Join(a, " ")
		switch {
		case strings.Contains(j, "context ls"):
			return []byte(`{"Name":"default","Current":true,"DockerEndpoint":"unix:///x"}`), nil
		case strings.Contains(j, "compose ls"):
			return []byte("[]"), nil
		case strings.Contains(j, "--no-interpolate"):
			return []byte(`{"services":{"app":{"env_file":[{"path":"/x/.env"}],"image":"${IMG}","command":"$PORT"}}}`), nil
		}
		return []byte("[]"), nil
	})
	h := New(Deps{UI: http.NotFoundHandler(), Store: st,
		Contexts: &contexts.Service{Docker: fakeDocker{d}, Store: st},
		Projects: &projects.Service{Docker: d, Store: st},
		Jobs:     nil, Stdin: stdinFake{d}})
	e := &fileEnv{h: h, dir: root, proj: proj}
	ps, _ := (&projects.Service{Docker: d, Store: st}).List(context.Background(), "")
	e.pid = ps[0].ID
	return e
}

func (e *fileEnv) do(method, path, body string, hdr ...string) *httptest.ResponseRecorder {
	req := localReq(method, path, strings.NewReader(body))
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func TestFilesListAndRead(t *testing.T) {
	e := newFileEnv(t)
	rec := e.do("GET", "/api/v1/projects/"+e.pid+"/files", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"id":"compose-0"`) || !strings.Contains(rec.Body.String(), `"id":"env"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	rec = e.do("GET", "/api/v1/projects/"+e.pid+"/files/compose-0", "")
	var c struct{ Content, ETag string }
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if rec.Code != 200 || !strings.Contains(c.Content, "env_file") || c.ETag == "" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	// ids that are not server-derived never resolve to a file
	for _, bad := range []string{"compose-9", "compose--1", "ENV", "env.bak", "%2e%2e%2fpasswd", "..%2f..%2fetc%2fpasswd"} {
		if rec := e.do("GET", "/api/v1/projects/"+e.pid+"/files/"+bad, ""); rec.Code != 404 {
			t.Errorf("file id %q must be a 404, got %d %s", bad, rec.Code, rec.Body)
		}
	}
	// literal `..` segments are redirected by the router to the cleaned path;
	// following that redirect must also end in a 404, never in file content
	for _, bad := range []string{"../../etc/passwd", "env/../x"} {
		rec := e.do("GET", "/api/v1/projects/"+e.pid+"/files/"+bad, "")
		if rec.Code != 307 && rec.Code != 404 {
			t.Errorf("%q: %d", bad, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "root:") {
			t.Errorf("%q leaked content", bad)
		}
	}
}

func TestEnvIsMaskedUnlessRevealed(t *testing.T) {
	e := newFileEnv(t)
	rec := e.do("GET", "/api/v1/projects/"+e.pid+"/files/env", "")
	body := rec.Body.String()
	if strings.Contains(body, "hunter2") || strings.Contains(body, "base64:abc") {
		t.Fatalf("secret values leaked in the masked raw text: %s", body)
	}
	if !strings.Contains(body, "APP_NAME=Shop") || rec.Header().Get("X-Content-Masked") != "1" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("non-secrets stay readable, response is not cacheable: %s %v", body, rec.Header())
	}
	rec = e.do("GET", "/api/v1/projects/"+e.pid+"/files/env?reveal=1", "")
	if !strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("reveal must return the real text: %s", rec.Body)
	}

	rec = e.do("GET", "/api/v1/projects/"+e.pid+"/env", "")
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("structured view leaked a secret: %s", rec.Body)
	}
	var v struct {
		Lines []struct {
			Key    string
			Value  string
			Masked bool
			Set    bool
		}
		Example struct {
			Missing []struct{ Key, Value string }
		}
		Usage struct {
			Services     []string
			Interpolated []string
		}
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	if len(v.Lines) != 4 || !v.Lines[1].Masked || !v.Lines[1].Set || v.Lines[3].Masked {
		t.Errorf("lines: %+v", v.Lines)
	}
	missing := map[string]string{}
	for _, m := range v.Example.Missing {
		missing[m.Key] = m.Value
	}
	if len(missing) != 2 || missing["MAIL_HOST"] != "smtp" || missing["SENTRY_DSN"] != "" {
		t.Errorf("example keys missing from .env (secret example values must not be offered): %+v", missing)
	}
	if len(v.Usage.Services) != 1 || v.Usage.Services[0] != "app" || len(v.Usage.Interpolated) != 2 {
		t.Errorf("usage: %+v", v.Usage)
	}
	if rec := e.do("GET", "/api/v1/projects/"+e.pid+"/env?reveal=1", ""); !strings.Contains(rec.Body.String(), "hunter2") {
		t.Error("structured reveal")
	}
}

func TestSaveNeedsEtagAndDetectsConflicts(t *testing.T) {
	e := newFileEnv(t)
	url := "/api/v1/projects/" + e.pid + "/files/compose-0"
	var c struct{ ETag string }
	_ = json.Unmarshal(e.do("GET", url, "").Body.Bytes(), &c)

	if rec := e.do("PUT", url, `{"content":"services: {}\n"}`); rec.Code != 409 {
		t.Errorf("no If-Match must not overwrite: %d", rec.Code)
	}
	if rec := e.do("PUT", url, `{"content":"services: {}\n"}`, "If-Match", `"stale"`); rec.Code != 409 {
		t.Errorf("stale etag: %d", rec.Code)
	}
	rec := e.do("PUT", url, `{"content":"services:\n  web:\n    image: nginx\n"}`, "If-Match", c.ETag)
	if rec.Code != 200 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	got, _ := os.ReadFile(filepath.Join(e.proj, "compose.yaml"))
	if !strings.Contains(string(got), "nginx") {
		t.Errorf("not written: %s", got)
	}
	if bak, _ := os.ReadFile(filepath.Join(e.proj, "compose.yaml.bak")); !strings.Contains(string(bak), "env_file") {
		t.Errorf(".bak must hold the previous version: %s", bak)
	}
	if rec := e.do("PUT", url, `not json`, "If-Match", `"x"`); rec.Code != 400 {
		t.Errorf("bad json: %d", rec.Code)
	}
}

func TestMaskedPlaceholdersAreNeverSaved(t *testing.T) {
	e := newFileEnv(t)
	url := "/api/v1/projects/" + e.pid + "/files/env"
	var c struct{ ETag string }
	_ = json.Unmarshal(e.do("GET", url+"?reveal=1", "").Body.Bytes(), &c)
	masked := "APP_NAME=Shop\nDB_PASSWORD=••••••••\n"
	if rec := e.do("PUT", url, mustJSON(map[string]string{"content": masked}), "If-Match", c.ETag); rec.Code != 400 {
		t.Fatalf("saving masked text must be refused (it would destroy the secret): %d", rec.Code)
	}
	if got, _ := os.ReadFile(filepath.Join(e.proj, ".env")); !strings.Contains(string(got), "hunter2") {
		t.Fatal("the secret was overwritten")
	}
	rec := e.do("PUT", url, mustJSON(map[string]string{"content": "APP_NAME=Shop\nDB_PASSWORD=new\n"}), "If-Match", c.ETag)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	st, _ := os.Stat(filepath.Join(e.proj, ".env"))
	if st.Mode().Perm() != 0o600 {
		t.Errorf(".env mode changed to %v", st.Mode().Perm())
	}
}

func TestCreateEnvWhenMissing(t *testing.T) {
	e := newFileEnv(t)
	_ = os.Remove(filepath.Join(e.proj, ".env"))
	url := "/api/v1/projects/" + e.pid + "/files/env"
	rec := e.do("GET", url, "")
	var c struct {
		Exists  bool
		Content string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if rec.Code != 200 || c.Exists || c.Content != "" {
		t.Fatalf("missing .env is an empty editable file: %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("PUT", url, `{"content":"A=1\n"}`, "If-Match", ""); rec.Code != 200 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	st, err := os.Stat(filepath.Join(e.proj, ".env"))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("created .env must be 0600: %v %v", st, err)
	}
}

func TestValidateEndpoint(t *testing.T) {
	e := newFileEnv(t)
	url := "/api/v1/projects/" + e.pid + "/files/compose-0/validate"
	rec := e.do("POST", url, `{"content":"services: {}\n"}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
	rec = e.do("POST", url, `{"content":"BROKEN"}`)
	if !strings.Contains(rec.Body.String(), `"ok":false`) || !strings.Contains(rec.Body.String(), `"line":3`) {
		t.Errorf("error with a line number: %s", rec.Body)
	}
	if rec := e.do("POST", "/api/v1/projects/"+e.pid+"/files/env/validate", `{"content":"A=1"}`); rec.Code != 400 {
		t.Errorf("env files are not validated by compose: %d", rec.Code)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
