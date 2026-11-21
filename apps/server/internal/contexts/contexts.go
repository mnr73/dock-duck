// Package contexts lists and switches Docker contexts.
//
// Contexts are never created, removed or edited. "Switching" selects the
// context DockDuck passes via --context; the user's CLI default is untouched
// unless Settings.SyncCLI is on.
package contexts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"dockduck/internal/config"
	"dockduck/internal/dockercli"
)

// ErrNotFound is returned when a context name is unknown.
var ErrNotFound = errors.New("docker context not found")

// Docker is what the service needs from the docker layer.
type Docker interface {
	dockercli.Runner
	UseContext(ctx context.Context, name string) error
}

// Info describes one context for the UI.
type Info struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
	Current     bool   `json:"current"`  // the Docker CLI's active context
	Selected    bool   `json:"selected"` // the context DockDuck uses
	Available   bool   `json:"available"`
	Version     string `json:"version,omitempty"` // engine version when available
	Error       string `json:"error,omitempty"`
}

// Service implements context operations.
type Service struct {
	Docker      Docker
	Store       *config.Store
	PingTimeout time.Duration
}

type rawContext struct {
	Name           string
	Description    string
	DockerEndpoint string
	Current        bool
	Error          string
}

// List returns all contexts with availability, pinging them concurrently.
func (s *Service) List(ctx context.Context) ([]Info, error) {
	raw, err := s.raw(ctx)
	if err != nil {
		return nil, err
	}
	selected := s.selectedFrom(raw)
	out := make([]Info, len(raw))
	var wg sync.WaitGroup
	for i, r := range raw {
		out[i] = Info{Name: r.Name, Description: r.Description, Endpoint: r.DockerEndpoint,
			Current: r.Current, Selected: r.Name == selected, Error: r.Error}
		if !dockercli.ValidName(r.Name) {
			out[i].Error = "unsupported context name"
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			ver, err := s.ping(ctx, r.Name)
			if err != nil {
				out[i].Error = firstLine(err.Error())
				return
			}
			out[i].Available, out[i].Version, out[i].Error = true, ver, ""
		}()
	}
	wg.Wait()
	return out, nil
}

// Selected returns the name of the context DockDuck should use now.
func (s *Service) Selected(ctx context.Context) (string, error) {
	raw, err := s.raw(ctx)
	if err != nil {
		return "", err
	}
	return s.selectedFrom(raw), nil
}

// Select switches the app-level context (and the CLI default if SyncCLI is on).
func (s *Service) Select(ctx context.Context, name string) error {
	if !dockercli.ValidName(name) {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	raw, err := s.raw(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, r := range raw {
		found = found || r.Name == name
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	if err := s.Store.Update(func(c *config.Config) error { c.SelectedContext = name; return nil }); err != nil {
		return err
	}
	if s.Store.Get().Settings.SyncCLI {
		return s.Docker.UseContext(ctx, name)
	}
	return nil
}

func (s *Service) raw(ctx context.Context) ([]rawContext, error) {
	b, err := s.Docker.Output(ctx, "context", "ls", "--format", "json")
	if err != nil {
		return nil, err
	}
	return dockercli.DecodeLines[rawContext](b)
}

// selectedFrom: saved selection → default setting → CLI current → first.
func (s *Service) selectedFrom(raw []rawContext) string {
	cfg := s.Store.Get()
	has := func(n string) bool {
		for _, r := range raw {
			if r.Name == n {
				return true
			}
		}
		return false
	}
	for _, n := range []string{cfg.SelectedContext, cfg.Settings.DefaultContext} {
		if n != "" && has(n) {
			return n
		}
	}
	for _, r := range raw {
		if r.Current {
			return r.Name
		}
	}
	if len(raw) > 0 {
		return raw[0].Name
	}
	return ""
}

func (s *Service) ping(ctx context.Context, name string) (string, error) {
	d := s.PingTimeout
	if d <= 0 {
		d = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	b, err := s.Docker.Output(ctx, dockercli.Scoped(name, "version", "--format", "{{.Server.Version}}")...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
