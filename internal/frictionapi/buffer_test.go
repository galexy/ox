package frictionapi

import (
	"sync"
	"testing"
)

func TestNewRingBuffer(t *testing.T) {
	tests := []struct {
		name             string
		capacity         int
		expectedCapacity int
	}{
		{
			name:             "positive capacity",
			capacity:         10,
			expectedCapacity: 10,
		},
		{
			name:             "capacity of one",
			capacity:         1,
			expectedCapacity: 1,
		},
		{
			name:             "zero capacity defaults to one",
			capacity:         0,
			expectedCapacity: 1,
		},
		{
			name:             "negative capacity defaults to one",
			capacity:         -5,
			expectedCapacity: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rb := NewRingBuffer(tc.capacity)
			if rb == nil {
				t.Fatal("NewRingBuffer returned nil")
			}
			if rb.capacity != tc.expectedCapacity {
				t.Errorf("capacity = %d, want %d", rb.capacity, tc.expectedCapacity)
			}
			if len(rb.events) != tc.expectedCapacity {
				t.Errorf("events slice length = %d, want %d", len(rb.events), tc.expectedCapacity)
			}
			if rb.count != 0 {
				t.Errorf("initial count = %d, want 0", rb.count)
			}
			if rb.head != 0 {
				t.Errorf("initial head = %d, want 0", rb.head)
			}
		})
	}
}

func TestRingBuffer_Add(t *testing.T) {
	tests := []struct {
		name          string
		capacity      int
		eventsToAdd   int
		expectedCount int
	}{
		{
			name:          "add single event",
			capacity:      5,
			eventsToAdd:   1,
			expectedCount: 1,
		},
		{
			name:          "add events up to capacity",
			capacity:      5,
			eventsToAdd:   5,
			expectedCount: 5,
		},
		{
			name:          "add events beyond capacity overwrites oldest",
			capacity:      3,
			eventsToAdd:   7,
			expectedCount: 3,
		},
		{
			name:          "capacity of one with multiple adds",
			capacity:      1,
			eventsToAdd:   5,
			expectedCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rb := NewRingBuffer(tc.capacity)

			for i := 0; i < tc.eventsToAdd; i++ {
				event := FrictionEvent{
					Kind:  string(FailureInvalidArg),
					Input: "test",
				}
				rb.Add(event)
			}

			if rb.Count() != tc.expectedCount {
				t.Errorf("Count() = %d, want %d", rb.Count(), tc.expectedCount)
			}
		})
	}
}

func TestRingBuffer_Drain(t *testing.T) {
	tests := []struct {
		name           string
		capacity       int
		eventsToAdd    []string // use Input field to track order
		expectedInputs []string // expected order after drain
	}{
		{
			name:           "drain empty buffer",
			capacity:       5,
			eventsToAdd:    []string{},
			expectedInputs: nil,
		},
		{
			name:           "drain single event",
			capacity:       5,
			eventsToAdd:    []string{"cmd1"},
			expectedInputs: []string{"cmd1"},
		},
		{
			name:           "drain multiple events in order",
			capacity:       5,
			eventsToAdd:    []string{"cmd1", "cmd2", "cmd3"},
			expectedInputs: []string{"cmd1", "cmd2", "cmd3"},
		},
		{
			name:           "drain at capacity",
			capacity:       3,
			eventsToAdd:    []string{"cmd1", "cmd2", "cmd3"},
			expectedInputs: []string{"cmd1", "cmd2", "cmd3"},
		},
		{
			name:           "drain after overwrite preserves chronological order",
			capacity:       3,
			eventsToAdd:    []string{"cmd1", "cmd2", "cmd3", "cmd4", "cmd5"},
			expectedInputs: []string{"cmd3", "cmd4", "cmd5"}, // oldest (cmd1, cmd2) overwritten
		},
		{
			name:           "drain with capacity one after multiple adds",
			capacity:       1,
			eventsToAdd:    []string{"cmd1", "cmd2", "cmd3"},
			expectedInputs: []string{"cmd3"}, // only last remains
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rb := NewRingBuffer(tc.capacity)

			for _, input := range tc.eventsToAdd {
				rb.Add(FrictionEvent{Input: input})
			}

			result := rb.Drain()

			// check nil vs empty slice
			if tc.expectedInputs == nil {
				if result != nil {
					t.Errorf("Drain() = %v, want nil", result)
				}
				return
			}

			if len(result) != len(tc.expectedInputs) {
				t.Fatalf("Drain() returned %d events, want %d", len(result), len(tc.expectedInputs))
			}

			for i, expected := range tc.expectedInputs {
				if result[i].Input != expected {
					t.Errorf("result[%d].Input = %q, want %q", i, result[i].Input, expected)
				}
			}

			// verify buffer is cleared after drain
			if rb.Count() != 0 {
				t.Errorf("Count() after Drain() = %d, want 0", rb.Count())
			}
			if rb.head != 0 {
				t.Errorf("head after Drain() = %d, want 0", rb.head)
			}
		})
	}
}

