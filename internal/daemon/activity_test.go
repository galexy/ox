package daemon

import (
	"sync"
	"testing"
	"time"
)

func TestActivityTracker_Record(t *testing.T) {
	tracker := NewActivityTracker(5)

	// record some events
	tracker.Record("repo-a")
	tracker.Record("repo-a")
	tracker.Record("repo-b")

	if tracker.Count("repo-a") != 2 {
		t.Errorf("expected 2 events for repo-a, got %d", tracker.Count("repo-a"))
	}
	if tracker.Count("repo-b") != 1 {
		t.Errorf("expected 1 event for repo-b, got %d", tracker.Count("repo-b"))
	}
	if tracker.Count("repo-c") != 0 {
		t.Errorf("expected 0 events for repo-c, got %d", tracker.Count("repo-c"))
	}
}

func TestActivityTracker_RingBuffer(t *testing.T) {
	tracker := NewActivityTracker(3) // small capacity

	now := time.Now()
	tracker.RecordAt("key", now.Add(-4*time.Second))
	tracker.RecordAt("key", now.Add(-3*time.Second))
	tracker.RecordAt("key", now.Add(-2*time.Second))
	tracker.RecordAt("key", now.Add(-1*time.Second))
	tracker.RecordAt("key", now)

	// should only have last 3 entries
	if tracker.Count("key") != 3 {
		t.Errorf("expected 3 events (capped), got %d", tracker.Count("key"))
	}

	timestamps := tracker.Get("key")
	if len(timestamps) != 3 {
		t.Fatalf("expected 3 timestamps, got %d", len(timestamps))
	}

	// oldest should be -2s, newest should be now
	if !timestamps[0].Equal(now.Add(-2 * time.Second)) {
		t.Errorf("expected oldest to be -2s, got %v", timestamps[0].Sub(now))
	}
	if !timestamps[2].Equal(now) {
		t.Errorf("expected newest to be now, got %v", timestamps[2].Sub(now))
	}
}

func TestActivityTracker_Last(t *testing.T) {
	tracker := NewActivityTracker(5)

	// empty key
	if !tracker.Last("empty").IsZero() {
		t.Error("expected zero time for empty key")
	}

	now := time.Now()
	tracker.RecordAt("key", now.Add(-1*time.Second))
	tracker.RecordAt("key", now)

	last := tracker.Last("key")
	if !last.Equal(now) {
		t.Errorf("expected last to be now, got %v", last)
	}
}

func TestActivityTracker_Keys(t *testing.T) {
	tracker := NewActivityTracker(5)

	tracker.Record("a")
	tracker.Record("b")
	tracker.Record("c")

	keys := tracker.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// check all keys present (order not guaranteed)
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !keyMap[expected] {
			t.Errorf("expected key %s to be present", expected)
		}
	}
}

func TestActivityTracker_Clear(t *testing.T) {
	tracker := NewActivityTracker(5)

	tracker.Record("keep")
	tracker.Record("remove")

	tracker.Clear("remove")

	if tracker.Count("keep") != 1 {
		t.Error("expected keep to still have 1 event")
	}
	if tracker.Count("remove") != 0 {
		t.Error("expected remove to have 0 events after clear")
	}
}

func TestActivityTracker_Reset(t *testing.T) {
	tracker := NewActivityTracker(5)

	tracker.Record("a")
	tracker.Record("b")

	tracker.Reset()

	if len(tracker.Keys()) != 0 {
		t.Error("expected no keys after reset")
	}
}

func TestRingBuffer_SliceOrder(t *testing.T) {
	ring := newRingBuffer(4)

	// add entries 1-6 (should wrap around)
	for i := 1; i <= 6; i++ {
		ring.Add(time.Unix(int64(i), 0))
	}

	// should contain 3,4,5,6 in order
	slice := ring.Slice()
	if len(slice) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(slice))
	}

	expected := []int64{3, 4, 5, 6}
	for i, ts := range slice {
		if ts.Unix() != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], ts.Unix())
		}
	}
}

// Edge case tests

func TestRingBuffer_Empty(t *testing.T) {
	ring := newRingBuffer(5)

	if ring.Count() != 0 {
		t.Error("expected count 0 for empty ring")
	}

	slice := ring.Slice()
	if slice != nil {
		t.Errorf("expected nil slice for empty ring, got %v", slice)
	}
}

