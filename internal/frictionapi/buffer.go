package frictionapi

import (
	"sync"
)

// RingBuffer is a thread-safe circular buffer for FrictionEvents.
// When full, new events overwrite the oldest.
type RingBuffer struct {
	mu       sync.Mutex
	events   []FrictionEvent
	capacity int
	head     int // next write position
	count    int // current number of stored events
}

// NewRingBuffer creates a RingBuffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer{
		events:   make([]FrictionEvent, capacity),
		capacity: capacity,
	}
}

// Add inserts an event into the buffer, overwriting the oldest if full.
func (rb *RingBuffer) Add(event FrictionEvent) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.events[rb.head] = event
	rb.head = (rb.head + 1) % rb.capacity

	if rb.count < rb.capacity {
		rb.count++
	}
}

// Drain returns all events in chronological order (oldest first) and clears the buffer.
func (rb *RingBuffer) Drain() []FrictionEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]FrictionEvent, rb.count)

	// calculate start position (oldest event)
	start := 0
	if rb.count == rb.capacity {
		// buffer is full, oldest is at head position
		start = rb.head
	}

	// copy events in chronological order
	for i := 0; i < rb.count; i++ {
		idx := (start + i) % rb.capacity
		result[i] = rb.events[idx]
	}

	// clear the buffer
	rb.head = 0
	rb.count = 0

	return result
}

// Count returns the current number of stored events.
func (rb *RingBuffer) Count() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}
