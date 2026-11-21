package projects

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dockduck/internal/config"
	"dockduck/internal/dockercli"
)

func TestParseStatus(t *testing.T) {
	cases := []struct {
		in     string
		r, t   int
		status string
	}{
		{"running(4)", 4, 4, "running"},
		{"running(2), exited(1)", 2, 3, "partial"},
		{"exited(6)", 0, 6, "stopped"},
		{"", 0, 0, "stopped"},
	}
	for _, c := range cases {
		r, tot, st := parseStatus(c.in)
		if r != c.r || tot != c.t || st != c.status {
			t.Errorf("%q -> %d %d %s", c.in, r, tot, st)
		}
	}
}

func TestDefaultName(t *testing.T) {
	if got := DefaultName("/x/My App.v2"); got != "myappv2" {
		t.Errorf("got %q", got)
	}
}

func TestListMerge(t *testing.T) {
	root := t.TempDir()
	mk := func(rel string) string {
		d := filepath.Join(root, rel)
		_ = os.MkdirAll(d, 0o755)
		_ = os.WriteFile(filepath.Join(d, "compose.yaml"), []byte("services: {}"), 0o644)
		return d
	}
	up, down := mk("up"), mk("down")
	outside := "/elsewhere/proj"

	st, _ := config.Open(filepath.Join(t.TempDir(), "c.json"))
	_ = st.Update(func(c *config.Config) error { c.Roots = []config.Root{{ID: "r", Path: root}}; return nil })

	ls := `[{"Name":"custom-name","Status":"running(2), exited(1)","ConfigFiles":"` + filepath.Join(up, "compose.yaml") + `"},` +
		`{"Name":"ext","Status":"exited(1)","ConfigFiles":"` + outside + `/compose.yml"}]`
	r := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) { return []byte(ls), nil })
	svc := &Service{Docker: r, Store: st}

	got, err := svc.List(context.Background(), "default")
	if err != nil || len(got) != 3 {
		t.Fatalf("%v %+v", err, got)
	}
	by := map[string]Project{}
	for _, p := range got {
		by[p.Dir] = p
	}
	if p := by[up]; p.Name != "custom-name" || p.Status != "partial" || p.Running != 2 || p.Root == "" {
		t.Errorf("up: %+v", p)
	}
	if p := by[down]; p.Status != "stopped" || p.Name != "down" {
		t.Errorf("down: %+v", p)
	}
	if p := by[outside]; p.Name != "ext" || p.Root != "" || p.Status != "stopped" {
		t.Errorf("ext: %+v", p)
	}
}
