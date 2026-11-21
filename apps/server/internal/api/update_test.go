package api

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
	"sync/atomic"
	"testing"
	"time"

	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
	"dockduck/internal/updater"
)

func fakeGitHub(t *testing.T, tag string, bin []byte) *httptest.Server {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "dockduck", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(bin)
	_ = tw.Close()
	_ = gz.Close()
	sum := sha256.Sum256(buf.Bytes())
	name := "dockduck_" + strings.TrimPrefix(tag, "v") + "_linux_amd64.tar.gz"
	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+updater.Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": tag, "html_url": "https://example.test", "assets": []updater.Asset{
			{Name: name, URL: base + "/a", Size: int64(buf.Len())}, {Name: "checksums.txt", URL: base + "/s"}}})
	})
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(buf.Bytes()) })
	mux.HandleFunc("/s", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  " + name + "\n"))
	})
	srv := httptest.NewServer(mux)
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func updateEnv(t *testing.T, gh *httptest.Server, version string, restarts *atomic.Int32) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "dockduck")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	st, _ := config.Open(filepath.Join(t.TempDir(), "c.json"))
	d := dockercli.RunnerFunc(func(context.Context, ...string) ([]byte, error) { return []byte("[]"), nil })
	return New(Deps{UI: http.NotFoundHandler(), Store: st,
		Contexts: &contexts.Service{Docker: fakeDocker{d}, Store: st},
		Projects: &projects.Service{Docker: d, Store: st},
		Updater:  &updater.Updater{Version: version, API: gh.URL, Exe: exe, OS: "linux", Arch: "amd64"},
		Restart:  func() { restarts.Add(1) }}), exe
}

func TestUpdateStatusAndInstall(t *testing.T) {
	gh := fakeGitHub(t, "v0.2.0", []byte("NEW-BINARY"))
	var restarts atomic.Int32
	h, exe := updateEnv(t, gh, "v0.1.0", &restarts)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, localReq("GET", "/api/v1/update", nil))
	var st updater.Status
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if rec.Code != 200 || !st.Available || st.Latest != "v0.2.0" || !st.CanInstall {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}

	rec = post(h, "/api/v1/update/install", `{}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"installed":"v0.2.0"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if b, _ := os.ReadFile(exe); string(b) != "NEW-BINARY" {
		t.Errorf("binary not replaced: %q", b)
	}
	deadline := time.Now().Add(3 * time.Second)
	for restarts.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if restarts.Load() != 1 {
		t.Errorf("restart requested %d times, want 1", restarts.Load())
	}
	// a second install must not run again; status reports the pending restart
	if rec := post(h, "/api/v1/update/install", `{}`); rec.Code != http.StatusConflict {
		t.Errorf("second install: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, localReq("GET", "/api/v1/update", nil))
	if !strings.Contains(rec.Body.String(), `"installed":"v0.2.0"`) {
		t.Errorf("status after install: %s", rec.Body)
	}
}

func TestUpdateRefusedForDevAndCrossOrigin(t *testing.T) {
	gh := fakeGitHub(t, "v0.2.0", []byte("NEW"))
	var restarts atomic.Int32
	h, exe := updateEnv(t, gh, "dev", &restarts)
	if rec := post(h, "/api/v1/update/install", `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("dev build: %d %s", rec.Code, rec.Body)
	}
	h2, exe2 := updateEnv(t, gh, "v0.1.0", &restarts)
	r := localReq("POST", "/api/v1/update/install", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h2.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a foreign page must not be able to trigger an update: %d", rec.Code)
	}
	for _, p := range []string{exe, exe2} {
		if b, _ := os.ReadFile(p); string(b) != "OLD" {
			t.Errorf("binary changed although the update was refused")
		}
	}
	if restarts.Load() != 0 {
		t.Error("no restart without an install")
	}
}
