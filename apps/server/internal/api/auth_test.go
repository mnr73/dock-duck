package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dockduck/internal/auth"
	"dockduck/internal/config"
)

type authEnv struct {
	h     http.Handler
	svc   *auth.Service
	store *config.Store
}

func newAuthEnv(t *testing.T, required bool) *authEnv {
	t.Helper()
	st, err := config.Open(filepath.Join(t.TempDir(), "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := auth.NewService(st, required)
	ui := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>ui</html>")) })
	h := New(Deps{UI: ui, Store: st, Auth: svc})
	return &authEnv{h: h, svc: svc, store: st}
}

type cli struct {
	t      *testing.T
	e      *authEnv
	cookie *http.Cookie
	csrf   string
	remote string
}

func (e *authEnv) client(t *testing.T) *cli { return &cli{t: t, e: e, remote: "127.0.0.1:50000"} }

func (c *cli) do(method, path, body string, hdr ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "127.0.0.1:8080"
	r.RemoteAddr = c.remote
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		r.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	c.e.h.ServeHTTP(rec, r)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == auth.CookieName {
			if ck.MaxAge < 0 {
				c.cookie = nil
			} else {
				c.cookie = ck
			}
		}
	}
	return rec
}

// mutate sends a state-changing request the way the UI does: CSRF token + same-origin.
func (c *cli) mutate(method, path, body string) *httptest.ResponseRecorder {
	return c.do(method, path, body, "X-CSRF-Token", c.csrf, "Origin", "http://127.0.0.1:8080", "Content-Type", "application/json")
}

func (c *cli) json(rec *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	return m
}

func (c *cli) login(pw string) *httptest.ResponseRecorder {
	rec := c.do("POST", "/api/v1/auth/login", `{"password":"`+pw+`"}`, "Content-Type", "application/json")
	if rec.Code == 200 {
		c.csrf, _ = c.json(rec)["csrf"].(string)
	}
	return rec
}

const goodPW = "a perfectly fine passphrase"

func setup(t *testing.T, e *authEnv) *cli {
	t.Helper()
	c := e.client(t)
	rec := c.do("POST", "/api/v1/auth/setup", `{"password":"`+goodPW+`"}`, "Content-Type", "application/json")
	if rec.Code != 200 {
		t.Fatalf("setup: %d %s", rec.Code, rec.Body)
	}
	c.csrf, _ = c.json(rec)["csrf"].(string)
	return c
}

func fastArgon(t *testing.T) {
	t.Helper()
	// the production cost is far too slow for unit tests; auth's own tests cover it
	t.Setenv("DOCKDUCK_TEST_FAST_ARGON", "1")
}

func TestOpenModeUntilAPasswordExists(t *testing.T) {
	e := newAuthEnv(t, false)
	c := e.client(t)
	st := c.json(c.do("GET", "/api/v1/auth/status", ""))
	if st["configured"] != false || st["authenticated"] != true || st["setup_here"] != true {
		t.Errorf("fresh loopback install: %+v", st)
	}
	if rec := c.do("GET", "/api/v1/contexts", ""); rec.Code == http.StatusUnauthorized {
		t.Errorf("no password yet: the API stays open (as before Phase 8), got %d", rec.Code)
	}
}

func TestEverythingIsClosedOnceAPasswordIsSet(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	c := setup(t, e)
	anon := e.client(t)

	for _, p := range []string{"/api/v1/contexts", "/api/v1/projects", "/api/v1/terminals", "/api/v1/ports", "/api/v1/terminals/system", "/ws/v1/events", "/ws/v1/terminals/abc", "/api/v1/nonsense"} {
		if rec := anon.do("GET", p, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s: %d (want 401)", p, rec.Code)
		}
		if rec := c.do("GET", p, ""); rec.Code == http.StatusUnauthorized {
			t.Errorf("logged-in GET %s: %d", p, rec.Code)
		}
	}
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		if rec := anon.do(m, "/api/v1/projects/x/actions/up", "{}"); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous %s: %d", m, rec.Code)
		}
	}
	// the shell of the UI and health stay public: they carry no data
	for _, p := range []string{"/", "/projects/abc/terminal", "/assets/x.js", "/api/v1/health", "/api/v1/auth/status"} {
		if rec := anon.do("GET", p, ""); rec.Code == http.StatusUnauthorized {
			t.Errorf("%s must be public, got 401", p)
		}
	}
	if body := anon.do("GET", "/api/v1/projects", "").Body.String(); strings.Contains(body, "[") && !strings.Contains(body, "error") {
		t.Errorf("a 401 must not carry data: %s", body)
	}
	// forged / garbage cookies
	for _, v := range []string{"", "x", strings.Repeat("a", 43), "../../etc/passwd"} {
		anon.cookie = &http.Cookie{Name: auth.CookieName, Value: v}
		if rec := anon.do("GET", "/api/v1/projects", ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("cookie %q accepted: %d", v, rec.Code)
		}
	}
}

