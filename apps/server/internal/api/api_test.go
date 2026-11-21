package api

import (
	"context"
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

func TestHealthAndFallback(t *testing.T) {
	ui := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ui")) })
	h := New(Deps{Info: Info{Version: "t"}, UI: ui})

	cases := []struct {
		path, want string
		code       int
	}{
		{"/api/v1/health", `"ok":true`, 200},
		{"/api/v1/nope", `not_found`, 404},
		{"/projects/x", "ui", 200},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.want) {
			t.Errorf("%s: got %d %q", c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestRootsAndProjects(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "scan", "app")
	_ = os.MkdirAll(proj, 0o755)
	_ = os.WriteFile(filepath.Join(proj, "compose.yaml"), []byte("services: {}"), 0o644)

	st, _ := config.Open(filepath.Join(dir, "c.json"))
	d := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		if strings.Contains(strings.Join(a, " "), "context ls") {
			return []byte(`{"Name":"default","Current":true,"DockerEndpoint":"unix:///x"}`), nil
		}
		if strings.Contains(strings.Join(a, " "), "version") {
			return []byte("1.0"), nil
		}
		return []byte("[]"), nil
	})
	fd := fakeDocker{d}
	h := New(Deps{UI: http.NotFoundHandler(), Store: st,
		Contexts: &contexts.Service{Docker: fd, Store: st},
		Projects: &projects.Service{Docker: d, Store: st}})

	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	if rec := do("POST", "/api/v1/roots", `{"path":"relative"}`); rec.Code != 400 {
		t.Errorf("relative path: %d", rec.Code)
	}
	if rec := do("POST", "/api/v1/roots", `{"path":"`+filepath.Join(dir, "scan")+`"}`); rec.Code != 201 || !strings.Contains(rec.Body.String(), `"projects":1`) {
		t.Errorf("add: %d %s", rec.Code, rec.Body)
	}
	if rec := do("POST", "/api/v1/roots", `{"path":"`+filepath.Join(dir, "scan")+`"}`); rec.Code != 409 {
		t.Errorf("dup: %d", rec.Code)
	}
	if rec := do("GET", "/api/v1/projects", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"app"`) {
		t.Errorf("projects: %d %s", rec.Code, rec.Body)
	}
	if rec := do("GET", "/api/v1/contexts", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"selected":true`) {
		t.Errorf("contexts: %d %s", rec.Code, rec.Body)
	}
	if rec := do("PUT", "/api/v1/contexts/selected", `{"name":"ghost"}`); rec.Code != 404 {
		t.Errorf("select ghost: %d", rec.Code)
	}
	id := config.RootID(filepath.Join(dir, "scan"))
	if rec := do("DELETE", "/api/v1/roots/"+id, ""); rec.Code != 204 {
		t.Errorf("delete: %d", rec.Code)
	}
	if rec := do("GET", "/api/v1/projects", ""); rec.Body.String() != "[]\n" {
		t.Errorf("after delete: %s", rec.Body)
	}
}

type fakeDocker struct{ dockercli.Runner }

func (fakeDocker) UseContext(context.Context, string) error { return nil }
