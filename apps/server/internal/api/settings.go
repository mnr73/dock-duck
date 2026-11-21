package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"dockduck/internal/config"
	"dockduck/internal/dockercli"
)

// settingsView is what the Settings page reads. Only server-side preferences live
// here; look and feel (theme, accent, terminal font ...) is per browser.
type settingsView struct {
	DefaultContext string `json:"default_context"`
	SyncCLI        bool   `json:"sync_cli"`
	ScanDepth      int    `json:"scan_depth"`
	MaxScanDepth   int    `json:"max_scan_depth"`
	// ComposeVersion is what `docker compose version --short` says ("" if unavailable).
	ComposeVersion string `json:"compose_version"`
	ConfigPath     string `json:"config_path"`
}

func (h *dockerHandlers) settingsView(ctx context.Context) settingsView {
	s := h.Store.Get().Settings
	v := settingsView{
		DefaultContext: s.DefaultContext, SyncCLI: s.SyncCLI, ScanDepth: s.ScanDepth,
		MaxScanDepth: config.MaxScanDepth, ConfigPath: h.Store.Path(),
	}
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	if b, err := h.Projects.Docker.Output(cctx, "compose", "version", "--short"); err == nil {
		v.ComposeVersion = strings.TrimSpace(string(b))
	}
	return v
}

func (h *dockerHandlers) getSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, h.settingsView(r.Context()))
}

// putSettings changes only the fields present in the body, so the UI can save one
// control at a time (the page saves automatically).
func (h *dockerHandlers) putSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DefaultContext *string `json:"default_context"`
		SyncCLI        *bool   `json:"sync_cli"`
		ScanDepth      *int    `json:"scan_depth"`
	}
	if !decode(w, r, &body) {
		return
	}
	if d := body.ScanDepth; d != nil && (*d < 1 || *d > config.MaxScanDepth) {
		WriteError(w, http.StatusBadRequest, "invalid_scan_depth", "scan depth must be between 1 and 8")
		return
	}
	if c := body.DefaultContext; c != nil && *c != "" {
		if !dockercli.ValidName(*c) {
			WriteError(w, http.StatusBadRequest, "invalid_context", "unsupported context name")
			return
		}
		ok, err := h.Contexts.Has(r.Context(), *c)
		if err != nil {
			WriteError(w, http.StatusBadGateway, "docker_unavailable", "cannot check the context: Docker is not reachable")
			return
		}
		if !ok {
			WriteError(w, http.StatusBadRequest, "invalid_context", "Docker has no context with this name")
			return
		}
	}
	rescan := false
	err := h.Store.Update(func(c *config.Config) error {
		if body.DefaultContext != nil {
			c.Settings.DefaultContext = *body.DefaultContext
		}
		if body.SyncCLI != nil {
			c.Settings.SyncCLI = *body.SyncCLI
		}
		if body.ScanDepth != nil && *body.ScanDepth != c.Settings.ScanDepth {
			c.Settings.ScanDepth = *body.ScanDepth
			rescan = true
		}
		return nil
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "save_failed", "could not save the settings")
		return
	}
	if rescan {
		// a different depth finds different projects; do not wait for the next manual rescan
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		_, _ = h.Projects.Rescan(ctx)
	}
	WriteJSON(w, http.StatusOK, h.settingsView(r.Context()))
}
