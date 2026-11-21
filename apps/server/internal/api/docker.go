package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
)

func registerDocker(mux *http.ServeMux, d Deps) {
	if d.Contexts == nil || d.Projects == nil || d.Store == nil {
		return
	}
	h := &dockerHandlers{d}

	mux.HandleFunc("GET /api/v1/contexts", h.listContexts)
	mux.HandleFunc("PUT /api/v1/contexts/selected", h.selectContext)

	mux.HandleFunc("GET /api/v1/roots", h.listRoots)
	mux.HandleFunc("POST /api/v1/roots", h.addRoot)
	mux.HandleFunc("DELETE /api/v1/roots/{id}", h.deleteRoot)
	mux.HandleFunc("POST /api/v1/roots/rescan", h.rescan)

	mux.HandleFunc("GET /api/v1/projects", h.listProjects)
	if d.Jobs != nil {
		mux.HandleFunc("GET /api/v1/projects/{id}", h.projectDetail)
		mux.HandleFunc("GET /api/v1/projects/{id}/stats", h.projectStats)
		mux.HandleFunc("POST /api/v1/projects/{id}/actions/{action}", h.projectAction)
		mux.HandleFunc("GET /api/v1/jobs", h.listJobs)
		mux.HandleFunc("GET /api/v1/jobs/{id}", h.getJob)
		mux.HandleFunc("GET /ws/v1/jobs/{id}", h.jobStream)
	}
	if d.Events != nil {
		mux.HandleFunc("GET /ws/v1/events", h.eventStream)
	}
}

type dockerHandlers struct{ Deps }

func (h *dockerHandlers) listContexts(w http.ResponseWriter, r *http.Request) {
	list, err := h.Contexts.List(r.Context())
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_unavailable", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *dockerHandlers) selectContext(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}
	switch err := h.Contexts.Select(r.Context(), body.Name); {
	case errors.Is(err, contexts.ErrNotFound):
		WriteError(w, http.StatusNotFound, "context_not_found", err.Error())
	case err != nil:
		WriteError(w, http.StatusInternalServerError, "select_failed", err.Error())
	default:
		WriteJSON(w, http.StatusOK, map[string]string{"selected": body.Name})
	}
}

func (h *dockerHandlers) listRoots(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, h.Store.Get().Roots)
}

func (h *dockerHandlers) addRoot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &body) {
		return
	}
	p, err := cleanDir(body.Path)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	root := config.Root{ID: config.RootID(p), Path: p}
	err = h.Store.Update(func(c *config.Config) error {
		for _, x := range c.Roots {
			if x.ID == root.ID {
				return errDuplicate
			}
		}
		c.Roots = append(c.Roots, root)
		return nil
	})
	switch {
	case errors.Is(err, errDuplicate):
		WriteError(w, http.StatusConflict, "duplicate_root", "this directory is already a root")
	case err != nil:
		WriteError(w, http.StatusInternalServerError, "save_failed", err.Error())
	default:
		n, _ := h.Projects.Rescan(r.Context())
		WriteJSON(w, http.StatusCreated, map[string]any{"root": root, "projects": n})
	}
}

func (h *dockerHandlers) deleteRoot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	found := false
	err := h.Store.Update(func(c *config.Config) error {
		kept := c.Roots[:0]
		for _, x := range c.Roots {
			if x.ID == id {
				found = true
				continue
			}
			kept = append(kept, x)
		}
		c.Roots = kept
		return nil
	})
	switch {
	case err != nil:
		WriteError(w, http.StatusInternalServerError, "save_failed", err.Error())
	case !found:
		WriteError(w, http.StatusNotFound, "root_not_found", "unknown root")
	default:
		_, _ = h.Projects.Rescan(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *dockerHandlers) rescan(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	n, err := h.Projects.Rescan(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "scan_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]int{"projects": n})
}

func (h *dockerHandlers) listProjects(w http.ResponseWriter, r *http.Request) {
	name, err := h.contextFor(r)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_unavailable", err.Error())
		return
	}
	list, err := h.Projects.List(r.Context(), name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	if len(list) == 0 {
		WriteJSON(w, http.StatusOK, []struct{}{})
		return
	}
	WriteJSON(w, http.StatusOK, list)
}

// contextFor resolves the docker context: X-Docker-Context header, else the
// `context` query parameter (browsers cannot set headers on WebSockets), else
// the app's saved selection.
func (h *dockerHandlers) contextFor(r *http.Request) (string, error) {
	n := r.Header.Get("X-Docker-Context")
	if n == "" {
		n = r.URL.Query().Get("context")
	}
	if n != "" {
		if !dockercli.ValidName(n) {
			return "", errors.New("invalid X-Docker-Context")
		}
		return n, nil
	}
	return h.Contexts.Selected(r.Context())
}

var errDuplicate = errors.New("duplicate")

func cleanDir(p string) (string, error) {
	if p == "" || !filepath.IsAbs(p) {
		return "", errors.New("path must be absolute")
	}
	p = filepath.Clean(p)
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return "", errors.New("not an existing directory")
	}
	return p, nil
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return false
	}
	return true
}
