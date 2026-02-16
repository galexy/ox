package resilience

import (
	"sync"
	"time"
)

// Throttle implements simple rate limiting by enforcing a minimum interval
// between requests.
type Throttle struct {
	mu          sync.Mutex
	minInterval time.Duration
	lastRequest time.Time
}

// ThrottleOption configures a Throttle
type ThrottleOption func(*Throttle)

// WithMinInterval sets the minimum interval between requests
func WithMinInterval(d time.Duration) ThrottleOption {
	return func(t *Throttle) {
		t.minInterval = d
	}
}

// NewThrottle creates a new throttle with sensible defaults
func NewThrottle(opts ...ThrottleOption) *Throttle {
	t := &Throttle{
		minInterval: 100 * time.Millisecond, // 10 req/sec max
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Wait blocks until the minimum interval has passed since the last request.
// If enough time has passed, returns immediately.
func (t *Throttle) Wait() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.lastRequest.IsZero() {
		t.lastRequest = time.Now()
		return
	}

	elapsed := time.Since(t.lastRequest)
	if elapsed < t.minInterval {
		time.Sleep(t.minInterval - elapsed)
	}

	t.lastRequest = time.Now()
}

// MinInterval returns the configured minimum interval
func (t *Throttle) MinInterval() time.Duration {
	return t.minInterval
}

// default throttle instance for the sageox API (lazy initialized)
var (
	defaultThrottleOnce sync.Once
	defaultThrottle     *Throttle
)

// DefaultThrottle returns the default throttle for sageox API calls
func DefaultThrottle() *Throttle {
	defaultThrottleOnce.Do(func() {
		defaultThrottle = NewThrottle()
	})
	return defaultThrottle
}
