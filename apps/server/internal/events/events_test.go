package events

import (
	"context"
	"testing"
	"time"

	"dockduck/internal/dockercli"
)

const line = `{"Type":"container","Action":"die","Actor":{"ID":"abc","Attributes":{"com.docker.compose.project":"erp","com.docker.compose.service":"redis","name":"r1"}},"timeNano":1795245062576419362}`

func TestParse(t *testing.T) {
	ev, ok := Parse(line)
	if !ok || ev.Project != "erp" || ev.Service != "redis" || ev.Action != "die" || ev.Time != 1795245062 {
		t.Fatalf("%+v %v", ev, ok)
	}
	for _, bad := range []string{`{"Type":"network"}`, `not json`,
		`{"Type":"container","Action":"exec_start: sh","Actor":{"Attributes":{"com.docker.compose.project":"x"}}}`,
		`{"Type":"container","Action":"start","Actor":{"Attributes":{}}}`} {
		if _, ok := Parse(bad); ok {
			t.Errorf("should drop %q", bad)
		}
	}
	if ev, _ := Parse(`{"Type":"container","Action":"health_status: healthy","Actor":{"Attributes":{"com.docker.compose.project":"x"}}}`); ev.Action != "health_status" {
		t.Errorf("health: %+v", ev)
	}
}

func TestHubFanOutAndCleanup(t *testing.T) {
	started := make(chan struct{}, 4)
	stopped := make(chan struct{}, 4)
	h := New(dockercli.StreamerFunc(func(ctx context.Context, _ []string, on func(string)) error {
		started <- struct{}{}
		on(line)
		<-ctx.Done()
		stopped <- struct{}{}
		return ctx.Err()
	}))
	a, cancelA := h.Subscribe("default")
	b, cancelB := h.Subscribe("default")
	<-started
	for _, ch := range []<-chan Event{a, b} {
		select {
		case ev := <-ch:
			if ev.Service != "redis" {
				t.Fatal(ev)
			}
		case <-time.After(time.Second):
			t.Fatal("no event")
		}
	}
	select {
	case <-started:
		t.Fatal("second subscriber must share the process")
	default:
	}
	cancelA()
	cancelA() // idempotent
	select {
	case <-stopped:
		t.Fatal("still one subscriber")
	case <-time.After(50 * time.Millisecond):
	}
	cancelB()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("process must stop with last subscriber")
	}
}
