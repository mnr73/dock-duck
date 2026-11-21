package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"dockduck/internal/dockercli"
)

// ContainerDetail is the trimmed `docker inspect` view shown in the UI.
// Environment variables are deliberately not included (they hold secrets).
type ContainerDetail struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Created       string            `json:"created"`
	Command       string            `json:"command"`
	Entrypoint    string            `json:"entrypoint,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	User          string            `json:"user,omitempty"`
	State         string            `json:"state"`
	ExitCode      int               `json:"exit_code"`
	Error         string            `json:"error,omitempty"`
	OOMKilled     bool              `json:"oom_killed"`
	StartedAt     string            `json:"started_at,omitempty"`
	FinishedAt    string            `json:"finished_at,omitempty"`
	RestartCount  int               `json:"restart_count"`
	RestartPolicy string            `json:"restart_policy"`
	Privileged    bool              `json:"privileged"`
	Health        string            `json:"health,omitempty"`
	HealthOutput  string            `json:"health_output,omitempty"`
	Mounts        []MountInfo       `json:"mounts"`
	Networks      []NetworkInfo     `json:"networks"`
	Ports         []Port            `json:"ports"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// MountInfo is a bind mount or volume.
type MountInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	RW          bool   `json:"rw"`
}

// NetworkInfo is a container's attachment to a network.
type NetworkInfo struct {
	Name    string   `json:"name"`
	IP      string   `json:"ip"`
	Aliases []string `json:"aliases,omitempty"`
}

type inspectRaw struct {
	ID      string `json:"Id"`
	Name    string
	Created string
	Path    string
	Args    []string
	Config  struct {
		Image      string
		Cmd        []string
		Entrypoint []string
		WorkingDir string
		User       string
	}
	State struct {
		Status     string
		ExitCode   int
		Error      string
		OOMKilled  bool
		StartedAt  string
		FinishedAt string
		Health     *struct {
			Status string
			Log    []struct{ Output string }
		}
	}
	RestartCount int
	HostConfig   struct {
		RestartPolicy struct {
			Name              string
			MaximumRetryCount int
		}
		Privileged bool
	}
	Mounts []struct {
		Type, Name, Source, Destination string
		RW                              bool
	}
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string
			Aliases   []string
		}
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string
		}
	}
}

// InspectContainers inspects the given container ids (must be hex).
func (s *Service) InspectContainers(ctx context.Context, ctxName string, ids []string) ([]ContainerDetail, error) {
	if len(ids) == 0 {
		return []ContainerDetail{}, nil
	}
	args := []string{"inspect"}
	for _, id := range ids {
		if !isHex(id) {
			return nil, fmt.Errorf("bad container id %q", id)
		}
		args = append(args, id)
	}
	b, err := s.Docker.Output(ctx, dockercli.Scoped(ctxName, args...)...)
	if err != nil {
		return nil, err
	}
	var raw []inspectRaw
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse docker inspect: %w", err)
	}
	out := make([]ContainerDetail, 0, len(raw))
	for _, r := range raw {
		out = append(out, toDetail(r))
	}
	return out, nil
}

func toDetail(r inspectRaw) ContainerDetail {
	d := ContainerDetail{
		ID: shortID(r.ID), Name: strings.TrimPrefix(r.Name, "/"), Image: r.Config.Image, Created: r.Created,
		Command:    strings.TrimSpace(strings.Join(append([]string{r.Path}, r.Args...), " ")),
		Entrypoint: strings.Join(r.Config.Entrypoint, " "), WorkingDir: r.Config.WorkingDir, User: r.Config.User,
		State: r.State.Status, ExitCode: r.State.ExitCode, Error: r.State.Error, OOMKilled: r.State.OOMKilled,
		RestartCount: r.RestartCount, Privileged: r.HostConfig.Privileged,
		Mounts: []MountInfo{}, Networks: []NetworkInfo{}, Ports: []Port{},
	}
	if !strings.HasPrefix(r.State.StartedAt, "0001-") {
		d.StartedAt = r.State.StartedAt
	}
	if !strings.HasPrefix(r.State.FinishedAt, "0001-") {
		d.FinishedAt = r.State.FinishedAt
	}
	d.RestartPolicy = r.HostConfig.RestartPolicy.Name
	if d.RestartPolicy == "" {
		d.RestartPolicy = "no"
	}
	if h := r.State.Health; h != nil {
		d.Health = h.Status
		if n := len(h.Log); n > 0 {
			d.HealthOutput = truncate(strings.TrimSpace(h.Log[n-1].Output), 500)
		}
	}
	for _, m := range r.Mounts {
		d.Mounts = append(d.Mounts, MountInfo{Type: m.Type, Name: m.Name, Source: m.Source, Destination: m.Destination, RW: m.RW})
	}
	for name, n := range r.NetworkSettings.Networks {
		d.Networks = append(d.Networks, NetworkInfo{Name: name, IP: n.IPAddress, Aliases: n.Aliases})
	}
	sort.Slice(d.Networks, func(i, j int) bool { return d.Networks[i].Name < d.Networks[j].Name })
	seen := map[string]bool{}
	for spec, binds := range r.NetworkSettings.Ports {
		cp, proto, ok := strings.Cut(spec, "/")
		if !ok {
			proto = "tcp"
		}
		for _, b := range binds {
			p := Port{HostIP: b.HostIP, HostPort: toInt(b.HostPort), ContainerPort: toInt(cp), Proto: proto}
			k := fmt.Sprint(p.HostPort, "/", p.ContainerPort, "/", proto)
			if p.HostPort == 0 || seen[k] {
				continue
			}
			seen[k] = true
			d.Ports = append(d.Ports, p)
		}
	}
	sort.Slice(d.Ports, func(i, j int) bool { return d.Ports[i].HostPort < d.Ports[j].HostPort })
	return d
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
