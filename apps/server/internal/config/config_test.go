package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Get().Settings.ScanDepth != DefaultScanDepth {
		t.Fatal("default depth")
	}
	if err := s.Update(func(c *Config) error {
		c.Roots = append(c.Roots, Root{ID: "a", Path: "/x"})
		c.SelectedContext = "colima"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); len(got.Roots) != 1 || got.SelectedContext != "colima" {
		t.Fatalf("reload: %+v", got)
	}
	boom := errors.New("boom")
	if err := s2.Update(func(c *Config) error { c.SelectedContext = "z"; return boom }); !errors.Is(err, boom) {
		t.Fatal("error not propagated")
	}
	if s2.Get().SelectedContext != "colima" {
		t.Fatal("failed update must not change state")
	}
}
