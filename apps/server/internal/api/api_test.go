package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndFallback(t *testing.T) {
	ui := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ui")) })
	h := New(Info{Version: "t"}, ui)

	cases := []struct {
		path, want string
		code       int
	}{
		{"/api/v1/health", `"ok":true`, 200},
		{"/api/v1/nope", `not_found`, 404},
		{"/projects/x", "ui", 200},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.want) {
			t.Errorf("%s: got %d %q", c.path, rec.Code, rec.Body.String())
		}
	}
}
