package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"dockduck/internal/updater"
)

// updateState serialises installs and caches the last check, so repeated clicks
// do not hammer GitHub's unauthenticated rate limit (60 requests an hour).
type updateState struct {
	mu        sync.Mutex
	installed string // tag installed by this process, waiting for the restart
	checkedAt time.Time
	last      *updater.Status
}

const checkCache = 2 * time.Minute

func (h *dockerHandlers) updateStatus(w http.ResponseWriter, r *http.Request) {
	u := h.Updater
	h.upd.mu.Lock()
	defer h.upd.mu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	if h.upd.installed != "" {
		// already replaced on disk: what runs now is the old code until the restart
		WriteJSON(w, http.StatusOK, map[string]any{"installed": h.upd.installed, "current": u.Version})
		return
	}
	if r.URL.Query().Get("refresh") == "" && h.upd.last != nil && time.Since(h.upd.checkedAt) < checkCache {
		WriteJSON(w, http.StatusOK, h.upd.last)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	st, err := u.Check(ctx)
	if err != nil {
		// the status still says what this build can do; the error is why the check failed
		WriteJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"code": "update_check_failed", "message": err.Error()}, "status": st})
		return
	}
	h.upd.last, h.upd.checkedAt = &st, time.Now()
	WriteJSON(w, http.StatusOK, st)
}

// installUpdate downloads, verifies and swaps the binary, answers, and only then
// asks the process to restart (so the browser gets its answer first).
func (h *dockerHandlers) installUpdate(w http.ResponseWriter, r *http.Request) {
	h.upd.mu.Lock()
	defer h.upd.mu.Unlock()
	if h.upd.installed != "" {
		WriteError(w, http.StatusConflict, "already_installed", "an update was already installed; restart is pending")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	tag, err := h.Updater.Install(ctx)
	if err != nil {
		slog.Warn("update failed", "err", firstLine(err.Error()), "remote", r.RemoteAddr)
		WriteError(w, http.StatusUnprocessableEntity, "update_failed", err.Error())
		return
	}
	h.upd.installed = tag
	slog.Info("update installed", "from", h.Updater.Version, "to", tag, "remote", r.RemoteAddr)
	how := "self"
	if updater.Managed() {
		how = "service"
	}
	WriteJSON(w, http.StatusOK, map[string]any{"installed": tag, "restart": how})
	if h.Restart != nil {
		go func() {
			time.Sleep(400 * time.Millisecond) // let the response reach the browser
			h.Restart()
		}()
	}
}
