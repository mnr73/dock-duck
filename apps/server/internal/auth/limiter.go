package auth

import (
	"net"
	"sync"
	"time"
)

// Login throttling. Per client: a few free attempts, then an exponentially growing
// lock. Globally: a cap on failures per window, which also stops an attacker who
// rotates addresses (at the price that a flood of bad logins can briefly lock the
// owner out too; the owner can always run `dockduck passwd` on the machine).
const (
	freeAttempts  = 4
	baseLock      = 5 * time.Second
	maxLock       = 15 * time.Minute
	forgetAfter   = time.Hour
	globalWindow  = 10 * time.Minute
	globalMaxFail = 100
)

type attempts struct {
	fails int
	last  time.Time
	until time.Time
}

// Limiter tracks failed logins.
type Limiter struct {
	mu     sync.Mutex
	now    func() time.Time
	byKey  map[string]*attempts
	global []time.Time
}

// NewLimiter returns an empty limiter.
func NewLimiter() *Limiter { return &Limiter{now: time.Now, byKey: map[string]*attempts{}} }

// ClientKey groups IPv6 addresses by /64 (a client owns a whole /64), IPv4 as is.
func ClientKey(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

// Allow says whether a login attempt may be checked now, else how long to wait.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	if len(l.global) >= globalMaxFail {
		return false, l.global[0].Add(globalWindow).Sub(now)
	}
	if a := l.byKey[key]; a != nil && now.Before(a.until) {
		return false, a.until.Sub(now)
	}
	return true, 0
}

// Fail records a wrong password and returns the lock it caused (0 = none yet).
func (l *Limiter) Fail(key string) time.Duration {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.global = append(l.global, now)
	a := l.byKey[key]
	if a == nil {
		a = &attempts{}
		l.byKey[key] = a
	}
	a.fails++
	a.last = now
	if a.fails <= freeAttempts {
		return 0
	}
	lock := baseLock
	for i := freeAttempts + 1; i < a.fails && lock < maxLock; i++ {
		lock *= 2
	}
	if lock > maxLock {
		lock = maxLock
	}
	a.until = now.Add(lock)
	return lock
}

// Success forgets a client's failures.
func (l *Limiter) Success(key string) {
	l.mu.Lock()
	delete(l.byKey, key)
	l.mu.Unlock()
}

func (l *Limiter) pruneLocked(now time.Time) {
	cut := now.Add(-globalWindow)
	i := 0
	for i < len(l.global) && l.global[i].Before(cut) {
		i++
	}
	l.global = l.global[i:]
	for k, a := range l.byKey {
		if now.Sub(a.last) > forgetAfter && now.After(a.until) {
			delete(l.byKey, k)
		}
	}
}
