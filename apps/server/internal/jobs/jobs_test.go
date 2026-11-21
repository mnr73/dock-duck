package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"dockduck/internal/dockercli"
)

func TestJobLifecycleAndFollow(t *testing.T) {
	release := make(chan struct{})
	m := New(dockercli.StreamerFunc(func(_ context.Context, args []string, on func(string)) error {
		on("one")
		<-release
		on("two")
		if args[0] == "fail" {
			return errors.New("exit 1")
		}
		return nil
	}))
	snap, err := m.Start("p", "up", nil, []string{"ok"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("p", "down", nil, []string{"ok"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("want busy, got %v", err)
	}
	if _, err := m.Start("q", "up", nil, []string{"ok"}); err != nil { // other project is fine
		t.Fatal(err)
	}

	pos := 0
	var got []string
	deadline := time.After(2 * time.Second)
	released := false
	for {
		lines, next, st, wait, ok := m.Follow(snap.ID, pos)
		if !ok {
			t.Fatal("job vanished")
		}
		got, pos = append(got, lines...), next
		if len(got) == 1 && !released {
			released = true
			close(release)
		}
		if st.Status != Running && len(lines) == 0 {
			if st.Status != Succeeded {
				t.Fatalf("status %s", st.Status)
			}
			break
		}
		if len(lines) > 0 {
			continue // drain first; only wait when there is nothing new
		}
		select {
		case <-wait:
		case <-deadline:
			t.Fatal("timeout")
		}
	}
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("lines %v", got)
	}
	if _, busy := m.Active("p"); busy {
		t.Fatal("p must be free again")
	}
}

func TestFailedJob(t *testing.T) {
	m := New(dockercli.StreamerFunc(func(context.Context, []string, func(string)) error { return errors.New("boom") }))
	s, _ := m.Start("p", "up", nil, nil)
	for i := 0; i < 100; i++ {
		if g, _ := m.Get(s.ID); g.Status == Failed {
			if g.Error != "boom" {
				t.Fatal(g.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("never failed")
}