func TestRingBuffer_SingleEntry(t *testing.T) {
	ring := newRingBuffer(5)
	ts := time.Now()
	ring.Add(ts)

	if ring.Count() != 1 {
		t.Errorf("expected count 1, got %d", ring.Count())
	}

	slice := ring.Slice()
	if len(slice) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(slice))
	}
	if !slice[0].Equal(ts) {
		t.Error("timestamp mismatch")
	}
}

func TestRingBuffer_ExactlyAtCapacity(t *testing.T) {
	ring := newRingBuffer(3)

	for i := 1; i <= 3; i++ {
		ring.Add(time.Unix(int64(i), 0))
	}

	if ring.Count() != 3 {
		t.Errorf("expected count 3, got %d", ring.Count())
	}

	slice := ring.Slice()
	expected := []int64{1, 2, 3}
	for i, ts := range slice {
		if ts.Unix() != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], ts.Unix())
		}
	}
}

func TestRingBuffer_MultipleWraps(t *testing.T) {
	ring := newRingBuffer(3)

	// add 10 entries, buffer wraps multiple times
	for i := 1; i <= 10; i++ {
		ring.Add(time.Unix(int64(i), 0))
	}

	if ring.Count() != 3 {
		t.Errorf("expected count 3, got %d", ring.Count())
	}

	// should have 8, 9, 10
	slice := ring.Slice()
	expected := []int64{8, 9, 10}
	for i, ts := range slice {
		if ts.Unix() != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], ts.Unix())
		}
	}
}

func TestActivityTracker_DefaultCapacity(t *testing.T) {
	// zero capacity should use default
	tracker := NewActivityTracker(0)
	if tracker.capacity != 50 {
		t.Errorf("expected default capacity 50, got %d", tracker.capacity)
	}

	// negative capacity should use default
	tracker = NewActivityTracker(-5)
	if tracker.capacity != 50 {
		t.Errorf("expected default capacity 50, got %d", tracker.capacity)
	}
}

func TestActivityTracker_GetNonExistent(t *testing.T) {
	tracker := NewActivityTracker(5)

	result := tracker.Get("nonexistent")
	if result != nil {
		t.Errorf("expected nil for nonexistent key, got %v", result)
	}
}

func TestActivityTracker_LastAfterWrap(t *testing.T) {
	tracker := NewActivityTracker(3)

	// add 5 entries, forcing wrap
	for i := 1; i <= 5; i++ {
		tracker.RecordAt("key", time.Unix(int64(i), 0))
	}

	last := tracker.Last("key")
	if last.Unix() != 5 {
		t.Errorf("expected last to be 5, got %d", last.Unix())
	}
}

func TestActivityTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewActivityTracker(100)

	var wg sync.WaitGroup
	numGoroutines := 10
	recordsPerGoroutine := 100

	// concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "key"
			for j := 0; j < recordsPerGoroutine; j++ {
				tracker.Record(key)
			}
		}(i)
	}

	// concurrent reads while writing
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < recordsPerGoroutine; j++ {
				_ = tracker.Get("key")
				_ = tracker.Last("key")
				_ = tracker.Count("key")
				_ = tracker.Keys()
			}
		}()
	}

	wg.Wait()

	// should have 100 entries (capped)
	if tracker.Count("key") != 100 {
		t.Errorf("expected 100 entries, got %d", tracker.Count("key"))
	}
}

func TestActivityTracker_ManyKeys(t *testing.T) {
	tracker := NewActivityTracker(5)

	// add 100 different keys
	for i := 0; i < 100; i++ {
		tracker.Record("key-" + string(rune('a'+i%26)) + string(rune('0'+i/26)))
	}

	keys := tracker.Keys()
	if len(keys) != 100 {
		t.Errorf("expected 100 keys, got %d", len(keys))
	}
}

func TestActivityTracker_ClearNonExistent(t *testing.T) {
	tracker := NewActivityTracker(5)
	tracker.Record("exists")

	// clearing non-existent key should be a no-op
	tracker.Clear("nonexistent")

	if tracker.Count("exists") != 1 {
		t.Error("clearing nonexistent key affected existing key")
	}
}

func TestActivityTracker_RecordAfterClear(t *testing.T) {
	tracker := NewActivityTracker(5)

	tracker.Record("key")
	tracker.Clear("key")
	tracker.Record("key")

	if tracker.Count("key") != 1 {
		t.Errorf("expected 1 after clear and re-record, got %d", tracker.Count("key"))
	}
}