func TestCSRFTokenIsRequiredForStateChanges(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	c := setup(t, e)
	origin := []string{"Origin", "http://127.0.0.1:8080", "Content-Type", "application/json"}
	if rec := c.do("POST", "/api/v1/roots/rescan", "{}", origin...); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "bad_csrf") {
		t.Errorf("no token: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do("POST", "/api/v1/roots/rescan", "{}", append(origin, "X-CSRF-Token", "wrong")...); rec.Code != http.StatusForbidden {
		t.Errorf("wrong token: %d", rec.Code)
	}
	if rec := c.do("POST", "/api/v1/roots/rescan", "{}", append(origin, "X-CSRF-Token", c.csrf[:len(c.csrf)-1])...); rec.Code != http.StatusForbidden {
		t.Errorf("truncated token: %d", rec.Code)
	}
	if rec := c.mutate("POST", "/api/v1/roots/rescan", "{}"); rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
		t.Errorf("right token: %d %s", rec.Code, rec.Body)
	}
	// reading never needs the token
	if rec := c.do("GET", "/api/v1/contexts", ""); rec.Code == http.StatusForbidden {
		t.Errorf("GET must not need CSRF: %d", rec.Code)
	}
	// a token from another session is wrong for this one
	other := e.client(t)
	other.login(goodPW)
	if rec := c.do("POST", "/api/v1/roots/rescan", "{}", append(origin, "X-CSRF-Token", other.csrf)...); rec.Code != http.StatusForbidden {
		t.Errorf("another session's token: %d", rec.Code)
	}
}

func TestLoginLogout(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	setup(t, e)
	c := e.client(t)
	if rec := c.login("not the password"); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "wrong password") {
		t.Errorf("wrong: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do("GET", "/api/v1/projects", ""); rec.Code != http.StatusUnauthorized {
		t.Error("a failed login must not give a session")
	}
	rec := c.login(goodPW)
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	ck := rec.Result().Cookies()[0]
	if !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode || ck.Path != "/" || ck.Secure {
		t.Errorf("cookie flags: %+v", ck)
	}
	st := c.json(c.do("GET", "/api/v1/auth/status", ""))
	if st["authenticated"] != true || st["csrf"] != c.csrf {
		t.Errorf("status: %+v", st)
	}
	if rec := c.do("POST", "/api/v1/auth/logout", "{}", "Origin", "http://127.0.0.1:8080"); rec.Code != http.StatusForbidden {
		t.Errorf("logout without the CSRF token (a foreign page could log you out): %d", rec.Code)
	}
	old := *c.cookie
	if rec := c.mutate("POST", "/api/v1/auth/logout", "{}"); rec.Code != 200 {
		t.Fatalf("logout: %d", rec.Code)
	}
	replay := e.client(t)
	replay.cookie = &old
	if rec := replay.do("GET", "/api/v1/projects", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("a logged-out cookie must be dead on the server, not just deleted in the browser: %d", rec.Code)
	}
}

