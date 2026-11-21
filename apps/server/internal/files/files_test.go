package files

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode().Perm()
}

func TestRefs(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "compose.yaml"), filepath.Join(dir, "compose.override.yaml")
	_ = os.WriteFile(a, []byte("services: {}"), 0o644)
	refs := Refs(dir, []string{a, b}, "")
	if len(refs) != 3 || refs[0].ID != "compose-0" || refs[1].ID != "compose-1" || refs[2].ID != "env" {
		t.Fatalf("%+v", refs)
	}
	if !refs[0].Exists || refs[1].Exists || refs[2].Exists || refs[2].Path != filepath.Join(dir, ".env") {
		t.Errorf("exists flags / env path: %+v", refs)
	}
	if _, err := Find(refs, "../../etc/passwd"); !errors.Is(err, ErrBadID) {
		t.Error("only server-derived ids resolve")
	}
}

func TestReadWriteRoundTripKeepsModeAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	_ = os.WriteFile(p, []byte("A=1\n"), 0o600)
	r := Ref{ID: "env", Kind: KindEnv, Name: ".env", Path: p}

	c, err := Read(r)
	if err != nil || c.Content != "A=1\n" || c.Mode != "0600" || c.ETag == "" {
		t.Fatalf("%+v %v", c, err)
	}
	etag, err := Write(r, "A=2\n", c.ETag)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); string(got) != "A=2\n" {
		t.Errorf("content %q", got)
	}
	if mode(t, p) != 0o600 {
		t.Errorf("a 0600 .env must stay 0600, got %v", mode(t, p))
	}
	if bak, _ := os.ReadFile(p + ".bak"); string(bak) != "A=1\n" || mode(t, p+".bak") != 0o600 {
		t.Errorf(".bak must hold the previous content with the same mode: %q %v", bak, mode(t, p+".bak"))
	}
	// next save with the returned etag works; the old etag is a conflict now
	if _, err := Write(r, "A=3\n", etag); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(r, "A=9\n", c.ETag); !errors.Is(err, ErrConflict) {
		t.Errorf("stale etag: %v", err)
	}
	if got, _ := os.ReadFile(p); string(got) != "A=3\n" {
		t.Errorf("a rejected save must not change the file: %q", got)
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".dockduck-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left behind: %v", leftovers)
	}
}

func TestExternalChangeIsDetected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "compose.yaml")
	_ = os.WriteFile(p, []byte("services: {}\n"), 0o644)
	r := Ref{ID: "compose-0", Kind: KindCompose, Path: p}
	c, _ := Read(r)
	_ = os.WriteFile(p, []byte("services:\n  edited: elsewhere\n"), 0o644) // someone else saves
	if _, err := Write(r, "mine\n", c.ETag); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestNewEnvIsPrivateAndComposeMustExist(t *testing.T) {
	dir := t.TempDir()
	env := Ref{ID: "env", Kind: KindEnv, Path: filepath.Join(dir, ".env")}
	c, err := Read(env)
	if err != nil || c.Exists || c.Content != "" || c.ETag != "" {
		t.Fatalf("missing .env is an empty file, not an error: %+v %v", c, err)
	}
	if _, err := Write(env, "SECRET=x\n", ""); err != nil {
		t.Fatal(err)
	}
	if m := mode(t, env.Path); m != 0o600 {
		t.Errorf("a new .env must be 0600, got %v", m)
	}
	if _, err := Write(env, "again\n", ""); !errors.Is(err, ErrConflict) {
		t.Errorf("creating over an existing file must conflict: %v", err)
	}
	comp := Ref{Kind: KindCompose, Path: filepath.Join(dir, "compose.yaml")}
	if _, err := Read(comp); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing compose file: %v", err)
	}
}

func TestLimitsAndBinary(t *testing.T) {
	dir := t.TempDir()
	r := Ref{Kind: KindCompose, Path: filepath.Join(dir, "c.yaml")}
	_ = os.WriteFile(r.Path, []byte("ok"), 0o644)
	c, _ := Read(r)
	if _, err := Write(r, strings.Repeat("x", MaxSize+1), c.ETag); !errors.Is(err, ErrTooLarge) {
		t.Errorf("too large: %v", err)
	}
	if _, err := Write(r, "a\x00b", c.ETag); !errors.Is(err, ErrBinary) {
		t.Errorf("NUL: %v", err)
	}
	if _, err := Write(r, "bad \xff\xfe utf8", c.ETag); !errors.Is(err, ErrBinary) {
		t.Errorf("invalid utf8: %v", err)
	}
	_ = os.WriteFile(r.Path, []byte{0xff, 0xfe, 0x00}, 0o644)
	if _, err := Read(r); !errors.Is(err, ErrBinary) {
		t.Errorf("binary file on disk: %v", err)
	}
	big := Ref{Kind: KindCompose, Path: filepath.Join(dir, "big.yaml")}
	_ = os.WriteFile(big.Path, []byte(strings.Repeat("a", MaxSize+10)), 0o644)
	if _, err := Read(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("big file on disk: %v", err)
	}
}

func TestSymlinkedComposeFileIsWrittenThroughNotReplaced(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.yaml")
	link := filepath.Join(dir, "compose.yaml")
	_ = os.WriteFile(real, []byte("a: 1\n"), 0o644)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks")
	}
	r := Ref{Kind: KindCompose, Path: link}
	c, err := Read(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(r, "a: 2\n", c.ETag); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Lstat(link); st.Mode()&os.ModeSymlink == 0 {
		t.Error("saving must keep the symlink a symlink")
	}
	if got, _ := os.ReadFile(real); string(got) != "a: 2\n" {
		t.Errorf("target not updated: %q", got)
	}
}