func TestRingBuffer_CapacityOne(t *testing.T) {
	ring := newRingBuffer(1)

	ring.Add(time.Unix(1, 0))
	ring.Add(time.Unix(2, 0))
	ring.Add(time.Unix(3, 0))

	if ring.Count() != 1 {
		t.Errorf("expected count 1, got %d", ring.Count())
	}

	slice := ring.Slice()
	if len(slice) != 1 || slice[0].Unix() != 3 {
		t.Error("expected only most recent entry")
	}
}

func TestActivityTracker_MaxKeysEviction(t *testing.T) {
	// create tracker with max 5 keys
	tracker := NewActivityTrackerWithMaxKeys(10, 5)

	// add first 5 keys
	for i := 0; i < 5; i++ {
		tracker.RecordAt("key-"+string(rune('a'+i)), time.Unix(int64(i+1), 0))
	}

	if len(tracker.Keys()) != 5 {
		t.Errorf("expected 5 keys, got %d", len(tracker.Keys()))
	}

	// add 6th key - should evict oldest
	tracker.RecordAt("key-f", time.Unix(10, 0))

	keys := tracker.Keys()
	if len(keys) != 5 {
		t.Errorf("expected 5 keys after eviction, got %d", len(keys))
	}

	// key-a (oldest at unix 1) should have been evicted
	if tracker.Count("key-a") != 0 {
		t.Error("expected key-a to be evicted")
	}
	if tracker.Count("key-f") != 1 {
		t.Error("expected key-f to be present")
	}
}

func TestActivityTracker_MaxKeysEvictsOldest(t *testing.T) {
	tracker := NewActivityTrackerWithMaxKeys(10, 3)

	// add keys with different timestamps (b is oldest)
	tracker.RecordAt("key-a", time.Unix(100, 0))
	tracker.RecordAt("key-b", time.Unix(50, 0)) // oldest
	tracker.RecordAt("key-c", time.Unix(200, 0))

	// add 4th key - should evict key-b (oldest)
	tracker.RecordAt("key-d", time.Unix(300, 0))

	if tracker.Count("key-b") != 0 {
		t.Error("expected key-b (oldest) to be evicted")
	}
	if tracker.Count("key-a") != 1 {
		t.Error("expected key-a to remain")
	}
	if tracker.Count("key-c") != 1 {
		t.Error("expected key-c to remain")
	}
	if tracker.Count("key-d") != 1 {
		t.Error("expected key-d to be added")
	}
}

func TestActivityTracker_UnlimitedKeys(t *testing.T) {
	// maxKeys = 0 means unlimited
	tracker := NewActivityTrackerWithMaxKeys(5, 0)

	// add many keys - none should be evicted
	for i := 0; i < 100; i++ {
		tracker.Record("key-" + string(rune('a'+i%26)) + string(rune('0'+i/26)))
	}

	if len(tracker.Keys()) != 100 {
		t.Errorf("expected 100 keys with unlimited, got %d", len(tracker.Keys()))
	}
}

func TestActivityTracker_ExistingKeyNoEviction(t *testing.T) {
	tracker := NewActivityTrackerWithMaxKeys(10, 3)

	// add 3 keys
	tracker.Record("key-a")
	tracker.Record("key-b")
	tracker.Record("key-c")

	// recording to existing key should not trigger eviction
	for i := 0; i < 10; i++ {
		tracker.Record("key-a")
	}

	if len(tracker.Keys()) != 3 {
		t.Errorf("expected 3 keys, got %d", len(tracker.Keys()))
	}
	if tracker.Count("key-a") != 10 {
		t.Errorf("expected key-a to have 10 entries, got %d", tracker.Count("key-a"))
	}
}

func TestActivityTracker_ConcurrentEviction(t *testing.T) {
	tracker := NewActivityTrackerWithMaxKeys(5, 10)

	var wg sync.WaitGroup

	// 20 goroutines trying to add different keys
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := "key-" + string(rune('a'+id)) + string(rune('0'+j%10))
				tracker.Record(key)
			}
		}(i)
	}

	wg.Wait()

	// should not exceed maxKeys
	keys := tracker.Keys()
	if len(keys) > 10 {
		t.Errorf("expected at most 10 keys, got %d", len(keys))
	}
}

func TestNewActivityTrackerWithMaxKeys_NegativeMaxKeys(t *testing.T) {
	// negative maxKeys should be treated as 0 (unlimited)
	tracker := NewActivityTrackerWithMaxKeys(5, -10)
	if tracker.maxKeys != 0 {
		t.Errorf("expected maxKeys 0 for negative input, got %d", tracker.maxKeys)
	}
}
