package files

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"dockduck/internal/dockercli"
)

// Problem is one validation message, with a line when Compose reported one.
type Problem struct {
	Severity string `json:"severity"` // "error" | "warning"
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

// Validation is the outcome of `docker compose config -q` on proposed content.
type Validation struct {
	OK       bool      `json:"ok"`
	Problems []Problem `json:"problems"`
}

// StdinRunner runs docker with stdin content (the proposed file).
type StdinRunner interface {
	OutputStdin(ctx context.Context, stdin string, args ...string) (stdout, stderr []byte, err error)
}

var (
	yamlLine   = regexp.MustCompile(`(?i)\byaml: (?:line (\d+)|unmarshal errors:\s*line (\d+))`)
	linePrefix = regexp.MustCompile(`^line \d+: `)
	warnLine   = regexp.MustCompile(`level=warning msg="?(.*?)"?$`)
)

// ValidateCompose checks content as the file at index `at` of the project's
// Compose files: every other file is used as-is from disk, the edited one is
// fed through stdin, so nothing is written before the user saves. The project
// directory keeps relative paths, .env interpolation and env_file working.
func ValidateCompose(ctx context.Context, r StdinRunner, ctxName, dir string, files []string, at int, content string) (Validation, error) {
	args := []string{"compose", "--project-directory", dir}
	for i, f := range files {
		if i == at {
			args = append(args, "-f", "-")
		} else {
			args = append(args, "-f", f)
		}
	}
	args = append(args, "config", "-q")
	_, stderr, err := r.OutputStdin(ctx, content, dockercli.Scoped(ctxName, args...)...)
	return parseValidation(string(stderr), err), nil
}

// parseValidation turns Compose's stderr (and the exit error, if any) into problems.
func parseValidation(stderr string, err error) Validation {
	v := Validation{OK: true, Problems: []Problem{}}
	var ee *dockercli.ExitError
	failed := err != nil
	isExit := errors.As(err, &ee)
	text := stderr
	if failed && !isExit {
		text = err.Error() // timeout, docker missing, ...
	}
	hardFail := false
	for _, raw := range strings.Split(strings.TrimSpace(text), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "level=warning") {
			if m := warnLine.FindStringSubmatch(raw); m != nil {
				v.Problems = append(v.Problems, Problem{Severity: "warning", Message: strings.ReplaceAll(m[1], `\"`, `"`)})
				continue
			}
		}
		p := Problem{Severity: "error", Message: cleanMessage(raw)}
		if m := yamlLine.FindStringSubmatch(raw); m != nil {
			for _, g := range m[1:] {
				if n, e := strconv.Atoi(g); e == nil && n > 0 {
					p.Line = n
				}
			}
		}
		if p.Line > 0 {
			p.Message = linePrefix.ReplaceAllString(p.Message, "") // the UI shows the line separately
		}
		v.Problems = append(v.Problems, p)
		hardFail = true
	}
	// warnings alone never invalidate a file (an unset ${VAR} defaults to blank);
	// a failed run without any error line (e.g. killed) must still not look valid.
	if hardFail || (failed && !hasError(v.Problems)) {
		v.OK = false
		if !hasError(v.Problems) {
			v.Problems = append(v.Problems, Problem{Severity: "error", Message: strings.TrimSpace(err.Error())})
		}
	}
	return v
}

func hasError(ps []Problem) bool {
	for _, p := range ps {
		if p.Severity == "error" {
			return true
		}
	}
	return false
}

func cleanMessage(s string) string {
	s = strings.TrimPrefix(s, "validating -: ")
	s = strings.TrimPrefix(s, "yaml: ")
	return s
}
