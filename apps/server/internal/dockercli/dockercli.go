// Package dockercli is the only place that executes the `docker` binary.
//
// Safety rules enforced here (see AGENTS.md):
//   - arguments are always an argv slice, never a shell string;
//   - `docker context create/rm/update/import/...` are refused; only
//     ls/list/inspect/show are allowed through Output. Switching the CLI's
//     default context is possible solely via Client.UseContext.
package dockercli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Runner runs docker with the given arguments and returns stdout.
type Runner interface {
	Output(ctx context.Context, args ...string) ([]byte, error)
}

// RunnerFunc adapts a function to Runner (handy for tests).
type RunnerFunc func(ctx context.Context, args ...string) ([]byte, error)

// Output implements Runner.
func (f RunnerFunc) Output(ctx context.Context, args ...string) ([]byte, error) {
	return f(ctx, args...)
}

// ErrForbidden is returned for docker subcommands DockDuck must never run.
var ErrForbidden = errors.New("dockercli: forbidden docker subcommand")

// ExitError is a failed docker invocation with its captured stderr.
type ExitError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *ExitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		msg = e.Err.Error()
	}
	return "docker " + strings.Join(e.Args, " ") + ": " + msg
}

func (e *ExitError) Unwrap() error { return e.Err }

// Client executes the real docker binary.
type Client struct {
	Bin     string        // default "docker"
	Timeout time.Duration // per-call default when ctx has no deadline; default 30s
}

// New returns a Client using `docker` from PATH.
func New() *Client { return &Client{Bin: "docker", Timeout: 30 * time.Second} }

// Output implements Runner.
func (c *Client) Output(ctx context.Context, args ...string) ([]byte, error) {
	if err := guard(args); err != nil {
		return nil, err
	}
	return c.run(ctx, args...)
}

// UseContext runs `docker context use <name>`: the single sanctioned context
// mutation, only called when the user enabled "sync with Docker CLI".
func (c *Client) UseContext(ctx context.Context, name string) error {
	if !ValidName(name) {
		return fmt.Errorf("invalid context name %q", name)
	}
	_, err := c.run(ctx, "context", "use", name)
	return err
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok && c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	bin := c.Bin
	if bin == "" {
		bin = "docker"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return stdout.Bytes(), &ExitError{Args: args, Stderr: stderr.String(), Err: err}
	}
	return stdout.Bytes(), nil
}

var allowedContextSubcommands = map[string]bool{"ls": true, "list": true, "inspect": true, "show": true}

func guard(args []string) error {
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--context" || a == "-c" || a == "--host" || a == "-H" {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		pos = append(pos, a)
		if len(pos) == 2 {
			break
		}
	}
	if len(pos) >= 1 && pos[0] == "context" {
		if len(pos) < 2 || !allowedContextSubcommands[pos[1]] {
			return fmt.Errorf("%w: docker %s", ErrForbidden, strings.Join(args, " "))
		}
	}
	return nil
}

// Scoped prefixes args with `--context name` (when name is non-empty).
func Scoped(contextName string, args ...string) []string {
	if contextName == "" {
		return args
	}
	return append([]string{"--context", contextName}, args...)
}

// ValidName reports whether s is a safe docker context/project/service/user token.
func ValidName(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for i, r := range s {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if i > 0 {
			ok = ok || r == '_' || r == '.' || r == '-'
		}
		if !ok {
			return false
		}
	}
	return true
}

// DecodeLines decodes docker's `--format json` output, which is either NDJSON
// (one object per line) or a single JSON array, depending on the command.
func DecodeLines[T any](b []byte) ([]T, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, nil
	}
	if b[0] == '[' {
		var out []T
		return out, json.Unmarshal(b, &out)
	}
	var out []T
	dec := json.NewDecoder(bytes.NewReader(b))
	for dec.More() {
		var v T
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