func TestCookieIsSecureBehindTLS(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	setup(t, e)
	r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"password":"`+goodPW+`"}`))
	r.Host, r.RemoteAddr = "127.0.0.1:8080", "127.0.0.1:1"
	r.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, r)
	if cs := rec.Result().Cookies(); len(cs) == 0 || !cs[0].Secure {
		t.Errorf("Secure flag behind a TLS proxy: %+v", cs)
	}
}

func TestSetupRules(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, true) // network-facing
	remote := e.client(t)
	remote.remote = "203.0.113.7:4444"
	if rec := remote.do("POST", "/api/v1/auth/setup", `{"password":"`+goodPW+`"}`, "Content-Type", "application/json"); rec.Code != http.StatusForbidden {
		t.Errorf("nobody on the network may claim a fresh server: %d", rec.Code)
	}
	if rec := remote.do("GET", "/api/v1/projects", ""); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "setup_required") {
		t.Errorf("network-facing without a password stays closed: %d %s", rec.Code, rec.Body)
	}
	st := remote.json(remote.do("GET", "/api/v1/auth/status", ""))
	if st["authenticated"] != false || st["setup_here"] != false {
		t.Errorf("status for a remote on an unconfigured server: %+v", st)
	}

	local := e.client(t)
	for _, weak := range []string{"short", "          ", ""} {
		if rec := local.do("POST", "/api/v1/auth/setup", `{"password":"`+weak+`"}`, "Content-Type", "application/json"); rec.Code != http.StatusBadRequest {
			t.Errorf("weak %q: %d", weak, rec.Code)
		}
	}
	if rec := local.do("POST", "/api/v1/auth/setup", `{"password":"`+goodPW+`"}`, "Content-Type", "application/json"); rec.Code != 200 {
		t.Fatalf("local setup: %d", rec.Code)
	}
	if rec := local.do("POST", "/api/v1/auth/setup", `{"password":"another long passphrase"}`, "Content-Type", "application/json"); rec.Code != http.StatusConflict {
		t.Errorf("a second setup must not replace the password: %d", rec.Code)
	}
	if rec := remote.login("another long passphrase"); rec.Code != http.StatusUnauthorized {
		t.Errorf("the first password stays: %d", rec.Code)
	}
}

func TestBruteForceIsThrottled(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	setup(t, e)
	c := e.client(t)
	c.remote = "198.51.100.9:1111"
	var locked *httptest.ResponseRecorder
	for i := 0; i < 12; i++ {
		rec := c.login("wrong password " + string(rune('a'+i)))
		if rec.Code == http.StatusTooManyRequests {
			locked = rec
			break
		}
	}
	if locked == nil {
		t.Fatal("repeated wrong passwords must end in a lock")
	}
	if ra := locked.Header().Get("Retry-After"); ra == "" || ra == "0" {
		t.Errorf("Retry-After: %q", ra)
	}
	// while locked even the RIGHT password is refused (no oracle for the attacker)
	if rec := c.login(goodPW); rec.Code != http.StatusTooManyRequests {
		t.Errorf("right password while locked: %d", rec.Code)
	}
	// another client is not affected
	other := e.client(t)
	other.remote = "198.51.100.77:2222"
	if rec := other.login(goodPW); rec.Code != 200 {
		t.Errorf("other client: %d", rec.Code)
	}
}

func TestChangePasswordEndsEverySession(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	a := setup(t, e)
	b := e.client(t)
	b.login(goodPW)
	oldB := *b.cookie

	if rec := a.mutate("POST", "/api/v1/auth/password", `{"current":"nope nope nope","next":"a brand new passphrase"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong current password: %d", rec.Code)
	}
	if rec := a.mutate("POST", "/api/v1/auth/password", `{"current":"`+goodPW+`","next":"short"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("weak new password: %d", rec.Code)
	}
	if rec := a.mutate("POST", "/api/v1/auth/password", `{"current":"`+goodPW+`","next":"a brand new passphrase"}`); rec.Code != 200 {
		t.Fatalf("change: %d %s", rec.Code, rec.Body)
	}
	a.csrf, _ = a.json(a.do("GET", "/api/v1/auth/status", ""))["csrf"].(string)
	if rec := a.do("GET", "/api/v1/contexts", ""); rec.Code == http.StatusUnauthorized {
		t.Error("the browser that changed it gets a fresh session")
	}
	b.cookie = &oldB
	if rec := b.do("GET", "/api/v1/contexts", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("every other session must end: %d", rec.Code)
	}
	if rec := e.client(t).login(goodPW); rec.Code != http.StatusUnauthorized {
		t.Error("the old password is dead")
	}
	if rec := e.client(t).login("a brand new passphrase"); rec.Code != 200 {
		t.Error("the new password works")
	}
}

// `dockduck passwd` edits the file under a running server: it must take effect and end sessions.
func TestPasswdOnTheMachineEndsSessionsOfARunningServer(t *testing.T) {
	fastArgon(t)
	e := newAuthEnv(t, false)
	c := setup(t, e)
	hash, _ := auth.Hash("reset from the console")
	time.Sleep(1100 * time.Millisecond) // the server rechecks the file at most once a second
	other, _ := config.Open(storePath(t, e))
	_ = other.Update(func(cf *config.Config) error { cf.Auth.PasswordHash = hash; return nil })
	time.Sleep(1100 * time.Millisecond)
	if rec := c.do("GET", "/api/v1/contexts", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("sessions made under the old password must end: %d", rec.Code)
	}
	if rec := e.client(t).login("reset from the console"); rec.Code != 200 {
		t.Errorf("the new password works without restarting: %d", rec.Code)
	}
}

func storePath(t *testing.T, e *authEnv) string {
	t.Helper()
	return e.store.Path()
}
