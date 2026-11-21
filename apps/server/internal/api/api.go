// Package api wires HTTP routes.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/events"
	"dockduck/internal/jobs"
	"dockduck/internal/ports"
	"dockduck/internal/projects"
)

// Info is static build/runtime metadata exposed by /api/v1/health.
type Info struct {
	Version string
	Home    string // user's home dir, lets the UI shorten paths to ~/…
}

// Deps are the services the API exposes. Nil services leave their routes unregistered.
type Deps struct {
	Info     Info
	UI       http.Handler
	Store    *config.Store
	Contexts *contexts.Service
	Projects *projects.Service
	Jobs     *jobs.Manager
	Events   *events.Hub
	Ports    *ports.Service
}

// New returns the root handler: /api/v1/* is the API, everything else goes to UI.
func New(d Deps) http.Handler {
	info, ui := d.Info, d.UI
	mux := http.NewServeMux()
	registerDocker(mux, d)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "version": info.Version, "home": info.Home})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotFound, "not_found", "unknown API endpoint")
	})
	mux.Handle("/", ui)
	return logRequests(mux)
}

// WriteJSON writes v as JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the standard error envelope.
func WriteError(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("request", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start))
	})
}
