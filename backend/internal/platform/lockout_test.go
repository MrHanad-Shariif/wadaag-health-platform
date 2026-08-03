package platform

import (
	"testing"
	"time"
)

// TestLoginLockout exercises the exact stateful sequence described in the
// task: 5th failure locks, a 6th attempt during cooldown is rejected
// (without ever reaching a password compare — IsLocked is checked before
// RecordFailure/RecordSuccess would run), and cooldown expiry resets the
// counter so the key is usable again.
func TestLoginLockout(t *testing.T) {
	const key = "user@example.com|1.2.3.4"

	t.Run("5th consecutive failure locks, 6th is rejected during cooldown", func(t *testing.T) {
		ll := NewLoginLockout(5, 10*time.Minute, 15*time.Minute)

		for i := 1; i <= 4; i++ {
			if ll.IsLocked(key) {
				t.Fatalf("locked after only %d failures, want unlocked", i)
			}
			ll.RecordFailure(key)
		}
		if ll.IsLocked(key) {
			t.Fatal("locked after 4 failures, want unlocked")
		}

		ll.RecordFailure(key) // 5th failure
		if !ll.IsLocked(key) {
			t.Fatal("not locked after 5th failure, want locked")
		}

		// A 6th attempt must be rejected purely by IsLocked, before any
		// password compare would run — simulated here by never calling
		// RecordFailure/RecordSuccess for it.
		if !ll.IsLocked(key) {
			t.Fatal("6th attempt not rejected during cooldown")
		}
	})

	t.Run("successful login resets the counter immediately", func(t *testing.T) {
		ll := NewLoginLockout(5, 10*time.Minute, 15*time.Minute)

		for i := 0; i < 4; i++ {
			ll.RecordFailure(key)
		}
		ll.RecordSuccess(key)
		if ll.IsLocked(key) {
			t.Fatal("locked after success, want unlocked")
		}

		// The counter should have started over, not just unlocked — a
		// single subsequent failure shouldn't be treated as the 5th.
		ll.RecordFailure(key)
		if ll.IsLocked(key) {
			t.Fatal("locked after 1 failure post-reset, want unlocked")
		}
	})

	t.Run("failure outside the rolling window doesn't accumulate", func(t *testing.T) {
		ll := NewLoginLockout(5, 10*time.Minute, 15*time.Minute)

		for i := 0; i < 4; i++ {
			ll.RecordFailure(key)
		}
		// Simulate the window having elapsed by rewriting lastFailure into
		// the past directly (no clock injection in the public API).
		ll.mu.Lock()
		ll.attempts[key].lastFailure = time.Now().Add(-11 * time.Minute)
		ll.mu.Unlock()

		ll.RecordFailure(key) // should restart the counter at 1, not hit 5
		if ll.IsLocked(key) {
			t.Fatal("locked after window reset + 1 failure, want unlocked")
		}
	})

	t.Run("cooldown expiry unlocks the key", func(t *testing.T) {
		ll := NewLoginLockout(5, 10*time.Minute, 15*time.Minute)

		for i := 0; i < 5; i++ {
			ll.RecordFailure(key)
		}
		if !ll.IsLocked(key) {
			t.Fatal("expected locked after 5 failures")
		}

		// Simulate cooldown having elapsed.
		ll.mu.Lock()
		ll.attempts[key].lockedUntil = time.Now().Add(-time.Second)
		ll.mu.Unlock()

		if ll.IsLocked(key) {
			t.Fatal("still locked after cooldown expiry, want unlocked")
		}
	})

	t.Run("different key is unaffected", func(t *testing.T) {
		ll := NewLoginLockout(5, 10*time.Minute, 15*time.Minute)

		for i := 0; i < 5; i++ {
			ll.RecordFailure(key)
		}
		if !ll.IsLocked(key) {
			t.Fatal("expected key locked")
		}
		if ll.IsLocked("someone-else@example.com|9.9.9.9") {
			t.Fatal("unrelated key locked out, want unaffected")
		}
	})
}
