package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"dockduck/internal/dockercli"
)

// Port is a published (or configured) port mapping.
type Port struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      int    `json:"host_port,omitempty"`
	ContainerPort int    `json:"container_port"`
	Proto         string `json:"proto"`
}

// Container is one container of a service.
type Container struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	State    string `json:"state"`  // docker state: running, exited, restarting, ...
	Status   string `json:"status"` // human text: "Up 2 hours (healthy)"
	Health   string `json:"health,omitempty"`
	ExitCode int    `json:"exit_code"`
	Image    string `json:"image"`
	Ports    []Port `json:"ports"`
}

// ComposeService is a Compose service with its containers.
type ComposeService struct {
	Name       string      `json:"name"`
	Image      string      `json:"image"`
	Status     string      `json:"status"` // running|starting|restarting|unhealthy|paused|exited|error|stopped
	Health     string      `json:"health,omitempty"`
	Containers []Container `json:"containers"`
	Ports      []Port      `json:"ports"`
	// Buildable: the service has a `build:` section (`compose build` applies, and
	// `compose pull` would fail for its locally built image). Unknown (false) when
	// `compose config` could not be resolved.
	Buildable bool `json:"buildable"`
}

// Detail is a project with its services.
type Detail struct {
	Project
	Services    []ComposeService `json:"services"`
	ConfigError string           `json:"config_error,omitempty"` // `compose config` failed; services come from containers only
}

type psRow struct {
	ID         string
	Name       string
	Service    string
	Image      string
	State      string
	Status     string
	Health     string
	ExitCode   int
	Publishers []struct {
		URL           string
		TargetPort    int
		PublishedPort int
		Protocol      string
	}
}

type composeConfig struct {
	Name     string
	Services map[string]struct {
		Image string
		Build any // present (object or string) when the service is built from source
		Ports []struct {
			Target    int
			Published any // string or number
			Protocol  string
			HostIP    string `json:"host_ip"`
		}
	}
}

// ComposeArgs builds `[--context X] compose --project-directory … [-p …] -f … <extra…>`.
func ComposeArgs(ctxName string, p Project, extra ...string) []string {
	a := []string{"compose", "--project-directory", p.Dir}
	if p.DockerName {
		a = append(a, "-p", p.Name)
	}
	for _, f := range p.Files {
		a = append(a, "-f", f)
	}
	a = append(a, "--ansi", "never", "--progress", "plain")
	return dockercli.Scoped(ctxName, append(a, extra...)...)
}

// Detail loads services for a project: `compose config` (declared services)
// merged with `compose ps -a` (what exists).
func (s *Service) Detail(ctx context.Context, ctxName string, p Project) (Detail, error) {
	d := Detail{Project: p, Services: []ComposeService{}}

	psOut, err := s.Docker.Output(ctx, ComposeArgs(ctxName, p, "ps", "-a", "--format", "json")...)
	if err != nil {
		return d, err
	}
	rows, err := dockercli.DecodeLines[psRow](psOut)
	if err != nil {
		return d, fmt.Errorf("parse compose ps: %w", err)
	}

	var cfg composeConfig
	cfgOut, cerr := s.Docker.Output(ctx, ComposeArgs(ctxName, p, "config", "--format", "json")...)
	if cerr == nil {
		cerr = json.Unmarshal(cfgOut, &cfg)
	}
	if cerr != nil {
		d.ConfigError = lastLines(cerr.Error(), 3)
	} else if !p.DockerName && cfg.Name != "" {
		d.Name = cfg.Name
	}

	d.Services = BuildServices(cfg, rows)
	d.Running, d.Total = 0, 0
	for _, svc := range d.Services {
		for _, c := range svc.Containers {
			d.Total++
			if c.State == "running" {
				d.Running++
			}
		}
	}
	d.Status = summarize(d.Running, d.Total)
	return d, nil
}

func summarize(running, total int) string {
	switch {
	case total == 0:
		return "stopped"
	case running == total:
		return "running"
	case running > 0:
		return "partial"
	}
	return "stopped"
}

