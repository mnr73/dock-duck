// Package jobs runs long docker operations (compose up/down/build…) in the
// background and lets clients follow their output.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"dockduck/internal/dockercli"
)

// Status of a job.
const (
	Running   = "running"
	Succeeded = "succeeded"
	Failed    = "failed"
)

const (
	maxLines = 5000
	keepDone = 50
)

// ErrBusy is returned when the project already has a running job.
var ErrBusy = errors.New("another operation is already running for this project")

// Job is one background operation. Fields are guarded by the manager mutex.
type Job struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	Action    string    `json:"action"`
	Services  []string  `json:"services,omitempty"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitzero"`

	lines  []string
	offset int           // number of lines dropped from the front
	notify chan struct{} // closed and replaced on every change
}

// Snapshot is a consistent copy of a job's state.
type Snapshot struct {
	Job
	Lines []string `json:"-"`
}

// Manager owns all jobs.
type Manager struct {
	Streamer dockercli.Streamer

	mu     sync.Mutex
	byID   map[string]*Job
	order  []string
	active map[string]string // project -> job id
}

// New creates a manager.
func New(s dockercli.Streamer) *Manager {
	return &Manager{Streamer: s, byID: map[string]*Job{}, active: map[string]string{}}
}

// Start launches args (a full docker argv) as a job for the project key.
func (m *Manager) Start(project, action string, services []string, args []string) (Snapshot, error) {
	m.mu.Lock()
	if _, busy := m.active[project]; busy {
		m.mu.Unlock()
		return Snapshot{}, ErrBusy
	}
	j := &Job{ID: newID(), Project: project, Action: action, Services: services,
		Status: Running, StartedAt: time.Now(), notify: make(chan struct{})}
	m.byID[j.ID] = j
	m.order = append(m.order, j.ID)
	m.active[project] = j.ID
	m.gcLocked()
	snap := Snapshot{Job: *j}
	m.mu.Unlock()

	go m.run(j, args)
	return snap, nil
}

func (m *Manager) run(j *Job, args []string) {
	err := m.Streamer.Stream(context.Background(), args, func(line string) { m.append(j, line) })
	m.mu.Lock()
	defer m.mu.Unlock()
	j.EndedAt = time.Now()
	if err != nil {
		j.Status, j.Error = Failed, err.Error()
	} else {
		j.Status = Succeeded
	}
	delete(m.active, j.Project)
	m.wakeLocked(j)
}

func (m *Manager) append(j *Job, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.lines = append(j.lines, line)
	if len(j.lines) > maxLines {
		drop := len(j.lines) - maxLines
		j.lines = append([]string(nil), j.lines[drop:]...)
		j.offset += drop
	}
	m.wakeLocked(j)
}

func (m *Manager) wakeLocked(j *Job) {
	close(j.notify)
	j.notify = make(chan struct{})
}

// Get returns the job state.
func (m *Manager) Get(id string) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return Snapshot{}, false
	}
	return Snapshot{Job: *j}, true
}

// Follow returns lines after position `from` (a monotonically increasing line
// index), the new position, the job state, and a channel that is closed on the
// next change. Contract: loop; if lines were returned, call Follow again at once;
// otherwise stop when state.Status != Running, else wait on the channel.
func (m *Manager) Follow(id string, from int) (lines []string, next int, state Snapshot, wait <-chan struct{}, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return nil, 0, Snapshot{}, nil, false
	}
	if from < j.offset {
		from = j.offset
	}
	idx := from - j.offset
	lines = append([]string(nil), j.lines[idx:]...)
	return lines, j.offset + len(j.lines), Snapshot{Job: *j}, j.notify, true
}

// Active returns the running job of a project, if any.
func (m *Manager) Active(project string) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.active[project]
	if !ok {
		return Snapshot{}, false
	}
	return Snapshot{Job: *m.byID[id]}, true
}

// Recent returns the newest jobs first.
func (m *Manager) Recent(n int) []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Snapshot
	for i := len(m.order) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, Snapshot{Job: *m.byID[m.order[i]]})
	}
	return out
}

// gcLocked forgets the oldest finished jobs beyond keepDone.
func (m *Manager) gcLocked() {
	done := 0
	for _, id := range m.order {
		if m.byID[id].Status != Running {
			done++
		}
	}
	for done > keepDone && len(m.order) > 0 {
		id := m.order[0]
		if m.byID[id].Status == Running {
			break
		}
		delete(m.byID, id)
		m.order = m.order[1:]
		done--
	}
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
