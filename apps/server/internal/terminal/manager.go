package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Limits and lifetimes.
const (
	MaxSessions     = 16
	ReplayBytes     = 256 << 10
	ExitedKeepFor   = 10 * time.Minute // a finished session stays listed for a while
	DetachedKeepFor = 6 * time.Hour    // a running session nobody is viewing is closed after this
	CleanupWait     = 8 * time.Second  // on shutdown, how long to wait for in-container cleanups
)

// Manager owns all terminal sessions of the server.
type Manager struct {
	spawn Spawner

	mu       sync.Mutex
	sessions map[string]*Session
	stop     chan struct{}
	once     sync.Once
	closing  atomic.Bool
}

// ShuttingDown tells viewers that a session ended because the server is stopping,
// not because the shell exited on its own.
func (m *Manager) ShuttingDown() bool { return m.closing.Load() }

// NewManager starts the reaper; call Close on shutdown to kill every session.
func NewManager(spawn Spawner) *Manager {
	m := &Manager{spawn: spawn, sessions: map[string]*Session{}, stop: make(chan struct{})}
	go m.reap()
	return m
}

// Open starts argv in a new session.
func (m *Manager) Open(meta Meta, argv []string, cols, rows uint16) (*Session, error) {
	return m.OpenTagged(meta, func(string) ([]string, string, func(), error) { return argv, "", nil, nil }, cols, rows)
}

// OpenTagged starts a session whose command is built from the session id.
// build returns the argv, a human readable equivalent and an optional cleanup that runs when the session is
// closed, reaped or the server shuts down while the process is still alive (for
// docker exec: end what the shell left running inside the container).
func (m *Manager) OpenTagged(meta Meta, build func(id string) (argv []string, display string, cleanup func(), err error), cols, rows uint16) (*Session, error) {
	if cols < 2 || cols > 1000 {
		cols = 80
	}
	if rows < 1 || rows > 1000 {
		rows = 24
	}
	m.mu.Lock()
	if len(m.sessions) >= MaxSessions {
		m.mu.Unlock()
		return nil, ErrTooMany
	}
	m.mu.Unlock()

	id := newID()
	argv, display, cleanup, err := build(id)
	if err != nil {
		return nil, err
	}
	meta.Display = display
	p, err := m.spawn(argv, cols, rows)
	if err != nil {
		return nil, err
	}
	s := newSession(id, meta, p, cols, rows, ReplayBytes)
	s.cleanup = cleanup
	if cleanup != nil {
		go func() { // the shell ended (on its own, or killed): sweep what it left behind
			<-s.done
			s.sweep()
		}()
	}
	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()
	return s, nil
}

// Get returns a session by id.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	return s, ok
}

// List returns all sessions, oldest first.
func (m *Manager) List() []Info {
	m.mu.Lock()
	all := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		all = append(all, s)
	}
	m.mu.Unlock()
	out := make([]Info, 0, len(all))
	for _, s := range all {
		out = append(out, s.Info())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started.Before(out[j].Started) })
	return out
}

// Close ends a session: the process is killed and the session is forgotten.
func (m *Manager) Close(id string) bool {
	m.mu.Lock()
	s, ok := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if !ok {
		return false
	}
	m.kill(s, false)
	return true
}

// kill hangs the session up and runs the cleanup, even when the shell already
// exited: it may have left background jobs (`cmd &`, nohup) running in the container.
// The cleanup runs in the background, or is waited for on shutdown.
func (m *Manager) kill(s *Session, wait bool) {
	_ = s.p.Kill() // harmless when the process already ended
	if s.cleanup == nil {
		return
	}
	done := make(chan struct{})
	go func() { s.sweep(); close(done) }()
	if wait {
		select {
		case <-done:
		case <-time.After(CleanupWait):
		}
	}
}

// Shutdown kills every session and stops the reaper.
func (m *Manager) Shutdown() {
	m.once.Do(func() { close(m.stop) })
	m.closing.Store(true)
	m.mu.Lock()
	all := m.sessions
	m.sessions = map[string]*Session{}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, s := range all {
		wg.Add(1)
		go func(s *Session) { defer wg.Done(); m.kill(s, true) }(s)
	}
	wg.Wait()
}

func (m *Manager) reap() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			m.reapOnce(time.Now())
		}
	}
}

func (m *Manager) reapOnce(now time.Time) {
	var kill []*Session
	m.mu.Lock()
	for id, s := range m.sessions {
		s.mu.Lock()
		attached := s.att != nil
		switch {
		case s.exited && !attached && now.Sub(s.finished) > ExitedKeepFor:
			delete(m.sessions, id)
		case !s.exited && !attached && now.Sub(s.lastSeen) > DetachedKeepFor:
			delete(m.sessions, id)
			kill = append(kill, s)
		}
		s.mu.Unlock()
	}
	m.mu.Unlock()
	for _, s := range kill {
		m.kill(s, false)
	}
}

// NewMarker returns a fresh session marker (also used as the session id).
func NewMarker() string { return newID() }

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
