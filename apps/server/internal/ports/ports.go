// Package ports lists every host port published by running containers, with
// the Compose project/service that owns it (empty for non-Compose containers).
package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"dockduck/internal/dockercli"
)

// Entry is one published port of one container. IPv4 and IPv6 bindings of the
// same mapping are merged into HostIPs.
type Entry struct {
	HostIPs       []string `json:"host_ips"`
	HostPort      int      `json:"host_port"`
	ContainerPort int      `json:"container_port"`
	Proto         string   `json:"proto"`
	Project       string   `json:"project,omitempty"`
	Service       string   `json:"service,omitempty"`
	Container     string   `json:"container"`
	ContainerID   string   `json:"container_id"`
	Image         string   `json:"image"`
}

// Service lists ports.
type Service struct{ Docker dockercli.Runner }

type inspectRow struct {
	ID     string `json:"Id"`
	Name   string
	Config struct {
		Image  string
		Labels map[string]string
	}
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string
		}
	}
}

// List returns published ports of all running containers in the docker context.
func (s *Service) List(ctx context.Context, ctxName string) ([]Entry, error) {
	b, err := s.Docker.Output(ctx, dockercli.Scoped(ctxName, "ps", "-q", "--no-trunc")...)
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(string(b))
	if len(ids) == 0 {
		return []Entry{}, nil
	}
	args := []string{"inspect"}
	for _, id := range ids {
		if !isHex(id) {
			return nil, fmt.Errorf("unexpected container id %q", id)
		}
		args = append(args, id)
	}
	out, err := s.Docker.Output(ctx, dockercli.Scoped(ctxName, args...)...)
	if err != nil {
		return nil, err
	}
	var rows []inspectRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("parse docker inspect: %w", err)
	}
	return Build(rows), nil
}

// Build turns inspect rows into merged, sorted entries.
func Build(rows []inspectRow) []Entry {
	type key struct {
		id, proto  string
		host, cont int
	}
	idx := map[key]*Entry{}
	var out []*Entry
	for _, r := range rows {
		for spec, binds := range r.NetworkSettings.Ports {
			cp, proto, ok := splitSpec(spec)
			if !ok {
				continue
			}
			for _, b := range binds {
				hp, err := strconv.Atoi(b.HostPort)
				if err != nil || hp == 0 {
					continue
				}
				k := key{r.ID, proto, hp, cp}
				e := idx[k]
				if e == nil {
					e = &Entry{
						HostPort: hp, ContainerPort: cp, Proto: proto,
						Project:   r.Config.Labels["com.docker.compose.project"],
						Service:   r.Config.Labels["com.docker.compose.service"],
						Container: strings.TrimPrefix(r.Name, "/"), ContainerID: shortID(r.ID), Image: r.Config.Image,
					}
					idx[k] = e
					out = append(out, e)
				}
				ip := b.HostIP
				if ip == "" {
					ip = "0.0.0.0"
				}
				e.HostIPs = append(e.HostIPs, ip)
			}
		}
	}
	res := make([]Entry, 0, len(out))
	for _, e := range out {
		sort.Strings(e.HostIPs)
		res = append(res, *e)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].HostPort != res[j].HostPort {
			return res[i].HostPort < res[j].HostPort
		}
		if res[i].Proto != res[j].Proto {
			return res[i].Proto < res[j].Proto
		}
		return res[i].Container < res[j].Container
	})
	return res
}

func splitSpec(s string) (port int, proto string, ok bool) {
	p, proto, found := strings.Cut(s, "/")
	if !found {
		proto = "tcp"
	}
	n, err := strconv.Atoi(p)
	return n, proto, err == nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
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
