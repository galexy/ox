package resilience

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThrottle_FirstCallImmediate(t *testing.T) {
	th := NewThrottle(WithMinInterval(100 * time.Millisecond))

	start := time.Now()
	th.Wait()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 10*time.Millisecond, "first call should be immediate")
}

func TestThrottle_EnforcesMinInterval(t *testing.T) {
	interval := 50 * time.Millisecond
	th := NewThrottle(WithMinInterval(interval))

	th.Wait() // first call

	start := time.Now()
	th.Wait() // should wait
	elapsed := time.Since(start)

	// should have waited close to the interval
	assert.GreaterOrEqual(t, elapsed, interval-5*time.Millisecond, "expected to wait ~%v, but only waited %v", interval, elapsed)
}

func TestThrottle_NoWaitAfterInterval(t *testing.T) {
	interval := 20 * time.Millisecond
	th := NewThrottle(WithMinInterval(interval))

	th.Wait()
	time.Sleep(interval + 10*time.Millisecond) // wait longer than interval

	start := time.Now()
	th.Wait()
	elapsed := time.Since(start)

	// should not wait since interval already passed
	assert.Less(t, elapsed, 10*time.Millisecond, "should not wait after interval passed")
}

func TestThrottle_DefaultInterval(t *testing.T) {
	th := NewThrottle()
	assert.Equal(t, 100*time.Millisecond, th.MinInterval())
}

func TestThrottle_DefaultThrottleIsSingleton(t *testing.T) {
	t1 := DefaultThrottle()
	t2 := DefaultThrottle()

	assert.Same(t, t1, t2, "expected DefaultThrottle to return same instance")
}
