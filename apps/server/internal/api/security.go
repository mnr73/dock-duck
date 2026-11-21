package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// guardBrowser protects the API from web pages the user happens to visit while
// DockDuck listens on loopback (until Phase 8 adds real authentication):
//
//   - CSRF: a cross-site form/fetch can POST `text/plain` JSON to 127.0.0.1 without a
//     CORS preflight. Browsers always attach `Origin` (and `Sec-Fetch-Site`) to such
//     requests, so a state-changing request from a foreign origin is refused.
//   - DNS rebinding: a hostile name that resolves to 127.0.0.1 makes Origin and Host
//     agree, so the Host itself must be a loopback name or an IP literal.
//
// Non-browser clients (curl, scripts) send no Origin and are unaffected.
//
// The Host rule only protects an open server (no password): a rebinding page has
// no session cookie, so once a password exists any Host name is accepted. That is
// what lets DockDuck sit behind a reverse proxy on a real domain name. The Origin
// check for state-changing requests and for WebSockets applies either way.
// `passwordSet` is read per request, so setting or removing the password takes
// effect without a restart.
func guardBrowser(passwordSet func() bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowedHost(r.Host) && !(passwordSet != nil && passwordSet()) {
			WriteError(w, http.StatusForbidden, "bad_host", "this host name is not allowed while no password is set; use localhost or an IP address, or set a password")
			return
		}
		if !safeMethod(r.Method) && !sameOrigin(r) {
			WriteError(w, http.StatusForbidden, "cross_origin", "cross-origin request refused")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// allowedHost accepts localhost, *.localhost and IP literals (with or without port).
func allowedHost(hostport string) bool {
	h := hostport
	if hh, _, err := net.SplitHostPort(hostport); err == nil {
		h = hh
	}
	h = strings.Trim(h, "[]")
	if h == "" {
		return false
	}
	if net.ParseIP(h) != nil {
		return true
	}
	h = strings.ToLower(h)
	return h == "localhost" || strings.HasSuffix(h, ".localhost")
}

// sameOrigin reports whether a state-changing request comes from DockDuck's own UI
// (or its Vite dev server on loopback). A missing Origin is a non-browser client.
func sameOrigin(r *http.Request) bool {
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs == "cross-site" || sfs == "same-site" {
		return false
	}
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	if o == "null" {
		return false
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return false
	}
	for _, p := range sameOriginPatterns(r) {
		if strings.EqualFold(u.Host, p) {
			return true
		}
	}
	return false
}
