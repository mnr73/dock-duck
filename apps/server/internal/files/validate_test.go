package files

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"dockduck/internal/dockercli"
)

type fakeStdin struct {
	gotStdin string
	gotArgs  []string
	err      error
}

func (f *fakeStdin) OutputStdin(_ context.Context, stdin string, args ...string) ([]byte, []byte, error) {
	f.gotStdin, f.gotArgs = stdin, args
	return nil, nil, f.err
}

func exit(stderr string) error {
	return &dockercli.ExitError{Stderr: stderr, Err: errors.New("exit status 1")}
}

// Messages below are what `docker compose config -q` really printed (measured).
func TestParseValidationRealMessages(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		ok       bool
		severity string
		line     int
		contains string
	}{
		{"valid", nil, true, "", 0, ""},
		{"yaml syntax", exit("yaml: line 4: did not find expected key\n"), false, "error", 4, "did not find expected key"},
		{"yaml unmarshal", exit("yaml: unmarshal errors:\n  line 7: cannot unmarshal !!str into map\n"), false, "error", 0, "unmarshal errors"},
		{"schema type", exit("validating -: services.app.ports must be a array\n"), false, "error", 0, "services.app.ports must be a array"},
		{"unknown key", exit("validating -: services.app.nonsense false schema\n"), false, "error", 0, "services.app.nonsense"},
		{"required var", exit("error while interpolating services.app.image: required variable NEEDED is missing a value: need it\n"), false, "error", 0, "NEEDED"},
		{"empty", exit("empty compose file\n"), false, "error", 0, "empty compose file"},
		{"warning only", exit("time=\"2026-11-21T12:06:13+03:30\" level=warning msg=\"The \\\"MISSING_VAR\\\" variable is not set. Defaulting to a blank string.\"\n"), true, "warning", 0, "MISSING_VAR"},
	}
	for _, c := range cases {
		stderr := ""
		var ee *dockercli.ExitError
		if errors.As(c.err, &ee) {
			stderr = ee.Stderr
		}
		if c.name == "warning only" {
			c.err = nil // real Compose exits 0 and only prints the warning
		}
		v := parseValidation(stderr, c.err)
		if v.OK != c.ok {
			t.Errorf("%s: ok=%v, want %v (%+v)", c.name, v.OK, c.ok, v.Problems)
		}
		if c.severity == "" {
			if len(v.Problems) != 0 {
				t.Errorf("%s: no problems expected: %+v", c.name, v.Problems)
			}
			continue
		}
		if len(v.Problems) == 0 || v.Problems[0].Severity != c.severity || !strings.Contains(v.Problems[0].Message, c.contains) || v.Problems[0].Line != c.line {
			t.Errorf("%s: %+v", c.name, v.Problems)
		}
	}
	if v := parseValidation("yaml: line 4: did not find expected key\n", exit("yaml: line 4: did not find expected key\n")); v.Problems[0].Message != "did not find expected key" || v.Problems[0].Line != 4 {
		t.Errorf("line prefix must move out of the message: %+v", v.Problems)
	}
	// a non-docker failure (timeout, binary missing) must not look valid
	if v := parseValidation("", errors.New("context deadline exceeded")); v.OK {
		t.Errorf("exec errors are not 'valid': %+v", v)
	}
}

func TestValidateComposeBuildsArgvAndFeedsEditedFileViaStdin(t *testing.T) {
	f := &fakeStdin{}
	_, _ = ValidateCompose(context.Background(), f, "remote", "/p", []string{"/p/compose.yaml", "/p/compose.override.yaml"}, 1, "services: {}\n")
	got := strings.Join(f.gotArgs, " ")
	want := "--context remote compose --project-directory /p -f /p/compose.yaml -f - config -q"
	if got != want {
		t.Errorf("\n got  %s\n want %s", got, want)
	}
	if f.gotStdin != "services: {}\n" {
		t.Errorf("stdin %q", f.gotStdin)
	}
}

// Runs the real docker CLI: config validation needs no daemon. Opt-in with
// DOCKDUCK_REAL_DOCKER=1, because the wording and line numbers of Compose's own
// errors change between Compose versions, so this must not decide whether a build
// passes on a machine whose Compose we do not control (CI runners).
func TestValidateComposeAgainstRealDocker(t *testing.T) {
	if os.Getenv("DOCKDUCK_REAL_DOCKER") != "1" {
		t.Skip("set DOCKDUCK_REAL_DOCKER=1 to run against the installed docker compose")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("no docker")
	}
	c := dockercli.New()
	dir := t.TempDir()
	check := func(content string) Validation {
		v, err := ValidateCompose(context.Background(), c, "", dir, []string{dir + "/compose.yaml"}, 0, content)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := check("services:\n  app:\n    image: redis:7-alpine\n"); !v.OK {
		t.Errorf("valid file rejected: %+v", v)
	}
	if v := check("services:\n  app:\n    image: x\n   bad: 1\n"); v.OK || v.Problems[0].Line != 4 {
		t.Errorf("syntax error must report line 4: %+v", v)
	}
	if v := check("services:\n  app:\n    image: x\n    ports: 80\n"); v.OK || !strings.Contains(v.Problems[0].Message, "ports") {
		t.Errorf("schema error: %+v", v)
	}
	if v := check("services:\n  app:\n    image: x\n    command: ${NOT_SET_ANYWHERE}\n"); !v.OK {
		t.Errorf("an unset variable is only a warning: %+v", v)
	}
	if v := check("services:\n  app:\n    image: x\n    command: ${NOT_SET_ANYWHERE}\n"); len(v.Problems) != 1 || strings.Contains(v.Problems[0].Message, `\"`) {
		t.Errorf("warning text must be unescaped: %+v", v.Problems)
	}
	if v := check(""); v.OK {
		t.Errorf("empty file: %+v", v)
	}
}