func TestRingBuffer_Drain_ClearsBuffer(t *testing.T) {
	rb := NewRingBuffer(5)

	rb.Add(FrictionEvent{Input: "cmd1"})
	rb.Add(FrictionEvent{Input: "cmd2"})

	// first drain should return events
	first := rb.Drain()
	if len(first) != 2 {
		t.Fatalf("first Drain() returned %d events, want 2", len(first))
	}

	// second drain should return nil (empty)
	second := rb.Drain()
	if second != nil {
		t.Errorf("second Drain() = %v, want nil", second)
	}
}

func TestRingBuffer_Count(t *testing.T) {
	rb := NewRingBuffer(5)

	// empty buffer
	if rb.Count() != 0 {
		t.Errorf("Count() on empty buffer = %d, want 0", rb.Count())
	}

	// add events
	rb.Add(FrictionEvent{Input: "cmd1"})
	if rb.Count() != 1 {
		t.Errorf("Count() after 1 add = %d, want 1", rb.Count())
	}

	rb.Add(FrictionEvent{Input: "cmd2"})
	rb.Add(FrictionEvent{Input: "cmd3"})
	if rb.Count() != 3 {
		t.Errorf("Count() after 3 adds = %d, want 3", rb.Count())
	}

	// drain and verify count resets
	rb.Drain()
	if rb.Count() != 0 {
		t.Errorf("Count() after Drain() = %d, want 0", rb.Count())
	}
}

func TestRingBuffer_ConcurrentAdd(t *testing.T) {
	rb := NewRingBuffer(100)

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				rb.Add(FrictionEvent{
					Kind:  string(FailureInvalidArg),
					Input: "concurrent",
				})
			}
		}(i)
	}

	wg.Wait()

	totalAdded := numGoroutines * eventsPerGoroutine
	expectedCount := rb.capacity // should cap at capacity
	if totalAdded < rb.capacity {
		expectedCount = totalAdded
	}

	if rb.Count() != expectedCount {
		t.Errorf("Count() after concurrent adds = %d, want %d", rb.Count(), expectedCount)
	}

	// verify drain works after concurrent adds
	events := rb.Drain()
	if len(events) != expectedCount {
		t.Errorf("Drain() returned %d events, want %d", len(events), expectedCount)
	}
}

func TestRingBuffer_HeadWrapAround(t *testing.T) {
	// test that head correctly wraps around the buffer
	rb := NewRingBuffer(3)

	// add 7 events: head should wrap around twice
	for i := 0; i < 7; i++ {
		rb.Add(FrictionEvent{Input: string(rune('a' + i))})
	}

	// after 7 adds with capacity 3:
	// positions: [e, f, g] (indices 0, 1, 2)
	// head should be at 7 % 3 = 1
	if rb.head != 1 {
		t.Errorf("head = %d, want 1 after 7 adds in capacity 3 buffer", rb.head)
	}

	// drain should return in chronological order: e, f, g
	events := rb.Drain()
	expected := []string{"e", "f", "g"}
	for i, exp := range expected {
		if events[i].Input != exp {
			t.Errorf("events[%d].Input = %q, want %q", i, events[i].Input, exp)
		}
	}
}
