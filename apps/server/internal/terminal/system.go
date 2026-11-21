package terminal

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed launch.sh
var launchScript string

// Candidates in order of preference. `-e` style terminals take the command as the
// rest of the argv. x-terminal-emulator is Debian's "default terminal" (its
// alternatives system: KDE -> konsole, GNOME -> gnome-terminal, ...), so it
// represents what the user configured.
var candidates = []struct {
	bin string
	pre []string
}{
	{"x-terminal-emulator", []string{"-e"}},
	{"konsole", []string{"-e"}},
	{"gnome-terminal", []string{"--"}},
	{"xfce4-terminal", []string{"-x"}},
	{"kitty", nil},
	{"alacritty", []string{"-e"}},
	{"foot", nil},
	{"wezterm", []string{"start", "--"}},
	{"xterm", []string{"-e"}},
}

// SystemTerminal is a terminal emulator that can run a command in its own window.
type SystemTerminal struct {
	Name string   // for the UI
	Bin  string   // absolute path
	Pre  []string // flags before the command
}

// DetectSystemTerminal finds the terminal to use. DOCKDUCK_TERMINAL ("kitty" or
// "alacritty -e") and $TERMINAL (run with -e) override the built-in list.
func DetectSystemTerminal() *SystemTerminal {
	return detect(os.Getenv, exec.LookPath)
}

func detect(getenv func(string) string, look func(string) (string, error)) *SystemTerminal {
	if v := strings.Fields(getenv("DOCKDUCK_TERMINAL")); len(v) > 0 {
		if p, err := look(v[0]); err == nil {
			return &SystemTerminal{Name: filepath.Base(v[0]), Bin: p, Pre: v[1:]}
		}
	}
	if t := strings.Fields(getenv("TERMINAL")); len(t) > 0 {
		if p, err := look(t[0]); err == nil {
			return &SystemTerminal{Name: filepath.Base(t[0]), Bin: p, Pre: append(append([]string{}, t[1:]...), "-e")}
		}
	}
	for _, c := range candidates {
		if p, err := look(c.bin); err == nil {
			name := c.bin
			if c.bin == "x-terminal-emulator" { // show what it points to (konsole, ...)
				if real, err := filepath.EvalSymlinks(p); err == nil {
					name = filepath.Base(real)
				}
			}
			return &SystemTerminal{Name: name, Bin: p, Pre: c.pre}
		}
	}
	return nil
}

// DisplayEnv returns the environment a GUI program needs. A server started from a
// desktop terminal already has it; one started by systemd/ssh usually does not, so
// the running session's sockets are looked up. It reports whether a display exists.
func DisplayEnv(env []string, runtimeDir string, x11Dir string) ([]string, bool) {
	get := func(k string) string {
		for _, e := range env {
			if strings.HasPrefix(e, k+"=") {
				return e[len(k)+1:]
			}
		}
		return ""
	}
	out := append([]string{}, env...)
	set := func(k, v string) { out = append(out, k+"="+v) }
	if get("WAYLAND_DISPLAY") != "" || get("DISPLAY") != "" {
		return out, true
	}
	found := false
	if runtimeDir != "" {
		if socks, _ := filepath.Glob(filepath.Join(runtimeDir, "wayland-[0-9]*")); len(socks) > 0 {
			for _, s := range socks {
				if !strings.HasSuffix(s, ".lock") {
					set("WAYLAND_DISPLAY", filepath.Base(s))
					found = true
					break
				}
			}
		}
		if get("XDG_RUNTIME_DIR") == "" {
			set("XDG_RUNTIME_DIR", runtimeDir)
		}
		if get("DBUS_SESSION_BUS_ADDRESS") == "" {
			if _, err := os.Stat(filepath.Join(runtimeDir, "bus")); err == nil {
				set("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(runtimeDir, "bus"))
			}
		}
	}
	if socks, _ := filepath.Glob(filepath.Join(x11Dir, "X[0-9]*")); len(socks) > 0 {
		set("DISPLAY", ":"+strings.TrimPrefix(filepath.Base(socks[0]), "X"))
		found = true
	}
	return out, found
}

// Argv builds the full command line that opens the window.
func (t *SystemTerminal) Argv(dockerBin, contextName, container, marker, user string, execArgs []string) []string {
	argv := append([]string{t.Bin}, t.Pre...)
	argv = append(argv, "sh", "-c", launchScript, "sh", dockerBin, contextName, container, marker, user)
	return append(argv, execArgs...)
}

// ErrNoDisplay means there is no graphical session to open a window in.
var ErrNoDisplay = errors.New("no graphical session found (no DISPLAY or WAYLAND_DISPLAY)")

// Launch opens the terminal window running the exec command. The window is its
// own session: closing DockDuck does not close it. A terminal that dies at once
// (no display, bad flags) is reported with its own error output.
func (t *SystemTerminal) Launch(dockerBin, contextName, container, marker, user string, execArgs []string) error {
	env, ok := DisplayEnv(os.Environ(), os.Getenv("XDG_RUNTIME_DIR"), "/tmp/.X11-unix")
	if !ok {
		return ErrNoDisplay
	}
	env = append(env, "DOCKDUCK_CLEANUP="+cleanupScript)
	argv := t.Argv(dockerBin, contextName, container, marker, user, execArgs)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		// Terminals that hand over to a running server (gnome-terminal, konsole with
		// an existing instance) exit at once with 0: that is success.
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if i := strings.IndexByte(msg, '\n'); i >= 0 {
				msg = msg[:i]
			}
			if msg == "" {
				msg = err.Error()
			}
			return fmt.Errorf("%s failed to start: %s", t.Name, msg)
		}
	case <-time.After(1500 * time.Millisecond):
	}
	return nil
}
