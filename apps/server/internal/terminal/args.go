package terminal

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"dockduck/internal/dockercli"
)

// Request is what the user chooses when opening a terminal.
type Request struct {
	Service   string `json:"service"`
	Container string `json:"container,omitempty"` // name of one replica; default: the first running one
	User      string `json:"user,omitempty"`      // "" = the container's default; "0", "1000", "root", "app:app", ...
	Shell     string `json:"shell,omitempty"`     // "" = auto (bash, then sh); or /bin/zsh, ...
	Command   string `json:"command,omitempty"`   // optional: run this instead of an interactive shell
	Workdir   string `json:"workdir,omitempty"`   // optional absolute path
	Cols      uint16 `json:"cols,omitempty"`
	Rows      uint16 `json:"rows,omitempty"`
}

var (
	userRe    = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.\-]{0,63}(:[A-Za-z0-9_][A-Za-z0-9_.\-]{0,63})?$`)
	shellRe   = regexp.MustCompile(`^(/|[A-Za-z0-9_])[A-Za-z0-9_.+/\-]*$`)
	workdirRe = regexp.MustCompile(`^/[^\x00-\x1f]*$`)
)

// MarkerVar is the environment variable that tags the processes of a session.
const MarkerVar = "DOCKDUCK_TERM"

// MaxCommand bounds the optional command.
const MaxCommand = 4096

// Validate checks the user-supplied fields. Nothing here is passed through a
// local shell (argv only), but they must not be able to turn into docker flags.
func (r Request) Validate() error {
	if r.User != "" && !userRe.MatchString(r.User) {
		return errors.New("invalid user: use a name or UID, optionally name:group")
	}
	if r.Shell != "" && !shellRe.MatchString(r.Shell) {
		return errors.New("invalid shell: use a path such as /bin/bash")
	}
	if r.Workdir != "" && (len(r.Workdir) > 1024 || !workdirRe.MatchString(r.Workdir)) {
		return errors.New("invalid working directory: use an absolute path")
	}
	if len(r.Command) > MaxCommand || strings.ContainsRune(r.Command, 0) {
		return errors.New("command is too long")
	}
	return nil
}

// autoShell prefers bash and falls back to sh, inside the container.
const autoShell = `if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi`

// BuildExecArgs returns the docker argv (without the leading `docker`) for an
// interactive session in `container`, and a human readable equivalent.
func BuildExecArgs(contextName, container, marker string, r Request) (args []string, display string, err error) {
	if err = r.Validate(); err != nil {
		return nil, "", err
	}
	if !dockercli.ValidName(container) {
		return nil, "", fmt.Errorf("invalid container name %q", container)
	}
	a := []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor"}
	if marker != "" {
		// tags every process of this session so it can be found again (see CleanupArgs)
		a = append(a, "-e", MarkerVar+"="+marker)
	}
	disp := []string{"docker", "exec", "-it"}
	if r.User != "" {
		a = append(a, "-u", r.User)
		disp = append(disp, "-u", r.User)
	}
	if r.Workdir != "" {
		a = append(a, "-w", r.Workdir)
		disp = append(disp, "-w", quote(r.Workdir))
	}
	a = append(a, container)
	disp = append(disp, container)

	switch {
	case r.Command != "":
		sh := r.Shell
		if sh == "" {
			sh = "sh"
		}
		a = append(a, sh, "-c", r.Command)
		disp = append(disp, sh, "-c", quote(r.Command))
	case r.Shell != "":
		a = append(a, r.Shell)
		disp = append(disp, r.Shell)
	default:
		a = append(a, "sh", "-c", autoShell)
		disp = append(disp, "bash")
	}
	return dockercli.Scoped(contextName, a...), strings.Join(disp, " "), nil
}

func quote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`&|;<>()*?[]{}#~!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

var markerRe = regexp.MustCompile(`^[a-f0-9]{16}$`)

//go:embed cleanup.sh
var cleanupScript string

// CleanupArgs returns the docker argv that ends what a session left running in
// the container. Killing the local `docker exec` client is not enough: Docker does
// not forward the hangup, so the shell (and e.g. a stuck mysql) would keep running.
//
// It runs as the SAME user as the session (`user` as given in the Request, empty =
// the image's default user): reading /proc/PID/environ of another user's process
// needs CAP_SYS_PTRACE, which Docker drops even for root, but a user can always
// read its own processes. Processes the user handed to another uid (sudo, su) are
// out of reach and stay running.
func CleanupArgs(contextName, container, marker, user string) ([]string, error) {
	if !dockercli.ValidName(container) || !markerRe.MatchString(marker) {
		return nil, errors.New("invalid cleanup target")
	}
	if user != "" && !userRe.MatchString(user) {
		return nil, errors.New("invalid user")
	}
	a := []string{"exec"}
	if user != "" {
		a = append(a, "-u", user)
	}
	a = append(a, container, "sh", "-c", cleanupScript, "sh", marker)
	return dockercli.Scoped(contextName, a...), nil
}
