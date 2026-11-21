// Package events turns `docker events` into a fan-out stream of container
// lifecycle events, one docker process per context while anyone is listening.
package events

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"dockduck/internal/dockercli"
)

// Event is a compose container event for the UI.
type Event struct {
	Type      string `json:"type"` // always "container"
	Action    string `json:"action"`
	Project   string `json:"project"`
	Service   string `json:"service"`
	Container string `json:"container"`
	ID        string `json:"id"`
	Time      int64  `json:"time"`
}

// Hub multiplexes docker event streams.
type Hub struct {
	Streamer dockercli.Streamer
	Backoff  time.Duration // restart delay when the stream dies; default 2s

	mu      sync.Mutex
	streams map[string]*stream
}

type stream struct {
	cancel context.CancelFunc
	subs   map[chan Event]struct{}
}

// New returns a hub.
func New(s dockercli.Streamer) *Hub { return &Hub{Streamer: s, streams: map[string]*stream{}} }

// Subscribe listens to ctxName. Call the returned func to stop; the docker
// process is killed when the last subscriber leaves.
func (h *Hub) Subscribe(ctxName string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	h.mu.Lock()
	st, ok := h.streams[ctxName]
	if !ok {
		ctx, cancel := context.WithCancel(context.Background())
		st = &stream{cancel: cancel, subs: map[chan Event]struct{}{}}
		h.streams[ctxName] = st
		go h.run(ctx, ctxName)
	}
	st.subs[ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(st.subs, ch)
			if len(st.subs) == 0 {
				st.cancel()
				if h.streams[ctxName] == st {
					delete(h.streams, ctxName)
				}
			}
		})
	}
}

func (h *Hub) run(ctx context.Context, ctxName string) {
	args := dockercli.Scoped(ctxName, "events", "--filter", "type=container", "--format", "{{json .}}")
	backoff := h.Backoff
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	for ctx.Err() == nil {
		_ = h.Streamer.Stream(ctx, args, func(line string) {
			if ev, ok := Parse(line); ok {
				h.broadcast(ctxName, ev)
			}
		})
		select {
		case <-ctx.Done():
		case <-time.After(backoff):
		}
	}
}

func (h *Hub) broadcast(ctxName string, ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.streams[ctxName]
	if st == nil {
		return
	}
	for ch := range st.subs {
		select {
		case ch <- ev:
		default: // slow consumer: drop, UI refetches on next event anyway
		}
	}
}

type rawEvent struct {
	Type   string
	Action string
	Actor  struct {
		ID         string
		Attributes map[string]string
	}
	TimeNano int64 `json:"timeNano"`
}

// Parse decodes one `docker events` JSON line; non-compose and noisy events are dropped.
func Parse(line string) (Event, bool) {
	var r rawEvent
	if json.Unmarshal([]byte(line), &r) != nil || r.Type != "container" {
		return Event{}, false
	}
	a := r.Actor.Attributes
	project := a["com.docker.compose.project"]
	if project == "" || strings.HasPrefix(r.Action, "exec_") || r.Action == "attach" || r.Action == "resize" {
		return Event{}, false
	}
	action := r.Action
	if strings.HasPrefix(action, "health_status") {
		action = "health_status"
	}
	return Event{Type: "container", Action: action, Project: project,
		Service: a["com.docker.compose.service"], Container: a["name"], ID: r.Actor.ID, Time: r.TimeNano / 1e9}, true
}
