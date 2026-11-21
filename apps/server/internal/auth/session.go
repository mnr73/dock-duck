package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"sync"
	"time"
)

// Session lifetimes: sliding idle timeout and an absolute cap.
const (
	IdleTimeout     = 12 * time.Hour
	AbsoluteTimeout = 7 * 24 * time.Hour
	maxSessions     = 64
)

// Session is one logged-in browser.
type Session struct {
	CSRF    string
	Created time.Time
	Seen    time.Time
	IP      string
	fp      string // fingerprint of the password hash it was created under
	done    chan struct{}
}

// Done is closed when the session is revoked (logout, password change, eviction,
// expiry). Long-lived connections (WebSockets) watch it so that logging out also
// ends a terminal that is already open.
func (s Session) Done() <-chan struct{} { return s.done }

// Sessions keeps sessions in memory only: a server restart logs everyone out,
// and tokens are stored hashed so a memory/core dump does not hand out live cookies.
type Sessions struct {
	mu  sync.Mutex
	m   map[string]*Session
	now func() time.Time
}

// NewSessions returns an empty store.
func NewSessions() *Sessions { return &Sessions{m: map[string]*Session{}, now: time.Now} }

func tokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func random(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system RNG failing is not recoverable
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Create starts a session bound to the current password fingerprint.
func (s *Sessions) Create(fp, ip string) (token string, sess Session) {
	token = random(32)
	now := s.now()
	ss := &Session{CSRF: random(24), Created: now, Seen: now, IP: ip, fp: fp, done: make(chan struct{})}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(now)
	if len(s.m) >= maxSessions { // evict the least recently used
		var oldest string
		for k, v := range s.m {
			if oldest == "" || v.Seen.Before(s.m[oldest].Seen) {
				oldest = k
			}
		}
		s.dropLocked(oldest)
	}
	s.m[tokenKey(token)] = ss
	return token, *ss
}

// Get returns the session when the token is valid, not expired and was created
// under the current password (changing the password ends every session). It
// slides the idle timeout.
func (s *Sessions) Get(token, fp string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	k := tokenKey(token)
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.m[k]
	if !ok {
		return Session{}, false
	}
	if now.Sub(ss.Seen) > IdleTimeout || now.Sub(ss.Created) > AbsoluteTimeout || subtle.ConstantTimeCompare([]byte(ss.fp), []byte(fp)) != 1 {
		s.dropLocked(k)
		return Session{}, false
	}
	ss.Seen = now
	return *ss, true
}

// Delete ends one session.
func (s *Sessions) Delete(token string) {
	s.mu.Lock()
	s.dropLocked(tokenKey(token))
	s.mu.Unlock()
}

// DeleteAll ends every session.
func (s *Sessions) DeleteAll() {
	s.mu.Lock()
	for k := range s.m {
		s.dropLocked(k)
	}
	s.mu.Unlock()
}

// dropLocked removes a session and signals everything watching it. Removing it
// from the map first guarantees done is closed exactly once.
func (s *Sessions) dropLocked(k string) {
	if ss, ok := s.m[k]; ok {
		delete(s.m, k)
		close(ss.done)
	}
}

// Len is the number of live sessions.
func (s *Sessions) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(s.now())
	return len(s.m)
}

func (s *Sessions) sweepLocked(now time.Time) {
	for k, v := range s.m {
		if now.Sub(v.Seen) > IdleTimeout || now.Sub(v.Created) > AbsoluteTimeout {
			s.dropLocked(k)
		}
	}
}
