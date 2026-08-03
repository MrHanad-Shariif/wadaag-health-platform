package platform

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a fixed-window per-IP request counter applied once at the
// router level so it covers the whole /api/v1 surface, not just auth. A
// fixed window (rather than a sliding window or token bucket) is
// deliberately the simplest thing that works here: golang.org/x/time isn't
// already a dependency (checked go.mod before writing this), and this app
// has no burst-shaping requirement sophisticated enough to justify adding
// one just for this — a hand-rolled mutex+map counter with periodic
// cleanup is a few dozen lines and one goroutine.
//
// defaultRateLimit=120 requests/minute per IP was picked to comfortably
// cover normal interactive use (a single page load can fan out to a
// handful of API calls, and the other Phase 0 tasks' own manual testing
// made several requests in a row against auth endpoints) while still
// capping a scripted burst well below anything that could meaningfully
// hammer the database. 120/min = 2/sec sustained, which no legitimate
// browser session comes close to.
const (
	defaultRateLimit  = 120
	defaultRateWindow = time.Minute
)

type rateBucket struct {
	count      int
	windowEnds time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

// NewRateLimiter constructs a limiter with the given per-window request
// budget. NewDefaultRateLimiter is what most callers want.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*rateBucket),
		limit:   limit,
		window:  window,
	}
	go rl.cleanupLoop()
	return rl
}

func NewDefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(defaultRateLimit, defaultRateWindow)
}

// allow reports whether key may proceed, advancing or resetting its window
// as needed. A window is reset lazily on the first request after it
// expires rather than on a timer, so idle keys cost nothing between
// requests.
func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok || now.After(b.windowEnds) {
		rl.buckets[key] = &rateBucket{count: 1, windowEnds: now.Add(rl.window)}
		return true
	}

	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// cleanupLoop periodically drops buckets whose window has already expired
// so a long-running process doesn't accumulate one entry per IP ever seen.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for now := range ticker.C {
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if now.After(b.windowEnds) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware rejects requests over the configured per-IP limit with 429,
// using the same error shape as every other handler (platform.WriteError).
// Must run after chimiddleware.RealIP so RemoteAddr is already normalized
// to the real client address rather than a proxy hop.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(ClientIP(r)) {
			WriteError(w, http.StatusTooManyRequests, "too many requests, please slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP extracts the host portion of RemoteAddr, stripping the port
// chimiddleware.RealIP leaves attached. Falls back to the raw value if it
// isn't in host:port form. Shared by the rate limiter and identity's login
// lockout so both key on the same notion of "the caller's IP".
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
