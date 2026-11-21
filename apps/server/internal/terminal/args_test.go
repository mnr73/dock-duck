package terminal

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildExecArgs(t *testing.T) {
	cases := []struct {
		name string
		ctx  string
		req  Request
		want []string
		disp string
	}{
		{"auto shell, default user", "", Request{}, []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "c1", "sh", "-c", autoShell}, "docker exec -it c1 bash"},
		{"root + context", "remote", Request{User: "0"}, []string{"--context", "remote", "exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "-u", "0", "c1", "sh", "-c", autoShell}, "docker exec -it -u 0 c1 bash"},
		{"explicit shell + workdir", "", Request{User: "1000:1000", Shell: "/bin/zsh", Workdir: "/var/www/html"}, []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "-u", "1000:1000", "-w", "/var/www/html", "c1", "/bin/zsh"}, "docker exec -it -u 1000:1000 -w /var/www/html c1 /bin/zsh"},
		{"command", "", Request{Command: "mysql -uerp -p"}, []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "c1", "sh", "-c", "mysql -uerp -p"}, "docker exec -it c1 sh -c 'mysql -uerp -p'"},
		{"command with chosen shell", "", Request{Command: "echo $HOME", Shell: "/bin/bash"}, []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "c1", "/bin/bash", "-c", "echo $HOME"}, "docker exec -it c1 /bin/bash -c 'echo $HOME'"},
	}
	for _, c := range cases {
		got, disp, err := BuildExecArgs(c.ctx, "c1", "", c.req)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got  %q\n want %q", c.name, got, c.want)
		}
		if disp != c.disp {
			t.Errorf("%s display:\n got  %s\n want %s", c.name, disp, c.disp)
		}
	}
}

func TestValidationRejectsFlagInjection(t *testing.T) {
	bad := []Request{
		{User: "-v /:/host"}, {User: "--privileged"}, {User: "root --user 0"}, {User: "a b"}, {User: ""[:0] + "x;y"},
		{User: "root:"}, {User: ":root"}, {User: strings.Repeat("a", 70)}, {User: "ro\nt"},
		{Shell: "-c"}, {Shell: "--rm"}, {Shell: "bash -c x"}, {Shell: "/bin/sh;id"}, {Shell: "$(id)"}, {Shell: "/bin/sh "},
		{Workdir: "relative"}, {Workdir: "-w"}, {Workdir: "/a\nb"}, {Workdir: "/" + strings.Repeat("a", 1100)},
		{Command: strings.Repeat("a", MaxCommand+1)}, {Command: "a\x00b"},
	}
	for _, r := range bad {
		if _, _, err := BuildExecArgs("", "c1", "", r); err == nil {
			t.Errorf("must reject %+v", r)
		}
	}
	for _, c := range []string{"-x", "a b", "", "--rm", "a;b", "../x"} {
		if _, _, err := BuildExecArgs("", c, "", Request{}); err == nil {
			t.Errorf("must reject container %q", c)
		}
	}
	good := []Request{{User: "0"}, {User: "1000"}, {User: "www-data"}, {User: "app:app"}, {User: "1000:1000"}, {Shell: "/bin/bash"}, {Shell: "bash"}, {Shell: "/usr/local/bin/fish"}, {Workdir: "/"}, {Workdir: "/var/www/my app"}}
	for _, r := range good {
		if _, _, err := BuildExecArgs("", "c1", "", r); err != nil {
			t.Errorf("must accept %+v: %v", r, err)
		}
	}
}

func TestUserNeverBecomesAFlag(t *testing.T) {
	// whatever passes validation lands right after -u / -w, never as its own argument
	args, _, _ := BuildExecArgs("", "c1", "", Request{User: "app", Workdir: "/srv"})
	i := slices.Index(args, "-u")
	if i < 0 || args[i+1] != "app" || strings.HasPrefix(args[i+1], "-") {
		t.Errorf("%q", args)
	}
}

func TestMarkerAndCleanupArgs(t *testing.T) {
	args, _, _ := BuildExecArgs("", "c1", "abcd1234ef567890", Request{})
	i := slices.Index(args, "DOCKDUCK_TERM=abcd1234ef567890")
	if i < 1 || args[i-1] != "-e" {
		t.Errorf("session is tagged through -e: %q", args)
	}
	// default user: no -u at all (the image's user, same as the session)
	c, err := CleanupArgs("remote", "c1", "abcd1234ef567890", "")
	if err != nil {
		t.Fatal(err)
	}
	if c[0] != "--context" || slices.Contains(c, "-u") {
		t.Errorf("cleanup keeps the session's context and default user: %q", c)
	}
	if c[len(c)-1] != "abcd1234ef567890" || !strings.Contains(c[slices.Index(c, "-c")+1], "DOCKDUCK_TERM=") {
		t.Errorf("%q", c)
	}
	// explicit user: the SAME user (root cannot read another user's /proc/PID/environ without CAP_SYS_PTRACE)
	c, _ = CleanupArgs("", "c1", "abcd1234ef567890", "1000:1000")
	if i := slices.Index(c, "-u"); i < 0 || c[i+1] != "1000:1000" {
		t.Errorf("cleanup must run as the session user: %q", c)
	}
	for _, m := range []string{"", "xyz", "abcd1234ef56789", "ABCD1234EF567890", "abcd1234ef567890; id"} {
		if _, err := CleanupArgs("", "c1", m, ""); err == nil {
			t.Errorf("marker %q must be rejected", m)
		}
	}
	if _, err := CleanupArgs("", "-x", "abcd1234ef567890", ""); err == nil {
		t.Error("container must be validated")
	}
	if _, err := CleanupArgs("", "c1", "abcd1234ef567890", "--privileged"); err == nil {
		t.Error("user must be validated")
	}
}
