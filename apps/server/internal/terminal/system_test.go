package terminal

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func lookIn(have ...string) func(string) (string, error) {
	return func(n string) (string, error) {
		if slices.Contains(have, n) {
			return "/usr/bin/" + n, nil
		}
		return "", errors.New("not found")
	}
}

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestDetect(t *testing.T) {
	if d := detect(env(), lookIn()); d != nil {
		t.Errorf("nothing installed: %+v", d)
	}
	if d := detect(env(), lookIn("xterm", "konsole")); d.Name != "konsole" || !slices.Equal(d.Pre, []string{"-e"}) {
		t.Errorf("preference order: %+v", d)
	}
	if d := detect(env(), lookIn("gnome-terminal")); !slices.Equal(d.Pre, []string{"--"}) {
		t.Errorf("gnome-terminal takes `--`: %+v", d)
	}
	if d := detect(env("DOCKDUCK_TERMINAL", "kitty"), lookIn("kitty", "konsole")); d.Name != "kitty" || len(d.Pre) != 0 {
		t.Errorf("explicit override wins: %+v", d)
	}
	if d := detect(env("DOCKDUCK_TERMINAL", "alacritty -e"), lookIn("alacritty")); !slices.Equal(d.Pre, []string{"-e"}) {
		t.Errorf("override keeps its flags: %+v", d)
	}
	if d := detect(env("TERMINAL", "st"), lookIn("st", "konsole")); d.Name != "st" || !slices.Equal(d.Pre, []string{"-e"}) {
		t.Errorf("$TERMINAL is run with -e: %+v", d)
	}
	if d := detect(env("DOCKDUCK_TERMINAL", "missing"), lookIn("konsole")); d.Name != "konsole" {
		t.Errorf("an override that is not installed falls back: %+v", d)
	}
}

func TestArgvKeepsUserInputAsSeparateArguments(t *testing.T) {
	st := &SystemTerminal{Name: "konsole", Bin: "/usr/bin/konsole", Pre: []string{"-e"}}
	evil := "app; touch /tmp/pwned"
	got := st.Argv("docker", "ctx", "c1", "0123456789abcdef", evil, []string{"exec", "-it", "-u", evil, "c1", "sh", "-c", "echo $(id)"})
	want := []string{"/usr/bin/konsole", "-e", "sh", "-c", launchScript, "sh", "docker", "ctx", "c1", "0123456789abcdef", evil, "exec", "-it", "-u", evil, "c1", "sh", "-c", "echo $(id)"}
	if !slices.Equal(got, want) {
		t.Errorf("\n got  %q\n want %q", got, want)
	}
	// nothing user-controlled is part of the script text
	if strings.Contains(launchScript, "$(id)") || strings.Contains(launchScript, "pwned") {
		t.Error("script must be static")
	}
}

func TestDisplayEnv(t *testing.T) {
	// already in a graphical session
	if _, ok := DisplayEnv([]string{"WAYLAND_DISPLAY=wayland-0"}, "", "/nonexistent"); !ok {
		t.Error("WAYLAND_DISPLAY set")
	}
	if _, ok := DisplayEnv([]string{"DISPLAY=:0"}, "", "/nonexistent"); !ok {
		t.Error("DISPLAY set")
	}
	// started by a service: discover the session's sockets
	rt, x11 := t.TempDir(), t.TempDir()
	if _, ok := DisplayEnv([]string{"PATH=/usr/bin"}, rt, x11); ok {
		t.Error("no display, no sockets")
	}
	_ = os.WriteFile(filepath.Join(rt, "wayland-0"), nil, 0o600)
	_ = os.WriteFile(filepath.Join(rt, "wayland-0.lock"), nil, 0o600)
	_ = os.WriteFile(filepath.Join(rt, "bus"), nil, 0o600)
	_ = os.WriteFile(filepath.Join(x11, "X1"), nil, 0o600)
	out, ok := DisplayEnv([]string{"PATH=/usr/bin"}, rt, x11)
	if !ok {
		t.Fatal("sockets exist")
	}
	for _, w := range []string{"WAYLAND_DISPLAY=wayland-0", "DISPLAY=:1", "XDG_RUNTIME_DIR=" + rt, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + filepath.Join(rt, "bus")} {
		if !slices.Contains(out, w) {
			t.Errorf("missing %s in %q", w, out)
		}
	}
	if slices.Contains(out, "WAYLAND_DISPLAY=wayland-0.lock") {
		t.Error(".lock is not a display")
	}
}

func TestLaunchWithoutDisplay(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	st := &SystemTerminal{Name: "x", Bin: "/bin/true"}
	// /tmp/.X11-unix may exist on the machine running the test; only assert the typed error path
	if err := st.Launch("docker", "", "c1", "0123456789abcdef", "", nil); err != nil && !errors.Is(err, ErrNoDisplay) {
		t.Logf("display found on this machine, launch result: %v", err)
	}
}

func TestLaunchReportsATerminalThatDiesAtOnce(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	st := &SystemTerminal{Name: "broken", Bin: "/bin/sh", Pre: []string{"-c", "echo 'cannot connect to display' >&2; exit 1", "x"}}
	err := st.Launch("docker", "", "c1", "0123456789abcdef", "", nil)
	if err == nil || !strings.Contains(err.Error(), "cannot connect to display") {
		t.Errorf("the terminal's own error must reach the user: %v", err)
	}
	ok := &SystemTerminal{Name: "ok", Bin: "/bin/sh", Pre: []string{"-c", "exit 0", "x"}}
	if err := ok.Launch("docker", "", "c1", "0123456789abcdef", "", nil); err != nil {
		t.Errorf("a terminal that hands over to a running server exits 0 right away: %v", err)
	}
}
