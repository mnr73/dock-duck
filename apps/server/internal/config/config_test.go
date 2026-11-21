package config

import (
	"errors"
	"os"
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

func TestRefreshPicksUpExternalChanges(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	s, _ := Open(p)
	_ = s.Update(func(c *Config) error { c.SelectedContext = "a"; return nil })
	if s.Refresh() {
		t.Error("our own write is not an external change")
	}
	// another process (dockduck passwd) edits the file
	if err := os.WriteFile(p, []byte(`{"selected_context":"b","auth":{"password_hash":"h"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if !s.Refresh() || s.Get().SelectedContext != "b" || s.Get().Auth.PasswordHash != "h" {
		t.Errorf("reload: %+v", s.Get())
	}
	if s.Get().Settings.ScanDepth != DefaultScanDepth {
		t.Error("defaults survive a reload")
	}
	// a half-written file must not break a running server
	_ = os.WriteFile(p, []byte(`{"selected_con`), 0o600)
	if s.Refresh() || s.Get().SelectedContext != "b" {
		t.Error("an unparsable file is ignored")
	}
}
