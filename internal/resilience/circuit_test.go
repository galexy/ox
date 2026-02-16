package resilience

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	cb := NewCircuitBreaker()
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(3))

	// record 3 failures
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State(), "expected CircuitClosed after 2 failures")

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State(), "expected CircuitOpen after 3 failures")
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(3))

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // should reset

	assert.Equal(t, 0, cb.Failures(), "expected 0 failures after success")

	// now need 3 more failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_ShouldAttemptAlwaysTrueWithRetryChance1(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(1), WithRetryChance(1.0))

	// closed - should attempt
	assert.True(t, cb.ShouldAttempt(), "expected ShouldAttempt=true when closed")

	// open it
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// with retryChance=1.0, still should attempt
	assert.True(t, cb.ShouldAttempt(), "expected ShouldAttempt=true when open with retryChance=1.0")
}

func TestCircuitBreaker_SuccessInOpenMovesToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(1))

	cb.RecordFailure() // opens circuit
	assert.Equal(t, CircuitOpen, cb.State())

	cb.RecordSuccess() // should move to half-open
	assert.Equal(t, CircuitHalfOpen, cb.State())
}

func TestCircuitBreaker_SuccessInHalfOpenCloses(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(1))

	cb.RecordFailure() // opens
	cb.RecordSuccess() // half-open
	cb.RecordSuccess() // should close

	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_FailureInHalfOpenReopens(t *testing.T) {
	cb := NewCircuitBreaker(WithThreshold(1))

	cb.RecordFailure() // opens
	cb.RecordSuccess() // half-open

	assert.Equal(t, CircuitHalfOpen, cb.State())

	cb.RecordFailure() // should reopen
	assert.Equal(t, CircuitOpen, cb.State())
}

func TestCircuitBreaker_DefaultCircuitIsSingleton(t *testing.T) {
	c1 := DefaultCircuit()
	c2 := DefaultCircuit()

	assert.Same(t, c1, c2, "expected DefaultCircuit to return same instance")
}
