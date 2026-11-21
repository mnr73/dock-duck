// Package files reads and writes the Compose and .env files of a project.
//
// Rules (AGENTS.md): the client never sends a path, only a file *id* that the
// server derived from the discovered project; every write is atomic, keeps the
// file's mode (a 0600 .env must stay 0600), is guarded by an ETag so concurrent
// edits are noticed, and leaves one .bak copy of the previous content.
package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// MaxSize bounds what the editor will load or save.
const MaxSize = 2 << 20

// Kind of a project file.
const (
	KindCompose = "compose"
	KindEnv     = "env"
)

var (
	ErrNotFound = errors.New("file not found")
	ErrConflict = errors.New("the file changed on disk")
	ErrTooLarge = errors.New("file is too large to edit here")
	ErrBinary   = errors.New("file is not valid UTF-8 text")
	ErrBadID    = errors.New("unknown file")
)

// Ref is one editable file of a project.
type Ref struct {
	ID     string `json:"id"` // "compose-0", "compose-1", "env"
	Kind   string `json:"kind"`
	Name   string `json:"name"` // base name
	Path   string `json:"path"` // absolute (shown to the user; never accepted back)
	Exists bool   `json:"exists"`
}

// Content is a file's text with the ETag to send back on save.
type Content struct {
	Ref
	Content string `json:"content"`
	ETag    string `json:"etag"`
	Mode    string `json:"mode"`
}

// Refs lists the editable files: every Compose file in merge order, then the
// project's .env (which may not exist yet).
func Refs(dir string, composeFiles []string, envFile string) []Ref {
	var out []Ref
	for i, p := range composeFiles {
		out = append(out, Ref{ID: fmt.Sprintf("compose-%d", i), Kind: KindCompose, Name: filepath.Base(p), Path: p, Exists: exists(p)})
	}
	env := envFile
	if env == "" {
		env = filepath.Join(dir, ".env")
	}
	out = append(out, Ref{ID: "env", Kind: KindEnv, Name: ".env", Path: env, Exists: exists(env)})
	return out
}

// Find resolves a file id to its Ref.
func Find(refs []Ref, id string) (Ref, error) {
	for _, r := range refs {
		if r.ID == id {
			return r, nil
		}
	}
	return Ref{}, ErrBadID
}

// Read loads the file. A missing .env is not an error: it yields empty content
// and an empty ETag, which `Write` treats as "create".
func Read(r Ref) (Content, error) {
	c := Content{Ref: r}
	f, err := os.Open(r.Path)
	if errors.Is(err, os.ErrNotExist) {
		c.Exists = false
		if r.Kind == KindCompose {
			return c, ErrNotFound
		}
		return c, nil
	}
	if err != nil {
		return c, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return c, err
	}
	if !st.Mode().IsRegular() {
		return c, ErrNotFound
	}
	if st.Size() > MaxSize {
		return c, ErrTooLarge
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return c, err
	}
	if !utf8.Valid(b) || strings.IndexByte(string(b), 0) >= 0 {
		return c, ErrBinary
	}
	c.Exists, c.Content, c.ETag, c.Mode = true, string(b), ETag(b), fmt.Sprintf("%04o", st.Mode().Perm())
	return c, nil
}

// ETag is a content hash (stable across mtime-only changes, e.g. `touch`).
func ETag(b []byte) string {
	sum := sha256.Sum256(b)
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

// Write replaces the file when ifMatch equals the current ETag (empty ifMatch
// means "the file must not exist"). It returns the new ETag.
func Write(r Ref, content, ifMatch string) (string, error) {
	if len(content) > MaxSize {
		return "", ErrTooLarge
	}
	if !utf8.ValidString(content) || strings.IndexByte(content, 0) >= 0 {
		return "", ErrBinary
	}
	// Write through symlinks: renaming over a link would replace the link itself
	// with a regular file and leave the real file stale.
	path := r.Path
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	r.Path = path
	cur, err := os.ReadFile(r.Path)
	mode := os.FileMode(0o644)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if ifMatch != "" {
			return "", ErrConflict
		}
		if r.Kind == KindEnv {
			mode = 0o600 // a new .env holds secrets: do not make it world-readable
		}
	case err != nil:
		return "", err
	default:
		if ifMatch == "" || ETag(cur) != ifMatch {
			return "", ErrConflict
		}
		st, serr := os.Stat(r.Path)
		if serr != nil {
			return "", serr
		}
		mode = st.Mode().Perm()
		if err := backup(r.Path, cur, mode); err != nil {
			return "", fmt.Errorf("backup: %w", err)
		}
	}
	if err := atomicWrite(r.Path, []byte(content), mode); err != nil {
		return "", err
	}
	return ETag([]byte(content)), nil
}

// backup keeps ONE previous version next to the file: <name>.bak (same mode as
// the original, so a .env backup is not more readable than the .env).
func backup(path string, prev []byte, mode os.FileMode) error {
	return atomicWrite(path+".bak", prev, mode)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".dockduck-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func exists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}
