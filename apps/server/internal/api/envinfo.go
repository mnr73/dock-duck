package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dockduck/internal/dockercli"
	"dockduck/internal/files"
	"dockduck/internal/projects"
)

// envExample reports the keys of `.env.example` (or `.env.sample`) and which of
// them the current `.env` lacks, so the UI can offer to add them.
func (h *dockerHandlers) envExample(p projects.Project) map[string]any {
	for _, name := range []string{".env.example", ".env.sample", "env.example"} {
		b, err := os.ReadFile(filepath.Join(p.Dir, name))
		if err != nil || len(b) > files.MaxSize {
			continue
		}
		have := map[string]bool{}
		if cur, err := os.ReadFile(filepath.Join(p.Dir, ".env")); err == nil {
			for _, k := range files.Keys(string(cur)) {
				have[k] = true
			}
		}
		var missing []map[string]any
		for _, l := range files.ParseEnv(string(b)) {
			if l.Kind == "var" && !have[l.Key] {
				missing = append(missing, map[string]any{"key": l.Key, "value": exampleValue(l)})
			}
		}
		return map[string]any{"name": name, "count": len(files.Keys(string(b))), "missing": missing}
	}
	return nil
}

// exampleValue never leaks a real-looking secret from an example file.
func exampleValue(l files.EnvLine) string {
	if l.Secret {
		return ""
	}
	return l.Value
}

// envUsage tells which services load the .env via `env_file:` and which
// `${VARS}` the Compose files interpolate (best effort; empty on failure).
func (h *dockerHandlers) envUsage(r *http.Request, ctxName string, p projects.Project) map[string]any {
	out := map[string]any{"services": []string{}, "interpolated": []string{}}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	b, err := h.Projects.Docker.Output(ctx, projects.ComposeArgs(ctxName, p, "config", "--no-interpolate", "--format", "json")...)
	if err != nil {
		return out
	}
	var cfg struct {
		Services map[string]struct {
			EnvFile any `json:"env_file"`
		}
	}
	if json.Unmarshal(b, &cfg) != nil {
		return out
	}
	var svcs []string
	for name, s := range cfg.Services {
		if usesDotEnv(s.EnvFile) {
			svcs = append(svcs, name)
		}
	}
	sort.Strings(svcs)
	out["services"] = svcs
	out["interpolated"] = interpolatedVars(string(b))
	return out
}

// usesDotEnv: env_file may be a string, a list of strings or a list of {path: ...}.
func usesDotEnv(v any) bool {
	var paths []string
	switch x := v.(type) {
	case string:
		paths = append(paths, x)
	case []any:
		for _, e := range x {
			switch y := e.(type) {
			case string:
				paths = append(paths, y)
			case map[string]any:
				if s, ok := y["path"].(string); ok {
					paths = append(paths, s)
				}
			}
		}
	}
	for _, p := range paths {
		if filepath.Base(p) == ".env" {
			return true
		}
	}
	return false
}

// interpolatedVars finds ${NAME} / $NAME references in the (non-interpolated) config text.
func interpolatedVars(text string) []string {
	seen := map[string]bool{}
	var out []string
	for i := 0; i < len(text)-2; i++ {
		if text[i] != '$' {
			continue
		}
		j := i + 1
		brace := text[j] == '{'
		if brace {
			j++
		}
		k := j
		for k < len(text) && (text[k] == '_' || text[k] >= 'A' && text[k] <= 'Z' || text[k] >= 'a' && text[k] <= 'z' || k > j && text[k] >= '0' && text[k] <= '9') {
			k++
		}
		if name := text[j:k]; name != "" && dockercli.ValidName(name) && !strings.HasPrefix(name, "_") && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