// BuildServices merges declared services with existing containers.
func BuildServices(cfg composeConfig, rows []psRow) []ComposeService {
	by := map[string]*ComposeService{}
	for name, c := range cfg.Services {
		sv := &ComposeService{Name: name, Image: c.Image, Buildable: c.Build != nil, Containers: []Container{}, Ports: []Port{}}
		for _, p := range c.Ports {
			sv.Ports = append(sv.Ports, Port{HostIP: p.HostIP, HostPort: toInt(p.Published), ContainerPort: p.Target, Proto: p.Protocol})
		}
		by[name] = sv
	}
	runtimePorts := map[string][]Port{}
	for _, r := range rows {
		sv, ok := by[r.Service]
		if !ok { // container of a service no longer in the file
			sv = &ComposeService{Name: r.Service, Image: r.Image, Containers: []Container{}, Ports: []Port{}}
			by[r.Service] = sv
		}
		c := Container{ID: r.ID, Name: r.Name, State: r.State, Status: r.Status, Health: r.Health,
			ExitCode: r.ExitCode, Image: r.Image, Ports: []Port{}}
		seen := map[string]bool{}
		for _, pb := range r.Publishers {
			if pb.PublishedPort == 0 {
				continue
			}
			k := fmt.Sprintf("%d/%d/%s", pb.PublishedPort, pb.TargetPort, pb.Protocol)
			if seen[k] {
				continue // IPv4 + IPv6 duplicate
			}
			seen[k] = true
			c.Ports = append(c.Ports, Port{HostIP: pb.URL, HostPort: pb.PublishedPort, ContainerPort: pb.TargetPort, Proto: pb.Protocol})
		}
		runtimePorts[r.Service] = append(runtimePorts[r.Service], c.Ports...)
		sv.Containers = append(sv.Containers, c)
		if sv.Image == "" {
			sv.Image = r.Image
		}
	}
	out := make([]ComposeService, 0, len(by))
	for name, sv := range by {
		if rp := runtimePorts[name]; len(rp) > 0 {
			sv.Ports = rp
		}
		sv.Status, sv.Health = serviceStatus(sv.Containers)
		out = append(out, *sv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// serviceStatus folds container states into the service-level indicator used by the UI.
func serviceStatus(cs []Container) (status, health string) {
	if len(cs) == 0 {
		return "stopped", ""
	}
	var running, restarting, paused, errored, exited int
	for _, c := range cs {
		switch c.State {
		case "running":
			running++
			switch c.Health {
			case "unhealthy":
				health = "unhealthy"
			case "starting":
				if health != "unhealthy" {
					health = "starting"
				}
			case "healthy":
				if health == "" {
					health = "healthy"
				}
			}
		case "restarting":
			restarting++
		case "paused":
			paused++
		case "exited", "dead":
			if c.ExitCode != 0 {
				errored++
			} else {
				exited++
			}
		}
	}
	switch {
	case restarting > 0:
		return "restarting", health
	case health == "unhealthy":
		return "unhealthy", health
	case health == "starting":
		return "starting", health
	case running > 0:
		return "running", health
	case paused > 0:
		return "paused", ""
	case errored > 0:
		return "error", ""
	case exited > 0:
		return "exited", ""
	}
	return "stopped", "" // created, removing, …
}

// ---------------------------------------------------------------- actions ----

// ActionOpts tune a project action.
type ActionOpts struct {
	Services      []string `json:"services"`
	Volumes       bool     `json:"volumes"`        // down -v (destroys named volumes!)
	RemoveOrphans bool     `json:"remove_orphans"` // up/down
	Build         bool     `json:"build"`          // up --build
	NoCache       bool     `json:"no_cache"`       // build --no-cache
	Pull          bool     `json:"pull"`           // build --pull (refresh base images)
}

// ErrBadAction is returned for unknown actions or invalid options.
var ErrBadAction = errors.New("invalid action")

// ActionArgs validates and builds the docker argv for a project action.
func ActionArgs(ctxName string, p Project, action string, o ActionOpts) ([]string, error) {
	for _, sv := range o.Services {
		if !dockercli.ValidName(sv) {
			return nil, fmt.Errorf("%w: bad service name %q", ErrBadAction, sv)
		}
	}
	var extra []string
	switch action {
	case "up":
		extra = []string{"up", "-d"}
		if o.Build {
			extra = append(extra, "--build")
		}
		if o.RemoveOrphans {
			extra = append(extra, "--remove-orphans")
		}
	case "down":
		if len(o.Services) > 0 {
			return nil, fmt.Errorf("%w: down applies to the whole project", ErrBadAction)
		}
		extra = []string{"down"}
		if o.Volumes {
			extra = append(extra, "-v")
		}
		if o.RemoveOrphans {
			extra = append(extra, "--remove-orphans")
		}
	case "start", "stop", "restart":
		extra = []string{action}
	case "pull":
		extra = []string{"pull"}
		if len(o.Services) == 0 {
			// whole project: images built locally have no registry to pull from
			extra = append(extra, "--ignore-buildable")
		}
	case "build":
		extra = []string{"build"}
		if o.NoCache {
			extra = append(extra, "--no-cache")
		}
		if o.Pull {
			extra = append(extra, "--pull")
		}
	default:
		return nil, fmt.Errorf("%w: %q", ErrBadAction, action)
	}
	extra = append(extra, o.Services...)
	return ComposeArgs(ctxName, p, extra...), nil
}

// ------------------------------------------------------------------ stats ----

// Usage is the resource usage of one container.
type Usage struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   string  `json:"mem_usage"` // "464.7MiB / 30.81GiB"
	MemBytes   int64   `json:"mem_bytes"` // used memory, for summing per service
	MemPercent float64 `json:"mem_percent"`
}

type statRow struct {
	ID       string
	CPUPerc  string
	MemUsage string
	MemPerc  string
}

type statsCache struct {
	mu sync.Mutex
	m  map[string]statsEntry
}

type statsEntry struct {
	at  time.Time
	val map[string]Usage
}

// StatsTTL bounds how often `docker stats` is executed per project.
const StatsTTL = 3 * time.Second

// Stats returns usage keyed by container id (short, 12 chars), cached for StatsTTL.
func (s *Service) Stats(ctx context.Context, ctxName string, p Project, containerIDs []string) (map[string]Usage, error) {
	key := ctxName + "|" + p.ID
	s.stats.mu.Lock()
	if e, ok := s.stats.m[key]; ok && time.Since(e.at) < StatsTTL {
		s.stats.mu.Unlock()
		return e.val, nil
	}
	s.stats.mu.Unlock()

	out := map[string]Usage{}
	if len(containerIDs) > 0 {
		args := dockercli.Scoped(ctxName, "stats", "--no-stream", "--format", "json")
		for _, id := range containerIDs {
			if !isHex(id) {
				return nil, fmt.Errorf("bad container id %q", id)
			}
			args = append(args, id)
		}
		b, err := s.Docker.Output(ctx, args...)
		if err != nil {
			return nil, err
		}
		rows, err := dockercli.DecodeLines[statRow](b)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			used, _, _ := strings.Cut(r.MemUsage, "/")
			out[r.ID] = Usage{CPUPercent: pct(r.CPUPerc), MemUsage: r.MemUsage, MemBytes: parseSize(used), MemPercent: pct(r.MemPerc)}
		}
	}
	s.stats.mu.Lock()
	if s.stats.m == nil {
		s.stats.m = map[string]statsEntry{}
	}
	s.stats.m[key] = statsEntry{time.Now(), out}
	s.stats.mu.Unlock()
	return out, nil
}

func pct(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return f
}

func isHex(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

var sizeUnits = map[string]float64{
	"b": 1, "kb": 1e3, "mb": 1e6, "gb": 1e9, "tb": 1e12,
	"kib": 1 << 10, "mib": 1 << 20, "gib": 1 << 30, "tib": 1 << 40,
}

// parseSize converts docker's "464.7MiB" / "12kB" strings to bytes (0 if unknown).
func parseSize(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	mult, ok := sizeUnits[strings.TrimSpace(s[i:])]
	if err != nil || !ok {
		return 0
	}
	return int64(n * mult)
}
