// Package discovery finds Compose projects below configured root directories.
package discovery

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Found is a directory that contains a Compose project.
type Found struct {
	Dir     string   `json:"dir"`
	Files   []string `json:"files"`              // absolute, in Compose merge order
	EnvFile string   `json:"env_file,omitempty"` // <dir>/.env when it exists
	Root    string   `json:"root"`
}

// Candidate file names in Compose's own precedence order for the base file.
var baseNames = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// Override file names Compose merges automatically with the base file.
var overrideNames = map[string][]string{
	"compose.yaml":        {"compose.override.yaml", "compose.override.yml"},
	"compose.yml":         {"compose.override.yaml", "compose.override.yml"},
	"docker-compose.yaml": {"docker-compose.override.yaml", "docker-compose.override.yml"},
	"docker-compose.yml":  {"docker-compose.override.yaml", "docker-compose.override.yml"},
}

var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true, "target": true,
	"__pycache__": true, "venv": true, ".venv": true,
}

// Scan walks root up to maxDepth levels (root itself is depth 0). It skips
// hidden directories, common dependency/output directories and symlinks, and
// does not descend into a directory once it is a project.
func Scan(ctx context.Context, root string, maxDepth int) ([]Found, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var out []Found
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			if path == root {
				return err
			}
			return fs.SkipDir // unreadable subtree
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			name := d.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return fs.SkipDir
			}
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(filepath.Separator))+1 > maxDepth {
				return fs.SkipDir
			}
		}
		if f, ok := inspectDir(path); ok {
			f.Root = root
			out = append(out, f)
			return fs.SkipDir
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out, err
}

// envComposeFiles reads COMPOSE_FILE from <dir>/.env, the way Compose does: paths are
// relative to the project directory and joined by COMPOSE_PATH_SEPARATOR (default ":").
// Compose ignores COMPOSE_FILE as soon as a -f flag is given, and DockDuck always passes
// -f, so the files named here have to be passed explicitly or they would never load
// (a dev override such as compose.dev.yml silently dropped).
func envComposeFiles(dir string) []string {
	f, err := os.Open(filepath.Join(dir, ".env"))
	if err != nil {
		return nil
	}
	defer f.Close()
	vars := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sc.Text()), "export "))
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || (k != "COMPOSE_FILE" && k != "COMPOSE_PATH_SEPARATOR") {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		} else if i := strings.Index(v, " #"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		vars[k] = v
	}
	list := vars["COMPOSE_FILE"]
	if list == "" {
		return nil
	}
	sep := vars["COMPOSE_PATH_SEPARATOR"]
	if sep == "" {
		sep = string(os.PathListSeparator)
	}
	var out []string
	for _, name := range strings.Split(list, sep) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !filepath.IsAbs(name) {
			name = filepath.Join(dir, name)
		}
		if !isFile(name) {
			return nil // a listed file is missing: fall back to the usual file names
		}
		out = append(out, name)
	}
	return out
}

func inspectDir(dir string) (Found, bool) {
	if files := envComposeFiles(dir); len(files) > 0 {
		return Found{Dir: dir, Files: files, EnvFile: filepath.Join(dir, ".env")}, true
	}
	for _, base := range baseNames {
		if !isFile(filepath.Join(dir, base)) {
			continue
		}
		f := Found{Dir: dir, Files: []string{filepath.Join(dir, base)}}
		for _, o := range overrideNames[base] {
			if isFile(filepath.Join(dir, o)) {
				f.Files = append(f.Files, filepath.Join(dir, o))
				break
			}
		}
		if isFile(filepath.Join(dir, ".env")) {
			f.EnvFile = filepath.Join(dir, ".env")
		}
		return f, true
	}
	return Found{}, false
}

func isFile(p string) bool {
	st, err := os.Lstat(p)
	return err == nil && st.Mode().IsRegular()
}

// InspectDir reports whether dir itself is a Compose project directory.
func InspectDir(dir string) (Found, bool) { return inspectDir(dir) }
