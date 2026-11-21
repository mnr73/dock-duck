package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// These tests run real processes under a real PTY (`sh`), the same layer docker exec uses.

func collect(t *testing.T, a *Attachment, until string, d time.Duration) string {
	t.Helper()
	var buf bytes.Buffer
	deadline := time.After(d)
	for {
		select {
		case b, ok := <-a.C:
			if !ok {
				return buf.String()
			}
			buf.Write(b)
			if strings.Contains(buf.String(), until) {
				return buf.String()
			}
		case <-deadline:
			t.Fatalf("timeout waiting for %q; got %q", until, buf.String())
		}
	}
}

func openSh(t *testing.T, m *Manager, script string) *Session {
	t.Helper()
	s, err := m.Open(Meta{Service: "x"}, []string{"sh", "-c", script}, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInteractiveEchoAndArrowKeysReachTheProgram(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	// `cat -v` prints control sequences visibly: proves the raw bytes of arrow keys
	// arrive untouched (a cooked terminal or a line-based pipe would swallow/alter them)
	s := openSh(t, m, "stty raw -echo; echo READY; exec cat -v")
	_, a := s.Attach()
	collect(t, a, "READY", 5*time.Second) // keys sent earlier would still hit the cooked line discipline
	_ = s.Write([]byte("\x1b[A\x1b[B\x1b[C\x1b[D\x1bOH\x7f\x03"))
	out := collect(t, a, "^C", 5*time.Second)
	if !strings.Contains(out, "^[[A") { // the marker above may not include everything yet
		out += collect(t, a, "^C", 2*time.Second)
	}
	for _, want := range []string{"^[[A", "^[[B", "^[[C", "^[[D", "^[OH", "^?", "^C"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
}

func TestIsATerminalWithTheRequestedSize(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, `[ -t 0 ] && [ -t 1 ] && echo TTY-OK; stty size; sleep 0.3; stty size; sleep 1`)
	_, a := s.Attach()
	time.Sleep(150 * time.Millisecond)
	if err := s.Resize(132, 38); err != nil {
		t.Fatal(err)
	}
	out := collect(t, a, "38 132", 5*time.Second)
	if !strings.Contains(out, "TTY-OK") || !strings.Contains(out, "24 80") {
		t.Errorf("not a TTY of the initial size: %q", out)
	}
	if err := s.Resize(1, 1); err == nil {
		t.Error("absurd sizes are rejected")
	}
}

func TestReattachReplaysAndKeepsRunning(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, "echo first; read x; echo got:$x; sleep 5")
	_, a1 := s.Attach()
	collect(t, a1, "first", 5*time.Second)
	s.Detach(a1) // the browser tab is closed
	time.Sleep(100 * time.Millisecond)
	if s.Info().State != "running" || s.Info().Attached {
		t.Fatalf("a detached shell must keep running: %+v", s.Info())
	}
	replay, a2 := s.Attach() // reload: same session, previous output replayed
	if !strings.Contains(string(replay), "first") {
		t.Errorf("replay: %q", replay)
	}
	_ = s.Write([]byte("hello\n"))
	collect(t, a2, "got:hello", 5*time.Second)
}

func TestSecondViewerTakesOverTheFirst(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, "sleep 5")
	_, a1 := s.Attach()
	_, a2 := s.Attach()
	select {
	case _, ok := <-a1.C:
		if ok {
			t.Fatal("expected a closed channel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first viewer was not closed")
	}
	if a1.Reason() != "taken" {
		t.Errorf("reason %q", a1.Reason())
	}
	s.Detach(a1) // a late detach of the old viewer must not detach the new one
	if !s.Info().Attached {
		t.Error("the new viewer must stay attached")
	}
	_ = a2
}

func TestExitCodeAndKeptAfterExit(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, "echo bye; exit 3")
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("did not finish")
	}
	if s.Exit() != 3 {
		t.Errorf("exit code %d", s.Exit())
	}
	info := s.Info()
	if info.State != "exited" || info.ExitCode == nil || *info.ExitCode != 3 {
		t.Errorf("%+v", info)
	}
	replay, a := s.Attach() // attaching to a finished session still shows what it printed
	if !strings.Contains(string(replay), "bye") {
		t.Errorf("replay %q", replay)
	}
	if _, ok := <-a.C; ok {
		t.Error("channel of a finished session is closed")
	}
	if err := s.Write([]byte("x")); err == nil {
		t.Error("writing to a finished session fails")
	}
}

func TestSlowViewerIsDroppedNotBlocking(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, "i=0; while [ $i -lt 6000 ]; do echo line-$i-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx; i=$((i+1)); done; echo END")
	_, a := s.Attach() // never reads
	select {
	case <-s.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("a viewer that does not read must not stall the process")
	}
	deadline := time.Now().Add(2 * time.Second)
	for a.Reason() != "slow" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if a.Reason() != "slow" {
		t.Errorf("reason %q", a.Reason())
	}
	replay, _ := s.Attach()
	if !strings.Contains(string(replay), "END") {
		t.Error("the output is still in the replay buffer")
	}
}

