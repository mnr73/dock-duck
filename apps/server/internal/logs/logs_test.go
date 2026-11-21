package logs

import (
	"strings"
	"testing"

	"dockduck/internal/projects"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in         string
		svc, ts, m string
	}{
		{"shivaerp-redis  | 2026-11-21T07:59:05.871936483Z 1:M * BGSAVE done", "shivaerp-redis", "2026-11-21T07:59:05.871936483Z", "1:M * BGSAVE done"},
		{"app-1  | 2026-11-21T07:59:05Z a | b | c", "app-1", "2026-11-21T07:59:05Z", "a | b | c"},
		{"app-1  | 2026-11-21T07:59:05Z ", "app-1", "2026-11-21T07:59:05Z", ""},
		{"app-1  | no timestamp here", "app-1", "", "no timestamp here"},
		{"no such service: nosuch", "", "", "no such service: nosuch"},
		{"some message | with pipe", "", "", "some message | with pipe"},
		{"", "", "", ""},
	}
	for _, c := range cases {
		got := Parse(c.in)
		if got.Service != c.svc || got.Time != c.ts || got.Text != c.m {
			t.Errorf("Parse(%q) = %+v", c.in, got)
		}
	}
}

func TestArgs(t *testing.T) {
	p := projects.Project{Dir: "/p", Files: []string{"/p/c.yml"}}
	a, err := Args("remote", p, Options{Services: []string{"app", "db"}, Tail: 200, Follow: true, Since: "15m"})
	if err != nil {
		t.Fatal(err)
	}
	j := strings.Join(a, " ")
	for _, want := range []string{"--context remote compose", "logs --timestamps --no-color --tail 200 --since 15m -f app db"} {
		if !strings.Contains(j, want) {
			t.Errorf("missing %q in %s", want, j)
		}
	}
	all, _ := Args("", p, Options{Tail: -1})
	if strings.Contains(strings.Join(all, " "), "--tail") {
		t.Error("tail<0 must not pass --tail")
	}
	big, _ := Args("", p, Options{Tail: 10_000_000})
	if !strings.Contains(strings.Join(big, " "), "--tail 20000") {
		t.Errorf("tail must be capped: %v", big)
	}
	for _, bad := range []Options{{Services: []string{"--all"}}, {Services: []string{"a;b"}}, {Since: "yesterday"}, {Since: "-1h; rm"}} {
		if _, err := Args("", p, bad); err == nil {
			t.Errorf("%+v must be rejected", bad)
		}
	}
	if _, err := Args("", p, Options{Since: "2026-11-21T10:00:00Z"}); err != nil {
		t.Errorf("RFC3339 since: %v", err)
	}
}
