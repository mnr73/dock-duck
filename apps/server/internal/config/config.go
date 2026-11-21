// Package config persists DockDuck's small app state (roots, settings,
// selected context) as a JSON file. No database.
package config

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Root is a directory scanned for Compose projects.
type Root struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// Settings are user preferences (extended in later phases).
type Settings struct {
	DefaultContext string `json:"default_context"`
	SyncCLI        bool   `json:"sync_cli"`   // also run `docker context use` when switching
	ScanDepth      int    `json:"scan_depth"` // max directory depth below a root
}

// Auth holds the app login. Only a hash is stored (argon2id, PHC string).
type Auth struct {
	PasswordHash string `json:"password_hash,omitempty"`
}

// Config is the whole persisted document.
type Config struct {
	Roots           []Root   `json:"roots"`
	Settings        Settings `json:"settings"`
	SelectedContext string   `json:"selected_context"`
	Auth            Auth     `json:"auth"`
}

// DefaultScanDepth is used when Settings.ScanDepth is unset.
const DefaultScanDepth = 4

// MaxScanDepth bounds how deep a root is walked: every level multiplies the work.
const MaxScanDepth = 8

// RootID derives a stable id from a root path.
func RootID(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:10]
}

// DefaultPath returns ~/.config/dockduck/config.json (platform aware).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dockduck", "config.json"), nil
}

// Store is a concurrency-safe, file-backed Config.
type Store struct {
	path string
	mu   sync.RWMutex
	cfg  Config
	sig  fileSig // what the file looked like when we last read or wrote it
}

type fileSig struct {
	mod  time.Time
	size int64
}

func sigOf(path string) fileSig {
	st, err := os.Stat(path)
	if err != nil {
		return fileSig{}
	}
	return fileSig{st.ModTime(), st.Size()}
}

// Open loads path (a missing file yields defaults).
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &s.cfg); err != nil {
			return nil, err
		}
	}
	if s.cfg.Settings.ScanDepth <= 0 {
		s.cfg.Settings.ScanDepth = DefaultScanDepth
	}
	s.sig = sigOf(path)
	return s, nil
}

// Refresh re-reads the file when something else changed it (e.g. `dockduck passwd`
// while the server runs). It reports whether the content was reloaded; a file that
// cannot be parsed is ignored so a half-written edit never takes the server down.
func (s *Store) Refresh() bool {
	now := sigOf(s.path)
	s.mu.RLock()
	same := now == s.sig
	s.mu.RUnlock()
	if same {
		return false
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return false
	}
	var next Config
	if err := json.Unmarshal(b, &next); err != nil {
		return false
	}
	if next.Settings.ScanDepth <= 0 {
		next.Settings.ScanDepth = DefaultScanDepth
	}
	s.mu.Lock()
	s.cfg, s.sig = next, now
	s.mu.Unlock()
	return true
}

// Path is the file backing the store.
func (s *Store) Path() string { return s.path }

// Get returns a deep copy of the current config.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.cfg
	c.Roots = append([]Root(nil), s.cfg.Roots...)
	return c
}

// Update applies fn under the write lock and persists the result atomically.
// If fn returns an error nothing is changed.
func (s *Store) Update(fn func(*Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cfg
	next.Roots = append([]Root(nil), s.cfg.Roots...)
	if err := fn(&next); err != nil {
		return err
	}
	if err := save(s.path, next); err != nil {
		return err
	}
	s.cfg = next
	s.sig = sigOf(s.path)
	return nil
}

func save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
