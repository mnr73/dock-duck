package contexts

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"dockduck/internal/config"
	"dockduck/internal/dockercli"
)

type fake struct {
	used []string
	run  dockercli.RunnerFunc
}

func (f *fake) Output(ctx context.Context, a ...string) ([]byte, error) { return f.run(ctx, a...) }
func (f *fake) UseContext(_ context.Context, n string) error            { f.used = append(f.used, n); return nil }

func newSvc(t *testing.T) (*Service, *fake) {
	t.Helper()
	st, err := config.Open(filepath.Join(t.TempDir(), "c.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := &fake{run: func(_ context.Context, a ...string) ([]byte, error) {
		j := strings.Join(a, " ")
		switch {
		case j == "context ls --format json":
			return []byte(`{"Name":"default","Current":true,"DockerEndpoint":"unix:///var/run/docker.sock"}` + "\n" +
				`{"Name":"remote","Description":"prod","DockerEndpoint":"ssh://x"}`), nil
		case strings.HasPrefix(j, "--context default version"):
			return []byte("29.1.0\n"), nil
		case strings.HasPrefix(j, "--context remote version"):
			return nil, errors.New("Cannot connect to the Docker daemon\nmore")
		}
		return nil, errors.New("unexpected: " + j)
	}}
	return &Service{Docker: f, Store: st}, f
}

func TestListAndSelect(t *testing.T) {
	s, f := newSvc(t)
	list, err := s.List(context.Background())
	if err != nil || len(list) != 2 {
		t.Fatalf("%v %v", list, err)
	}
	if !list[0].Selected || !list[0].Available || list[0].Version != "29.1.0" {
		t.Errorf("default: %+v", list[0])
	}
	if list[1].Available || list[1].Error != "Cannot connect to the Docker daemon" {
		t.Errorf("remote: %+v", list[1])
	}

	if err := s.Select(context.Background(), "remote"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Selected(context.Background()); got != "remote" {
		t.Errorf("selected = %q", got)
	}
	if len(f.used) != 0 {
		t.Error("must not touch CLI default unless sync enabled")
	}
	if err := s.Select(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
	_ = s.Store.Update(func(c *config.Config) error { c.Settings.SyncCLI = true; return nil })
	if err := s.Select(context.Background(), "default"); err != nil || len(f.used) != 1 || f.used[0] != "default" {
		t.Errorf("sync: %v %v", err, f.used)
	}
}
