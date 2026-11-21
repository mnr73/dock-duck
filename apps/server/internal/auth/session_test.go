package auth

import (
	"strings"
	"testing"
	"time"
)

func clocked() (*Sessions, *time.Time) {
	t := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := NewSessions()
	s.now = func() time.Time { return t }
	return s, &t
}

func TestSessionLifecycle(t *testing.T) {
	s, now := clocked()
	tok, sess := s.Create("fp1", "10.0.0.1")
	if len(tok) < 40 || sess.CSRF == "" || sess.CSRF == tok {
		t.Errorf("token/csrf: %q %q", tok, sess.CSRF)
	}
	if got, ok := s.Get(tok, "fp1"); !ok || got.CSRF != sess.CSRF {
		t.Fatal("valid session")
	}
	for _, bad := range []string{"", "nope", tok + "x", strings.ToUpper(tok)} {
		if _, ok := s.Get(bad, "fp1"); ok {
			t.Errorf("token %q must not work", bad)
		}
	}
	// stored hashed: the raw token is not a key anywhere
	for k := range s.m {
		if k == tok {
			t.Error("tokens must be stored hashed")
		}
	}
	// sliding idle timeout
	*now = now.Add(IdleTimeout - time.Minute)
	if _, ok := s.Get(tok, "fp1"); !ok {
		t.Error("still inside the idle window")
	}
	*now = now.Add(IdleTimeout - time.Minute)
	if _, ok := s.Get(tok, "fp1"); !ok {
		t.Error("use renews the idle window")
	}
	*now = now.Add(IdleTimeout + time.Minute)
	if _, ok := s.Get(tok, "fp1"); ok {
		t.Error("idle for too long")
	}
}

func TestAbsoluteExpiryEvenWhenActive(t *testing.T) {
	s, now := clocked()
	tok, _ := s.Create("fp", "ip")
	for i := 0; i < 20; i++ {
		*now = now.Add(11 * time.Hour)
		if _, ok := s.Get(tok, "fp"); !ok {
			break
		}
	}
	if _, ok := s.Get(tok, "fp"); ok {
		t.Error("a session must end after the absolute lifetime however active it is")
	}
}

func TestPasswordChangeEndsSessions(t *testing.T) {
	s, _ := clocked()
	tok, _ := s.Create("old-hash", "ip")
	if _, ok := s.Get(tok, "new-hash"); ok {
		t.Error("a session created under another password must be invalid")
	}
	if _, ok := s.Get(tok, "old-hash"); ok {
		t.Error("and it is dropped, not just hidden")
	}
}

func TestDeleteAndLimit(t *testing.T) {
	s, now := clocked()
	a, _ := s.Create("fp", "ip")
	b, _ := s.Create("fp", "ip")
	s.Delete(a)
	if _, ok := s.Get(a, "fp"); ok {
		t.Error("logout")
	}
	if _, ok := s.Get(b, "fp"); !ok {
		t.Error("other sessions unaffected")
	}
	s.DeleteAll()
	if s.Len() != 0 {
		t.Error("delete all")
	}
	first, _ := s.Create("fp", "ip")
	for i := 0; i < maxSessions+10; i++ {
		*now = now.Add(time.Second)
		s.Create("fp", "ip")
	}
	if s.Len() > maxSessions {
		t.Errorf("bounded: %d", s.Len())
	}
	if _, ok := s.Get(first, "fp"); ok {
		t.Error("the least recently used session is the one evicted")
	}
}

func TestDoneIsClosedOnEveryRevocation(t *testing.T) {
	s, now := clocked()
	closed := func(c <-chan struct{}) bool {
		select {
		case <-c:
			return true
		default:
			return false
		}
	}
	tok, sess := s.Create("fp", "ip")
	if closed(sess.Done()) {
		t.Fatal("alive")
	}
	got, _ := s.Get(tok, "fp")
	s.Delete(tok)
	if !closed(sess.Done()) || !closed(got.Done()) {
		t.Error("logout closes Done (also for copies handed out earlier)")
	}
	s.Delete(tok) // idempotent: must not panic on a second close

	_, a := s.Create("fp", "ip")
	_, b := s.Create("fp", "ip")
	s.DeleteAll()
	if !closed(a.Done()) || !closed(b.Done()) {
		t.Error("password change ends all")
	}

	tok, c := s.Create("fp", "ip")
	*now = now.Add(IdleTimeout + time.Minute)
	s.Get(tok, "fp")
	if !closed(c.Done()) {
		t.Error("expiry closes Done")
	}
	tok, d := s.Create("old", "ip")
	s.Get(tok, "new")
	if !closed(d.Done()) {
		t.Error("a changed password closes Done")
	}
}
