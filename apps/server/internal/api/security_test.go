package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func guarded() http.Handler {
	return guardBrowser(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
}

// With a password, any Host name is fine (reverse proxy on a domain); the Origin
// rule still applies. Without one, only localhost and IP literals.
func TestHostRuleDependsOnThePassword(t *testing.T) {
	set := false
	h := guardBrowser(func() bool { return set }, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	const domain = "duck.example.com"
	if got := do(h, "GET", domain, nil); got != 403 {
		t.Errorf("open server, foreign Host (DNS rebinding): %d", got)
	}
	set = true
	if got := do(h, "GET", domain, nil); got != 204 {
		t.Errorf("password set, domain behind a proxy: %d", got)
	}
	if got := do(h, "POST", domain, map[string]string{"Origin": "https://" + domain}); got != 204 {
		t.Errorf("own origin through the proxy: %d", got)
	}
	if got := do(h, "POST", domain, map[string]string{"Origin": "https://evil.example"}); got != 403 {
		t.Errorf("a foreign page must still be refused: %d", got)
	}
	set = false
	if got := do(h, "GET", domain, nil); got != 403 {
		t.Errorf("password removed: the strict rule is back immediately: %d", got)
	}
}

func do(h http.Handler, method, host string, hdr map[string]string) int {
	req := httptest.NewRequest(method, "http://"+host+"/api/v1/x", strings.NewReader("{}"))
	req.Host = host
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestCSRFIsRefused(t *testing.T) {
	h := guarded()
	cases := []struct {
		name string
		host string
		hdr  map[string]string
		want int
	}{
		{"curl, no Origin", "127.0.0.1:8080", nil, 204},
		{"own UI", "127.0.0.1:8080", map[string]string{"Origin": "http://127.0.0.1:8080"}, 204},
		{"vite dev origin", "127.0.0.1:8080", map[string]string{"Origin": "http://localhost:5173"}, 204},
		{"vite proxying (Host is the dev server)", "localhost:5199", map[string]string{"Origin": "http://localhost:5199"}, 204},
		{"evil page (text/plain POST, no preflight)", "127.0.0.1:8080", map[string]string{"Origin": "https://evil.example"}, 403},
		{"evil page on another local port", "127.0.0.1:8080", map[string]string{"Origin": "http://localhost:9999"}, 403},
		{"sandboxed iframe / file: (null)", "127.0.0.1:8080", map[string]string{"Origin": "null"}, 403},
		{"Sec-Fetch-Site cross-site", "127.0.0.1:8080", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"Sec-Fetch-Site same-origin", "127.0.0.1:8080", map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://127.0.0.1:8080"}, 204},
		{"garbage Origin", "127.0.0.1:8080", map[string]string{"Origin": "::::"}, 403},
	}
	for _, c := range cases {
		for _, m := range []string{"POST", "PUT", "DELETE"} {
			if got := do(h, m, c.host, c.hdr); got != c.want {
				t.Errorf("%s %s: got %d want %d", m, c.name, got, c.want)
			}
		}
	}
	// reading is not state-changing: a foreign page may GET (CORS stops it reading the answer)
	if got := do(h, "GET", "127.0.0.1:8080", map[string]string{"Origin": "https://evil.example"}); got != 204 {
		t.Errorf("GET: %d", got)
	}
}

func TestDNSRebindingHostsAreRefused(t *testing.T) {
	h := guarded()
	for host, want := range map[string]int{
		"localhost:8080": 204, "127.0.0.1:8080": 204, "[::1]:8080": 204, "127.0.0.1": 204, "192.168.1.5:8080": 204,
		"app.localhost:8080": 204,
		"evil.example:8080":  403, "evil.example": 403, "localhost.evil.example:8080": 403, "127.0.0.1.evil.example:8080": 403,
	} {
		if got := do(h, "GET", host, nil); got != want {
			t.Errorf("Host %q: got %d want %d", host, got, want)
		}
		if got := do(h, "POST", host, map[string]string{"Origin": "http://" + host}); got != want {
			t.Errorf("POST with matching Origin, Host %q: got %d want %d", host, got, want)
		}
	}
}
