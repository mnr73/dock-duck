// Package projects merges discovered directories with Docker's view of
// Compose projects (`docker compose ls -a`).
package projects

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"dockduck/internal/config"
	"dockduck/internal/discovery"
	"dockduck/internal/dockercli"
)

// Project is one Compose project as shown in the sidebar.
type Project struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Dir     string   `json:"dir"`
	Files   []string `json:"files"`
	EnvFile string   `json:"env_file,omitempty"`
	Root    string   `json:"root,omitempty"` // empty: known only to Docker (outside scanned roots)
	Status  string   `json:"status"`         // running | partial | stopped | unknown
	Running int      `json:"running"`
	Total   int      `json:"total"`
}

// Service lists projects. Scan results are cached until Rescan.
type Service struct {
	Docker dockercli.Runner
	Store  *config.Store

	mu     sync.Mutex
	scan   []discovery.Found
	loaded bool
}

type lsEntry struct {
	Name        string
	Status      string
	ConfigFiles string
}

// Rescan walks all roots again and returns how many projects were found.
func (s *Service) Rescan(ctx context.Context) (int, error) {
	cfg := s.Store.Get()
	seen := map[string]bool{}
	var all []discovery.Found
	for _, r := range cfg.Roots {
		found, err := discovery.Scan(ctx, r.Path, cfg.Settings.ScanDepth)
		if err != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		for _, f := range found { // a broken root must not hide the others
			if !seen[f.Dir] {
				seen[f.Dir] = true
				all = append(all, f)
			}
		}
	}
	s.mu.Lock()
	s.scan, s.loaded = all, true
	s.mu.Unlock()
	return len(all), nil
}

// List returns scanned projects merged with live state in docker context ctxName.
func (s *Service) List(ctx context.Context, ctxName string) ([]Project, error) {
	s.mu.Lock()
	loaded := s.loaded
	s.mu.Unlock()
	if !loaded {
		if _, err := s.Rescan(ctx); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	scan := append([]discovery.Found(nil), s.scan...)
	s.mu.Unlock()

	// Docker's view is best-effort: an unreachable context still shows projects.
	var ls []lsEntry
	if b, err := s.Docker.Output(ctx, dockercli.Scoped(ctxName, "compose", "ls", "-a", "--format", "json")...); err == nil {
		ls, _ = dockercli.DecodeLines[lsEntry](b)
	}
	byDir := map[string]lsEntry{}
	for _, e := range ls {
		if files := splitFiles(e.ConfigFiles); len(files) > 0 {
			byDir[filepath.Dir(files[0])] = e
		}
	}

	var out []Project
	used := map[string]bool{}
	for _, f := range scan {
		p := Project{ID: config.RootID(f.Dir), Dir: f.Dir, Files: f.Files, EnvFile: f.EnvFile,
			Root: f.Root, Name: DefaultName(f.Dir), Status: "unknown"}
		if e, ok := byDir[f.Dir]; ok {
			used[f.Dir] = true
			p.Name = e.Name
			p.Files = splitFiles(e.ConfigFiles)
			p.Running, p.Total, p.Status = parseStatus(e.Status)
		} else if ls != nil {
			p.Status = "stopped"
		}
		out = append(out, p)
	}
	for dir, e := range byDir { // started elsewhere, outside scanned roots
		if used[dir] {
			continue
		}
		p := Project{ID: config.RootID(dir), Name: e.Name, Dir: dir, Files: splitFiles(e.ConfigFiles)}
		p.Running, p.Total, p.Status = parseStatus(e.Status)
		if f, ok := discovery.InspectDir(dir); ok {
			p.EnvFile = f.EnvFile
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

var statusRe = regexp.MustCompile(`([a-z]+)\((\d+)\)`)

// parseStatus converts "running(2), exited(1)" into counts and a summary.
func parseStatus(s string) (running, total int, status string) {
	for _, m := range statusRe.FindAllStringSubmatch(s, -1) {
		n, _ := strconv.Atoi(m[2])
		total += n
		if m[1] == "running" {
			running += n
		}
	}
	switch {
	case total == 0:
		return 0, 0, "stopped"
	case running == total:
		return running, total, "running"
	case running > 0:
		return running, total, "partial"
	}
	return 0, total, "stopped"
}

var nonName = regexp.MustCompile(`[^a-z0-9_-]`)

// DefaultName mimics Compose's default project name for a directory.
func DefaultName(dir string) string {
	return strings.TrimLeft(nonName.ReplaceAllString(strings.ToLower(filepath.Base(dir)), ""), "_-")
}

func splitFiles(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
