package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
)

func settingsEnv(t *testing.T) (http.Handler, *config.Store) {
	t.Helper()
	st, err := config.Open(filepath.Join(t.TempDir(), "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	d := dockercli.RunnerFunc(func(_ context.Context, args ...string) ([]byte, error) {
		switch {
		case len(args) >= 2 && args[0] == "context" && args[1] == "ls":
			return []byte(`{"Name":"default","Current":true}` + "\n" + `{"Name":"orbstack"}`), nil
		case len(args) >= 2 && args[0] == "compose" && args[1] == "version":
			return []byte("2.29.7\n"), nil
		}
		return []byte("[]"), nil
	})
	return New(Deps{UI: http.NotFoundHandler(), Store: st,
		Contexts: &contexts.Service{Docker: fakeDocker{d}, Store: st},
		Projects: &projects.Service{Docker: d, Store: st}}), st
}

func put(h http.Handler, body string) *httptest.ResponseRecorder {
	r := localReq("PUT", "/api/v1/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestSettingsReadDefaults(t *testing.T) {
	h, _ := settingsEnv(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, localReq("GET", "/api/v1/settings", nil))
	b := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(b, `"scan_depth":4`) || !strings.Contains(b, `"compose_version":"2.29.7"`) || !strings.Contains(b, `"max_scan_depth":8`) {
		t.Fatalf("%d %s", rec.Code, b)
	}
	if strings.Contains(b, "password") {
		t.Errorf("settings must never carry auth data: %s", b)
	}
}

func TestSettingsPartialUpdate(t *testing.T) {
	h, st := settingsEnv(t)
	if rec := put(h, `{"scan_depth":6}`); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec := put(h, `{"default_context":"orbstack"}`); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec := put(h, `{"sync_cli":true}`); rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	s := st.Get().Settings
	if s.ScanDepth != 6 || s.DefaultContext != "orbstack" || !s.SyncCLI {
		t.Errorf("a field was lost by a later partial update: %+v", s)
	}
	if rec := put(h, `{"default_context":""}`); rec.Code != 200 || st.Get().Settings.DefaultContext != "" {
		t.Errorf("an empty default context means 'last used' and must be allowed: %d", rec.Code)
	}
}

func TestSettingsValidation(t *testing.T) {
	h, st := settingsEnv(t)
	for name, body := range map[string]string{
		"depth 0":         `{"scan_depth":0}`,
		"depth negative":  `{"scan_depth":-3}`,
		"depth too deep":  `{"scan_depth":9}`,
		"unknown context": `{"default_context":"nope"}`,
		"flag as context": `{"default_context":"--host=tcp://evil"}`,
		"not json":        `scan_depth=3`,
	} {
		if rec := put(h, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	if s := st.Get().Settings; s.ScanDepth != config.DefaultScanDepth || s.DefaultContext != "" {
		t.Errorf("a rejected request must change nothing: %+v", s)
	}
	// cross-origin write is stopped by the browser guard
	r := localReq("PUT", "/api/v1/settings", strings.NewReader(`{"sync_cli":true}`))
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden || st.Get().Settings.SyncCLI {
		t.Errorf("cross-origin: %d", rec.Code)
	}
}