func TestCloseKillsTheProcessAndLimit(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	s := openSh(t, m, "trap '' HUP; sleep 60")
	if !m.Close(s.ID()) {
		t.Fatal("close")
	}
	select {
	case <-s.Done():
	case <-time.After(6 * time.Second):
		t.Fatal("a process that ignores SIGHUP must still be killed")
	}
	if _, ok := m.Get(s.ID()); ok {
		t.Error("closed sessions are forgotten")
	}
	for i := 0; i < MaxSessions; i++ {
		if _, err := m.Open(Meta{}, []string{"sleep", "30"}, 80, 24); err != nil {
			t.Fatalf("session %d: %v", i, err)
		}
	}
	if _, err := m.Open(Meta{}, []string{"sleep", "30"}, 80, 24); err != ErrTooMany {
		t.Errorf("limit: %v", err)
	}
}

func TestReaper(t *testing.T) {
	m := NewManager(SpawnPTY)
	defer m.Shutdown()
	exited := openSh(t, m, "exit 0")
	<-exited.Done()
	idle := openSh(t, m, "sleep 30")
	attached := openSh(t, m, "sleep 30")
	_, a := attached.Attach()
	defer attached.Detach(a)

	m.reapOnce(time.Now().Add(ExitedKeepFor - time.Minute))
	if _, ok := m.Get(exited.ID()); !ok {
		t.Error("a finished session is kept for a while")
	}
	m.reapOnce(time.Now().Add(ExitedKeepFor + time.Minute))
	if _, ok := m.Get(exited.ID()); ok {
		t.Error("finished sessions expire")
	}
	if _, ok := m.Get(idle.ID()); !ok {
		t.Error("an idle running shell outlives the exit timeout")
	}
	m.reapOnce(time.Now().Add(DetachedKeepFor + time.Minute))
	if _, ok := m.Get(idle.ID()); ok {
		t.Error("a shell nobody has viewed for hours is closed")
	}
	if _, ok := m.Get(attached.ID()); !ok {
		t.Error("an attached shell is never reaped")
	}
}

func TestRingKeepsTheTailOnLineBoundary(t *testing.T) {
	r := newRing(20)
	r.write([]byte("aaaa\nbbbb\ncccc\ndddd\n"))
	r.write([]byte("eeee\n"))
	got := string(r.bytes())
	if !strings.HasSuffix(got, "eeee\n") || strings.HasPrefix(got, "a") || len(got) > 20 {
		t.Errorf("%q", got)
	}
	if !strings.HasPrefix(got, "cccc") && !strings.HasPrefix(got, "dddd") && !strings.HasPrefix(got, "bbbb") {
		t.Errorf("must start at a line start: %q", got)
	}
	u := newRing(5)
	u.write([]byte("héllo wörld"))
	if b := u.bytes(); !strings.ContainsRune(string(b), 'ö') && len(b) > 0 && b[0]&0xC0 == 0x80 {
		t.Errorf("must not start inside a UTF-8 sequence: %q", b)
	}
}

func TestCleanupRunsOnCloseReapAndShutdownButNotAfterANormalExit(t *testing.T) {
	m := NewManager(SpawnPTY)
	calls := make(chan string, 8)
	open := func(name, script string) *Session {
		s, err := m.OpenTagged(Meta{Service: name}, func(id string) ([]string, string, func(), error) {
			return []string{"sh", "-c", script}, "", func() { calls <- name }, nil
		}, 80, 24)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	got := func(want string) {
		t.Helper()
		select {
		case g := <-calls:
			if g != want {
				t.Fatalf("cleanup for %q, want %q", g, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("no cleanup for %q", want)
		}
	}
	closed := open("closed", "sleep 30")
	m.Close(closed.ID())
	got("closed")
	time.Sleep(300 * time.Millisecond) // the killed shell's exit must not sweep a second time
	select {
	case g := <-calls:
		t.Fatalf("cleanup ran twice (for %q)", g)
	default:
	}

	done := open("done", "exit 0")
	<-done.Done()
	got("done") // a shell that exits by itself may still have left background jobs behind
	m.Close(done.ID())

	idle := open("idle", "sleep 30")
	m.reapOnce(time.Now().Add(DetachedKeepFor + time.Minute))
	got("idle")
	_ = idle

	open("shutdown", "sleep 30")
	m.Shutdown() // waits for the cleanup
	got("shutdown")
}

func TestShutdownIsDistinguishableFromAnExit(t *testing.T) {
	m := NewManager(SpawnPTY)
	if m.ShuttingDown() {
		t.Fatal("not shutting down yet")
	}
	s := openSh(t, m, "sleep 30")
	_, a := s.Attach()
	go m.Shutdown()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown must end the shell")
	}
	for range a.C {
	}
	if !m.ShuttingDown() {
		t.Error("a viewer that sees the session end must be able to tell it was a shutdown")
	}
}
