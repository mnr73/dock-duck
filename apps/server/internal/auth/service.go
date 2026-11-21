package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"dockduck/internal/config"
)

// CookieName of the session cookie.
const CookieName = "dockduck_session"

var (
	// ErrAlreadyConfigured: setup only works while no password exists.
	ErrAlreadyConfigured = errors.New("a password is already set")
	// ErrBadCredentials is deliberately the only login failure the client sees.
	ErrBadCredentials = errors.New("wrong password")
	// ErrBusy: too many password checks are running at once.
	ErrBusy = errors.New("too many login attempts at once")
	// ErrNotConfigured: there is no password to check.
	ErrNotConfigured = errors.New("no password is set")
)

// LockedError is returned while a client is locked out.
type LockedError struct{ Wait time.Duration }

func (e *LockedError) Error() string { return "too many failed attempts" }

// Service ties the password (persisted in config), sessions and throttling together.
type Service struct {
	Store    *config.Store
	Sessions *Sessions
	Limiter  *Limiter
	// Required: the server is reachable from other machines, so it must never run
	// open even if the password disappears from the config file.
	Required bool

	verifying chan struct{} // bounds concurrent (memory-hungry) argon2 runs
	mu        sync.Mutex
	checked   time.Time
}

// NewService returns a service over store.
func NewService(store *config.Store, required bool) *Service {
	return &Service{Store: store, Sessions: NewSessions(), Limiter: NewLimiter(), Required: required, verifying: make(chan struct{}, 4)}
}

// hash returns the stored hash, re-reading the file at most once a second so that
// `dockduck passwd` on the machine takes effect in a running server.
func (s *Service) hash() string {
	s.mu.Lock()
	if time.Since(s.checked) > time.Second {
		s.checked = time.Now()
		s.mu.Unlock()
		s.Store.Refresh()
	} else {
		s.mu.Unlock()
	}
	return s.Store.Get().Auth.PasswordHash
}

// Configured reports whether a password exists.
func (s *Service) Configured() bool { return s.hash() != "" }

// fingerprint ties sessions to one password: changing it invalidates them all.
func fingerprint(hash string) string {
	sum := sha256.Sum256([]byte(hash))
	return hex.EncodeToString(sum[:8])
}

// FromRequest returns the session of a request's cookie.
func (s *Service) FromRequest(r *http.Request) (Session, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return Session{}, false
	}
	h := s.hash()
	if h == "" {
		return Session{}, false
	}
	return s.Sessions.Get(c.Value, fingerprint(h))
}

// Setup sets the first password (and starts a session). The caller decides who is
// allowed to call it (the HTTP layer only allows local requests).
func (s *Service) Setup(pw, ip string) (string, Session, error) {
	if err := CheckPolicy(pw); err != nil {
		return "", Session{}, err
	}
	hash, err := Hash(pw)
	if err != nil {
		return "", Session{}, err
	}
	err = s.Store.Update(func(c *config.Config) error {
		if c.Auth.PasswordHash != "" { // checked under the store's lock: two racing setups cannot both win
			return ErrAlreadyConfigured
		}
		c.Auth.PasswordHash = hash
		return nil
	})
	if err != nil {
		return "", Session{}, err
	}
	tok, sess := s.Sessions.Create(fingerprint(hash), ip)
	return tok, sess, nil
}

// check verifies pw for a client, applying throttling. It returns the current hash.
func (s *Service) check(pw, clientKey string) (string, error) {
	h := s.hash()
	if h == "" {
		return "", ErrNotConfigured
	}
	if ok, wait := s.Limiter.Allow(clientKey); !ok {
		return "", &LockedError{Wait: wait}
	}
	select {
	case s.verifying <- struct{}{}:
		defer func() { <-s.verifying }()
	default:
		return "", ErrBusy
	}
	ok, err := Verify(pw, h)
	if err != nil || !ok {
		if wait := s.Limiter.Fail(clientKey); wait > 0 {
			return "", &LockedError{Wait: wait}
		}
		return "", ErrBadCredentials
	}
	s.Limiter.Success(clientKey)
	return h, nil
}

// Login checks the password and starts a session.
func (s *Service) Login(pw, ip, clientKey string) (string, Session, error) {
	h, err := s.check(pw, clientKey)
	if err != nil {
		return "", Session{}, err
	}
	tok, sess := s.Sessions.Create(fingerprint(h), ip)
	return tok, sess, nil
}

// ChangePassword verifies the current password, sets the new one and ends every
// session (including this one: the caller gets a fresh one).
func (s *Service) ChangePassword(current, next, ip, clientKey string) (string, Session, error) {
	if _, err := s.check(current, clientKey); err != nil {
		return "", Session{}, err
	}
	if err := CheckPolicy(next); err != nil {
		return "", Session{}, err
	}
	hash, err := Hash(next)
	if err != nil {
		return "", Session{}, err
	}
	if err := s.Store.Update(func(c *config.Config) error { c.Auth.PasswordHash = hash; return nil }); err != nil {
		return "", Session{}, err
	}
	s.Sessions.DeleteAll()
	tok, sess := s.Sessions.Create(fingerprint(hash), ip)
	return tok, sess, nil
}
