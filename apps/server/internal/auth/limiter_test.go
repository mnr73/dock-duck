package auth

import (
	"testing"
	"time"
)

func limiter() (*Limiter, *time.Time) {
	t := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l := NewLimiter()
	l.now = func() time.Time { return t }
	return l, &t
}

func TestBackoffGrowsAndResets(t *testing.T) {
	l, now := limiter()
	k := "1.2.3.4"
	for i := 0; i < freeAttempts; i++ {
		if ok, _ := l.Allow(k); !ok {
			t.Fatalf("attempt %d must be allowed", i+1)
		}
		if d := l.Fail(k); d != 0 {
			t.Errorf("free attempt %d locked for %v", i+1, d)
		}
	}
	d1 := l.Fail(k)
	if d1 != baseLock {
		t.Errorf("first lock %v", d1)
	}
	if ok, wait := l.Allow(k); ok || wait <= 0 || wait > baseLock {
		t.Errorf("locked: %v %v", ok, wait)
	}
	*now = now.Add(baseLock + time.Second)
	if ok, _ := l.Allow(k); !ok {
		t.Error("allowed after the lock")
	}
	d2 := l.Fail(k)
	d3 := l.Fail(k)
	if d2 != 2*baseLock || d3 != 4*baseLock {
		t.Errorf("exponential: %v %v", d2, d3)
	}
	for i := 0; i < 30; i++ {
		l.Fail(k)
	}
	if d := l.Fail(k); d != maxLock {
		t.Errorf("capped at %v, got %v", maxLock, d)
	}
	l.Success(k)
	*now = now.Add(maxLock + time.Minute)
	if ok, _ := l.Allow(k); !ok {
		t.Error("success forgets the failures")
	}
	if d := l.Fail(k); d != 0 {
		t.Errorf("counting restarts: %v", d)
	}
}

func TestClientsAreIndependentAndGlobalCap(t *testing.T) {
	l, now := limiter()
	for i := 0; i < 20; i++ {
		l.Fail("a")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Error("a is locked")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("b must not be affected by a")
	}
	// many different addresses each failing once: the global cap stops them
	l2, now2 := limiter()
	for i := 0; i < globalMaxFail; i++ {
		l2.Fail("ip" + string(rune('A'+i%26)) + string(rune('a'+i/26)))
	}
	if ok, wait := l2.Allow("fresh"); ok || wait <= 0 {
		t.Errorf("address rotation must hit the global cap: %v %v", ok, wait)
	}
	*now2 = now2.Add(globalWindow + time.Second)
	if ok, _ := l2.Allow("fresh"); !ok {
		t.Error("the global window passes")
	}
	_ = now
}

func TestClientKey(t *testing.T) {
	cases := map[string]string{
		"203.0.113.9:5555":                   "203.0.113.9",
		"203.0.113.9":                        "203.0.113.9",
		"[2001:db8:1:2:aaaa:bbbb:cccc:1]:80": "2001:db8:1:2::/64",
		"[2001:db8:1:2:1111:2222:3333:4]:1":  "2001:db8:1:2::/64",
		"[::1]:80":                           "::/64",
	}
	for in, want := range cases {
		if got := ClientKey(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
	if ClientKey("[2001:db8:1:2::1]:1") != ClientKey("[2001:db8:1:2::ffff]:1") {
		t.Error("one /64 is one client")
	}
}
