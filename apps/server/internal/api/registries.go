package api

import (
	"errors"
	"log/slog"
	"net/http"

	"dockduck/internal/registries"
)

type loginBody struct {
	Server   string `json:"server"` // "" = Docker Hub
	Username string `json:"username"`
	Password string `json:"password"` // password or access token
}

func (h *dockerHandlers) registryState(w http.ResponseWriter, r *http.Request) {
	st, err := h.Registries.Read(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "registry_error", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, st)
}

func registryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, registries.ErrInvalid):
		WriteError(w, http.StatusBadRequest, "bad_request", registries.Reason(err))
	case errors.Is(err, registries.ErrFailed):
		// docker's own reason (wrong password, unreachable, ...), already scrubbed of the secret
		WriteError(w, http.StatusUnprocessableEntity, "registry_refused", registries.Reason(err))
	default:
		slog.Error("registry", "err", firstLine(err.Error()))
		WriteError(w, http.StatusInternalServerError, "registry_error", "internal error")
	}
}

func (h *dockerHandlers) registryLogin(w http.ResponseWriter, r *http.Request) {
	var b loginBody
	if !decode(w, r, &b) {
		return
	}
	res, err := h.Registries.Login(r.Context(), b.Server, b.Username, b.Password)
	b.Password = "" // gone as soon as docker has it
	if err != nil {
		slog.Warn("registry login failed", "server", b.Server, "user", b.Username, "remote", r.RemoteAddr)
		registryError(w, err)
		return
	}
	slog.Info("registry login", "server", res.Server, "user", b.Username, "remote", r.RemoteAddr)
	WriteJSON(w, http.StatusOK, res)
}

func (h *dockerHandlers) registryLogout(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Server string `json:"server"`
	}
	if !decode(w, r, &b) {
		return
	}
	if err := h.Registries.Logout(r.Context(), b.Server); err != nil {
		registryError(w, err)
		return
	}
	slog.Info("registry logout", "server", b.Server, "remote", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}
