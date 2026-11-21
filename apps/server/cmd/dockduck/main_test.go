package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dockduck/internal/auth"
	"dockduck/internal/config"
)

func TestIsLoopbackHost(t *testing.T) {
	for h, want := range map[string]bool{
		"127.0.0.1": true, "::1": true, "localhost": true, "127.0.0.5": true,
		"": false, "0.0.0.0": false, "::": false, "192.168.1.5": false, "example.com": false, "10.0.0.1": false,
	} {
		if got := isLoopbackHost(h); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", h, got, want)
		}
	}
}

func pipeIn(t *testing.T, content string) *os.File {
	t.Helper()
	f := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { in.Close() })
	return in
}

func TestPasswdSetsAHashNeverThePassword(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	var out, errOut bytes.Buffer
	if code := runPasswd([]string{"--config", cfg}, pipeIn(t, "a perfectly fine passphrase\n"), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	raw, _ := os.ReadFile(cfg)
	if strings.Contains(string(raw), "perfectly fine") {
		t.Fatal("the password must never be written to disk")
	}
	st, _ := os.Stat(cfg)
	if st.Mode().Perm() != 0o600 {
		t.Errorf("config mode %v", st.Mode().Perm())
	}
	store, _ := config.Open(cfg)
	if ok, err := auth.Verify("a perfectly fine passphrase", store.Get().Auth.PasswordHash); !ok || err != nil {
		t.Errorf("stored hash must verify: %v %v", ok, err)
	}
	if strings.Contains(out.String(), "perfectly fine") {
		t.Error("must not echo the password")
	}

	// changing it keeps the rest of the config
	_ = store.Update(func(c *config.Config) error { c.SelectedContext = "keep-me"; return nil })
	if code := runPasswd([]string{"--config", cfg}, pipeIn(t, "another long passphrase\n"), &out, &errOut); code != 0 {
		t.Fatalf("change: %s", errOut.String())
	}
	store2, _ := config.Open(cfg)
	if store2.Get().SelectedContext != "keep-me" {
		t.Error("other settings survive")
	}
	if ok, _ := auth.Verify("another long passphrase", store2.Get().Auth.PasswordHash); !ok {
		t.Error("new password")
	}
}

func TestPasswdRefusesWeakAndMissing(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	for name, in := range map[string]string{"short": "short\n", "blank": "          \n", "empty": ""} {
		var out, errOut bytes.Buffer
		if code := runPasswd([]string{"--config", cfg}, pipeIn(t, in), &out, &errOut); code == 0 {
			t.Errorf("%s must fail", name)
		}
	}
	if _, err := os.Stat(cfg); err == nil {
		t.Error("a refused password must not create or change the config")
	}
	var out, errOut bytes.Buffer
	if code := runPasswd([]string{"--bogus"}, pipeIn(t, ""), &out, &errOut); code != 2 {
		t.Errorf("unknown flag: %d", code)
	}
	if code := runPasswd([]string{"--config"}, pipeIn(t, ""), &out, &errOut); code != 2 {
		t.Errorf("missing value: %d", code)
	}
}
