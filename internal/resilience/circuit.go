package resilience

import (
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation
	CircuitOpen                         // failing fast
	CircuitHalfOpen                     // testing recovery
)

// CircuitBreaker implements the circuit breaker pattern for external service calls.
// When failures exceed the threshold, the circuit "opens" and can optionally
// skip requests to fail fast.
type CircuitBreaker struct {
	mu sync.Mutex

	failures    int
	successes   int
	threshold   int           // failures before opening circuit
	retryChance float64       // probability of retry when open (1.0 = always retry)
	resetAfter  time.Duration // reset to closed after this duration of success

	state       CircuitState
	lastFailure time.Time
	lastSuccess time.Time
}

// CircuitBreakerOption configures a CircuitBreaker
type CircuitBreakerOption func(*CircuitBreaker)

// WithThreshold sets the failure threshold before opening circuit
func WithThreshold(n int) CircuitBreakerOption {
	return func(cb *CircuitBreaker) {
		cb.threshold = n
	}
}

// WithRetryChance sets probability of retry when circuit is open (0.0-1.0)
func WithRetryChance(chance float64) CircuitBreakerOption {
	return func(cb *CircuitBreaker) {
		cb.retryChance = chance
	}
}

// WithResetAfter sets duration after which circuit resets to closed
func WithResetAfter(d time.Duration) CircuitBreakerOption {
	return func(cb *CircuitBreaker) {
		cb.resetAfter = d
	}
}

// NewCircuitBreaker creates a new circuit breaker with sensible defaults
func NewCircuitBreaker(opts ...CircuitBreakerOption) *CircuitBreaker {
	cb := &CircuitBreaker{
		threshold:   3,   // open after 3 failures
		retryChance: 1.0, // always retry (100%)
		resetAfter:  30 * time.Second,
		state:       CircuitClosed,
	}

	for _, opt := range opts {
		opt(cb)
	}

	return cb
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ShouldAttempt returns true if a request should be attempted.
// With retryChance=1.0 (default), this always returns true.
func (cb *CircuitBreaker) ShouldAttempt() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// with retryChance=1.0, we always attempt
		// future: could use rand.Float64() < cb.retryChance
		return cb.retryChance >= 1.0
	case CircuitHalfOpen:
		return true
	}

	return true
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	cb.lastSuccess = time.Now()

	switch cb.state {
	case CircuitHalfOpen:
		// success in half-open means we can close
		cb.state = CircuitClosed
		cb.failures = 0
	case CircuitOpen:
		// success while open (we attempted anyway) - move to half-open
		cb.state = CircuitHalfOpen
		cb.failures = 0
	case CircuitClosed:
		// reset failure count on success
		cb.failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.threshold {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		// failure in half-open means back to open
		cb.state = CircuitOpen
	case CircuitOpen:
		// already open, stay open
	}
}

// IsOpen returns true if the circuit is currently open (failing)
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == CircuitOpen
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// default circuit breaker instance for the sageox API (lazy initialized)
var (
	defaultCircuitOnce sync.Once
	defaultCircuit     *CircuitBreaker
)

// DefaultCircuit returns the default circuit breaker for sageox API calls
func DefaultCircuit() *CircuitBreaker {
	defaultCircuitOnce.Do(func() {
		defaultCircuit = NewCircuitBreaker()
	})
	return defaultCircuit
}
