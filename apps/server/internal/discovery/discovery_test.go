package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a", "compose.yaml"))
	write(t, filepath.Join(root, "a", "compose.override.yml"))
	write(t, filepath.Join(root, "a", ".env"))
	write(t, filepath.Join(root, "a", "nested", "compose.yaml")) // inside a project: ignored
	write(t, filepath.Join(root, "g", "h", "docker-compose.yml"))
	write(t, filepath.Join(root, "node_modules", "x", "compose.yaml"))
	write(t, filepath.Join(root, ".hidden", "compose.yaml"))
	write(t, filepath.Join(root, "1", "2", "3", "4", "5", "compose.yaml")) // too deep

	got, err := Scan(context.Background(), root, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 projects, got %+v", got)
	}
	a := got[0]
	if filepath.Base(a.Dir) != "a" || len(a.Files) != 2 || a.EnvFile == "" {
		t.Errorf("a: %+v", a)
	}
	if filepath.Base(got[1].Dir) != "h" || got[1].EnvFile != "" {
		t.Errorf("h: %+v", got[1])
	}
}

func TestScanRootItselfIsProject(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "compose.yml"))
	got, err := Scan(context.Background(), root, 2)
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
}
