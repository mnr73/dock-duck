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

// Config is the whole persisted document.
type Config struct {
	Roots           []Root   `json:"roots"`
	Settings        Settings `json:"settings"`
	SelectedContext string   `json:"selected_context"`
}

// DefaultScanDepth is used when Settings.ScanDepth is unset.
const DefaultScanDepth = 4

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
	return s, nil
}

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
