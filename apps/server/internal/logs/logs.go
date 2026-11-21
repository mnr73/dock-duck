// Package logs builds and parses `docker compose logs` output.
package logs

import (
	"fmt"
	"strings"
	"time"

	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
)

// Line is one log line of one container.
type Line struct {
	Service string `json:"s"`
	Time    string `json:"t,omitempty"` // RFC3339Nano as printed by docker
	Text    string `json:"m"`
}

// Options select what to read.
type Options struct {
	Services []string
	Tail     int    // number of past lines per container; 0 = none, <0 = all
	Since    string // "" or a docker --since value (e.g. 15m, 2026-11-21T10:00:00Z)
	Follow   bool
}

// MaxTail bounds how much history a single request can pull.
const MaxTail = 20000

// Args builds the docker argv for `compose logs`.
func Args(ctxName string, p projects.Project, o Options) ([]string, error) {
	for _, s := range o.Services {
		if !dockercli.ValidName(s) {
			return nil, fmt.Errorf("invalid service name %q", s)
		}
	}
	if o.Since != "" && !validSince(o.Since) {
		return nil, fmt.Errorf("invalid since %q", o.Since)
	}
	extra := []string{"logs", "--timestamps", "--no-color"}
	switch {
	case o.Tail < 0:
		// everything
	default:
		t := o.Tail
		if t > MaxTail {
			t = MaxTail
		}
		extra = append(extra, "--tail", fmt.Sprint(t))
	}
	if o.Since != "" {
		extra = append(extra, "--since", o.Since)
	}
	if o.Follow {
		extra = append(extra, "-f")
	}
	extra = append(extra, o.Services...)
	return projects.ComposeArgs(ctxName, p, extra...), nil
}

// validSince accepts a Go-style duration ("15m", "1h30m") or an RFC3339 time.
func validSince(s string) bool {
	if _, err := time.ParseDuration(s); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil
}

// Parse splits `service  | 2026-…Z message`. Compose pads the service name to a
// common width and prints `|` after it. Lines that do not fit (docker's own
// notices, or a continuation without prefix) keep Service empty.
func Parse(raw string) Line {
	name, rest, ok := strings.Cut(raw, "|")
	if !ok {
		return Line{Text: raw}
	}
	svc := strings.TrimSpace(name)
	// service names are [A-Za-z0-9_.-]; anything else means the "|" is part of the message
	if svc == "" || !dockercli.ValidName(strings.TrimSuffix(svc, " ")) || strings.ContainsAny(svc, " \t") {
		return Line{Text: raw}
	}
	rest = strings.TrimPrefix(rest, " ")
	ts, msg, found := strings.Cut(rest, " ")
	if !found || !looksLikeTime(ts) {
		return Line{Service: serviceFromContainer(svc), Text: rest}
	}
	return Line{Service: serviceFromContainer(svc), Time: ts, Text: msg}
}

func looksLikeTime(s string) bool {
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil
}

// serviceFromContainer: with a single replica compose prints the container
// name (`shivaerp-laravel`), with several `proj-svc-1`. The API maps these back
// to services using the project's detail; this keeps the raw label.
func serviceFromContainer(label string) string { return label }
