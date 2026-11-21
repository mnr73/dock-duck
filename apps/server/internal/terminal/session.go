// Package terminal runs interactive `docker exec` sessions under a real PTY and
// keeps them alive on the server, so closing a browser tab, reloading the page or
// switching to another page does not kill the shell: the client re-attaches and
// the recent output is replayed.
package terminal

import (
	"errors"
	"io"
	"sync"
	"time"
)

// Process is a running command attached to a PTY.
type Process interface {
	io.ReadWriter
	Resize(cols, rows uint16) error
	// Wait blocks until the command ends and returns its exit code.
	Wait() (int, error)
	// Kill hangs the terminal up and, if that is not enough, kills the command.
	Kill() error
}

// Spawner starts argv under a PTY of the given size.
type Spawner func(argv []string, cols, rows uint16) (Process, error)

// Meta describes what a session is attached to (for the UI).
type Meta struct {
	Project     string `json:"project"` // project id
	ProjectName string `json:"project_name"`
	Service     string `json:"service"`
	Container   string `json:"container"`
	User        string `json:"user"`    // "" = the container's default user
	Shell       string `json:"shell"`   // "" = auto-detected
	Command     string `json:"command"` // "" = an interactive shell
	// Display is the equivalent command to run by hand.
	Display string `json:"display"`
}

// Info is the JSON view of a session.
type Info struct {
	ID       string `json:"id"`
	Meta     `json:",inline"`
	Cols     uint16    `json:"cols"`
	Rows     uint16    `json:"rows"`
	Started  time.Time `json:"started"`
	State    string    `json:"state"` // running | exited
	ExitCode *int      `json:"exit_code,omitempty"`
	Attached bool      `json:"attached"`
}

// ErrTooMany is returned when the session limit is reached.
var ErrTooMany = errors.New("too many open terminal sessions")

// Attachment is a viewer of a session. Output arrives on C; C is closed when the
// viewer was replaced by another one, was too slow, or the session ended.
type Attachment struct {
	C      <-chan []byte
	ch     chan []byte
	reason string
	closed bool
}

// Reason tells why C was closed: "taken" (another viewer attached), "slow", or ""
// (the session ended or the viewer detached).
func (a *Attachment) Reason() string { return a.reason }

// Session is one running (or just exited) terminal.
type Session struct {
	id      string
	meta    Meta
	p       Process
	started time.Time

	mu         sync.Mutex
	cols, rows uint16
	ring       *ring
	att        *Attachment
	lastSeen   time.Time // last time a viewer was attached/detached
	done       chan struct{}
	finished   time.Time
	exitCode   int
	exited     bool
	cleanup    func() // ends what the process left behind; see Manager.OpenTagged
	cleanOnce  sync.Once
}

// sweep runs the cleanup once, however many paths (shell exit, close, reaper,
// shutdown) ask for it.
func (s *Session) sweep() {
	if s.cleanup != nil {
		s.cleanOnce.Do(s.cleanup)
	}
}

func newSession(id string, meta Meta, p Process, cols, rows uint16, ringSize int) *Session {
	s := &Session{id: id, meta: meta, p: p, started: time.Now(), cols: cols, rows: rows,
		ring: newRing(ringSize), lastSeen: time.Now(), done: make(chan struct{})}
	go s.pump()
	return s
}

// pump copies PTY output to the replay buffer and the attached viewer.
func (s *Session) pump() {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.p.Read(buf)
		if n > 0 {
			s.emit(buf[:n])
		}
		if err != nil {
			break
		}
	}
	code, _ := s.p.Wait()
	s.mu.Lock()
	s.exited, s.exitCode, s.finished = true, code, time.Now()
	if a := s.att; a != nil {
		s.closeAtt(a, "")
		s.att = nil
	}
	s.mu.Unlock()
	close(s.done)
}

func (s *Session) emit(b []byte) {
	cp := append([]byte(nil), b...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ring.write(cp)
	if a := s.att; a != nil {
		select {
		case a.ch <- cp:
		default: // viewer cannot keep up: drop it; it re-attaches and gets a replay
			s.closeAtt(a, "slow")
			s.att = nil
		}
	}
}

func (s *Session) closeAtt(a *Attachment, reason string) {
	if a.closed {
		return
	}
	a.closed, a.reason = true, reason
	close(a.ch)
}

// Attach registers a viewer and returns the output so far. A previous viewer is
// closed with reason "taken".
func (s *Session) Attach() (replay []byte, a *Attachment) {
	ch := make(chan []byte, 512)
	a = &Attachment{C: ch, ch: ch}
	s.mu.Lock()
	defer s.mu.Unlock()
	replay = s.ring.bytes()
	if s.att != nil {
		s.closeAtt(s.att, "taken")
	}
	if s.exited {
		s.closeAtt(a, "")
		return replay, a
	}
	s.att = a
	s.lastSeen = time.Now()
	return replay, a
}

// Detach removes the viewer (a no-op when it was already replaced).
func (s *Session) Detach(a *Attachment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.att == a {
		s.att = nil
		s.lastSeen = time.Now()
	}
	s.closeAtt(a, "")
}

// Write sends keyboard input to the process.
func (s *Session) Write(b []byte) error {
	select {
	case <-s.done:
		return io.ErrClosedPipe
	default:
	}
	_, err := s.p.Write(b)
	return err
}

// Resize changes the terminal size; the process gets SIGWINCH.
func (s *Session) Resize(cols, rows uint16) error {
	if cols < 2 || rows < 1 || cols > 1000 || rows > 1000 {
		return errors.New("invalid terminal size")
	}
	if err := s.p.Resize(cols, rows); err != nil {
		return err
	}
	s.mu.Lock()
	s.cols, s.rows = cols, rows
	s.mu.Unlock()
	return nil
}

// Done is closed when the process has ended.
func (s *Session) Done() <-chan struct{} { return s.done }

// ID of the session.
func (s *Session) ID() string { return s.id }

// Exit returns the exit code once Done is closed.
func (s *Session) Exit() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

// Info returns the JSON view.
func (s *Session) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := Info{ID: s.id, Meta: s.meta, Cols: s.cols, Rows: s.rows, Started: s.started, State: "running", Attached: s.att != nil}
	if s.exited {
		c := s.exitCode
		i.State, i.ExitCode = "exited", &c
	}
	return i
}

// ring keeps the last `max` bytes of output.
type ring struct {
	max       int
	b         []byte
	truncated bool
}

func newRing(max int) *ring { return &ring{max: max} }

func (r *ring) write(p []byte) {
	r.b = append(r.b, p...)
	if over := len(r.b) - r.max; over > 0 {
		r.b = append(r.b[:0], r.b[over:]...)
		r.truncated = true
	}
}

// bytes returns a copy; after a truncation the start is moved to a line start
// (and off a UTF-8 continuation byte) so the replay does not begin mid-sequence.
func (r *ring) bytes() []byte {
	b := r.b
	if r.truncated {
		limit := len(b)
		if limit > 4096 {
			limit = 4096
		}
		cut := -1
		for i := 0; i < limit; i++ {
			if b[i] == '\n' {
				cut = i + 1
				break
			}
		}
		if cut < 0 {
			cut = 0
			for cut < limit && b[cut]&0xC0 == 0x80 {
				cut++
			}
		}
		b = b[cut:]
	}
	return append([]byte(nil), b...)
}
