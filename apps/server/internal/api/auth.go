package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dockduck/internal/auth"
)

// How a request is authorized:
//
//   - public: /api/v1/health, /api/v1/auth/*, and everything that is not /api/ or
//     /ws/ (the static UI shell holds no data; the data comes from the API).
//   - everything else under /api/ and /ws/ needs a session once a password exists
//     (or always, when the server is reachable from the network: Required).
//   - state-changing requests also need the session's CSRF token (X-CSRF-Token),
//     on top of the Origin check in guardBrowser.
//   - WebSocket upgrades are GETs, so they are covered by the session cookie and
//     the Origin check; they are closed when the session ends (see sessionContext).
//
// With no password configured on a loopback-only server the API stays open, as it
// was before Phase 8: the UI offers to set a password.

type ctxKey struct{}

// requireAuth wraps the whole mux.
func requireAuth(svc *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		protected := strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/")
		if !protected || path == "/api/v1/health" || strings.HasPrefix(path, "/api/v1/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		configured := svc.Configured()
		if !configured && !svc.Required {
			next.ServeHTTP(w, r) // open mode (loopback, no password yet)
			return
		}
		if !configured {
			WriteError(w, http.StatusUnauthorized, "setup_required", "no password is set; run `dockduck passwd` on the server")
			return
		}
		sess, ok := svc.FromRequest(r)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "unauthenticated", "login required")
			return
		}
		if !safeMethod(r.Method) && !csrfOK(r, sess) {
			WriteError(w, http.StatusForbidden, "bad_csrf", "missing or wrong CSRF token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, sess)))
	})
}

func csrfOK(r *http.Request, s auth.Session) bool {
	t := r.Header.Get("X-CSRF-Token")
	return t != "" && constantEq(t, s.CSRF)
}

func constantEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// sessionContext returns a context that ends when the request's session is
// revoked, so a logout (or a password change) also closes terminals and streams
// that are already open. Without auth it is the request context unchanged.
func sessionContext(r *http.Request) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(r.Context())
	if s, ok := r.Context().Value(ctxKey{}).(auth.Session); ok {
		go func() {
			select {
			case <-s.Done():
				cancel()
			case <-ctx.Done():
			}
		}()
	}
	return ctx, cancel
}

type authHandlers struct {
	svc *auth.Service
	// secure: the page is served over TLS (set on the cookie)
}

func registerAuth(mux *http.ServeMux, svc *auth.Service) {
	h := &authHandlers{svc}
	mux.HandleFunc("GET /api/v1/auth/status", h.status)
	mux.HandleFunc("POST /api/v1/auth/setup", h.setup)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("POST /api/v1/auth/password", h.password)
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isHTTPS(r),
		MaxAge:   int(auth.AbsoluteTimeout / time.Second),
	})
}

func clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r), MaxAge: -1})
}

// status tells the UI what to show. It never reveals more than whether a password
// exists and whether this browser is logged in.
func (h *authHandlers) status(w http.ResponseWriter, r *http.Request) {
	configured := h.svc.Configured()
	out := map[string]any{
		"configured":    configured,
		"required":      h.svc.Required,
		"authenticated": !configured && !h.svc.Required, // open mode counts as "in"
		"setup_here":    !configured && isLoopback(r.RemoteAddr),
	}
	if configured {
		if sess, ok := h.svc.FromRequest(r); ok {
			out["authenticated"] = true
			out["csrf"] = sess.CSRF
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, out)
}

type pwBody struct {
	Password string `json:"password"`
	Current  string `json:"current"`
	Next     string `json:"next"`
}

func (h *authHandlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var locked *auth.LockedError
	switch {
	case errors.As(err, &locked):
		secs := int(locked.Wait.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		WriteError(w, http.StatusTooManyRequests, "locked", "too many failed attempts; try again in "+locked.Wait.Round(time.Second).String())
	case errors.Is(err, auth.ErrBusy):
		w.Header().Set("Retry-After", "2")
		WriteError(w, http.StatusTooManyRequests, "busy", "the server is busy checking other logins; try again")
	case errors.Is(err, auth.ErrBadCredentials):
		WriteError(w, http.StatusUnauthorized, "bad_credentials", "wrong password")
	case errors.Is(err, auth.ErrPolicy):
		WriteError(w, http.StatusBadRequest, "weak_password", strings.TrimPrefix(err.Error(), "password policy: "))
	case errors.Is(err, auth.ErrAlreadyConfigured):
		WriteError(w, http.StatusConflict, "already_configured", "a password is already set")
	case errors.Is(err, auth.ErrNotConfigured):
		WriteError(w, http.StatusConflict, "not_configured", "no password is set")
	default:
		slog.Error("auth", "err", err)
		WriteError(w, http.StatusInternalServerError, "auth_error", "internal error")
	}
}

// setup sets the first password. Only from the machine itself: on a network-facing
// server anyone who can reach the port could otherwise claim it.
func (h *authHandlers) setup(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) {
		WriteError(w, http.StatusForbidden, "local_only", "set the first password on the server itself: run `dockduck passwd`")
		return
	}
	var b pwBody
	if !decode(w, r, &b) {
		return
	}
	tok, sess, err := h.svc.Setup(b.Password, auth.ClientKey(r.RemoteAddr))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	slog.Info("password set (first run)", "remote", r.RemoteAddr)
	setCookie(w, r, tok)
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "csrf": sess.CSRF})
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var b pwBody
	if !decode(w, r, &b) {
		return
	}
	key := auth.ClientKey(r.RemoteAddr)
	tok, sess, err := h.svc.Login(b.Password, key, key)
	if err != nil {
		slog.Warn("login failed", "remote", r.RemoteAddr, "reason", firstLine(err.Error()))
		h.fail(w, r, err)
		return
	}
	slog.Info("login", "remote", r.RemoteAddr)
	setCookie(w, r, tok)
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "csrf": sess.CSRF})
}

func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	// logout needs the CSRF token like any other state change, else a foreign page
	// could log the user out; without a session it is a harmless no-op
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if sess, ok := h.svc.FromRequest(r); ok {
			if !csrfOK(r, sess) {
				WriteError(w, http.StatusForbidden, "bad_csrf", "missing or wrong CSRF token")
				return
			}
			h.svc.Sessions.Delete(c.Value)
			slog.Info("logout", "remote", r.RemoteAddr)
		}
	}
	clearCookie(w, r)
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (h *authHandlers) password(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.svc.FromRequest(r)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}
	if !csrfOK(r, sess) {
		WriteError(w, http.StatusForbidden, "bad_csrf", "missing or wrong CSRF token")
		return
	}
	var b pwBody
	if !decode(w, r, &b) {
		return
	}
	key := auth.ClientKey(r.RemoteAddr)
	tok, nsess, err := h.svc.ChangePassword(b.Current, b.Next, key, key)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	slog.Info("password changed; all sessions ended", "remote", r.RemoteAddr)
	setCookie(w, r, tok)
	WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "csrf": nsess.CSRF})
}
