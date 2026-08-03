package platform

import (
	"sync"
	"time"
)

// LoginLockout tracks consecutive failed login attempts keyed by an
// arbitrary caller-supplied string. identity wires this as
// identifier+IP (see identity/handler.go's loginLockoutKey) rather than
// identifier alone, by design: an attacker throwing bad passwords at a
// victim's account from unrelated IPs shouldn't be able to lock the victim
// out of their own account, but a specific identifier+IP pair that's
// actually misbehaving should get locked.
//
// defaultFailureThreshold=5 failures within defaultFailureWindow=10 minutes
// trigger a defaultCooldown=15 minute lockout of that pair. These numbers
// are deliberately generous for a legitimate user who mistypes a password a
// couple of times in a row, while still making credential-stuffing against
// a single account+IP pair impractical — 5 guesses per ~15 minutes is far
// below any meaningful brute-force rate.
const (
	defaultFailureThreshold = 5
	defaultFailureWindow    = 10 * time.Minute
	defaultCooldown         = 15 * time.Minute
)

type loginAttemptState struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
}

type LoginLockout struct {
	mu       sync.Mutex
	attempts map[string]*loginAttemptState

	failureThreshold int
	failureWindow    time.Duration
	cooldown         time.Duration
}

func NewLoginLockout(failureThreshold int, failureWindow, cooldown time.Duration) *LoginLockout {
	ll := &LoginLockout{
		attempts:         make(map[string]*loginAttemptState),
		failureThreshold: failureThreshold,
		failureWindow:    failureWindow,
		cooldown:         cooldown,
	}
	go ll.cleanupLoop()
	return ll
}

func NewDefaultLoginLockout() *LoginLockout {
	return NewLoginLockout(defaultFailureThreshold, defaultFailureWindow, defaultCooldown)
}

// IsLocked reports whether key is currently under a cooldown lockout. The
// caller (identity's login handler) must check this before doing anything
// else — in particular before the bcrypt password compare — which is the
// entire point of a lockout: it avoids spending bcrypt's deliberately
// expensive compute on an attempt that's going to be rejected regardless,
// and a locked-out caller learns nothing about whether the password they
// tried was actually correct.
func (ll *LoginLockout) IsLocked(key string) bool {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	s, ok := ll.attempts[key]
	if !ok {
		return false
	}
	return time.Now().Before(s.lockedUntil)
}

// RecordFailure counts a failed attempt for key. If the previous failure
// fell outside the rolling failureWindow, the counter starts over instead
// of accumulating indefinitely. Once failureThreshold consecutive failures
// land inside the window, a cooldown lockout starts immediately.
func (ll *LoginLockout) RecordFailure(key string) {
	now := time.Now()

	ll.mu.Lock()
	defer ll.mu.Unlock()

	s, ok := ll.attempts[key]
	if !ok || now.Sub(s.lastFailure) > ll.failureWindow {
		s = &loginAttemptState{}
		ll.attempts[key] = s
	}

	s.failures++
	s.lastFailure = now

	if s.failures >= ll.failureThreshold {
		s.lockedUntil = now.Add(ll.cooldown)
	}
}

// RecordSuccess clears any tracked failures for key immediately — a
// successful login resets the counter rather than waiting for the rolling
// window to expire on its own.
func (ll *LoginLockout) RecordSuccess(key string) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	delete(ll.attempts, key)
}

// cleanupLoop periodically drops state for keys that are neither locked nor
// within their failure window, so a long-running process doesn't
// accumulate one entry per identifier+IP pair ever attempted.
func (ll *LoginLockout) cleanupLoop() {
	ticker := time.NewTicker(ll.failureWindow)
	defer ticker.Stop()
	for now := range ticker.C {
		ll.mu.Lock()
		for k, s := range ll.attempts {
			if now.After(s.lockedUntil) && now.Sub(s.lastFailure) > ll.failureWindow {
				delete(ll.attempts, k)
			}
		}
		ll.mu.Unlock()
	}
}
