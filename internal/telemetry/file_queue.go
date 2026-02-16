package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileQueue provides disk-based event queueing for telemetry.
//
// BEST EFFORT: Telemetry is intentionally "fire and forget". Events may be lost due to:
//   - CLI crash before flush
//   - Network unavailability during POST
//   - File truncation on overflow (>1MB)
//   - Disk write failures (silently ignored)
//
// This is acceptable - telemetry is for aggregate analytics, not critical data.
// We optimize for CLI responsiveness over delivery guarantees.
//
// Events are appended to a JSONL file and POSTed lazily on subsequent CLI invocations.
// This keeps each CLI command fast (no blocking network calls) while ensuring events
// are eventually transmitted when conditions allow.
//
// FUTURE: Consider moving telemetry transmission to a separate daemon process.
// A daemon would provide more reliable delivery without impacting CLI responsiveness,
// and could handle retries, backoff, and batching more intelligently. For now, the
// lazy flush approach is simpler and sufficient for our needs.
type FileQueue struct {
	path string
	mu   sync.Mutex
}

const (
	// telemetryFileName is the name of the telemetry queue file
	telemetryFileName = "telemetry.jsonl"

	// flushThresholdCount triggers a flush when this many events are queued
	flushThresholdCount = 10

	// flushThresholdAge triggers a flush when oldest event is older than this
	flushThresholdAge = 1 * time.Hour

	// maxFileSize limits the telemetry file to prevent unbounded growth (1MB)
	maxFileSize = 1 * 1024 * 1024
)

// NewFileQueue creates a new file-based event queue.
// The queue file is stored at .sageox/cache/telemetry.jsonl in the project root.
// Returns nil if projectRoot is empty (telemetry disabled without project context).
func NewFileQueue(projectRoot string) *FileQueue {
	if projectRoot == "" {
		return nil
	}

	cacheDir := filepath.Join(projectRoot, ".sageox", "cache")
	return &FileQueue{
		path: filepath.Join(cacheDir, telemetryFileName),
	}
}

// Append adds an event to the queue file.
// This is a fast, append-only operation that does not block on network I/O.
// Returns nil on success; errors are non-fatal (telemetry should never break the CLI).
func (q *FileQueue) Append(event Event) error {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// only write telemetry if SageOx is properly installed
	// check for .sageox/README.md or .sageox/.repo_* marker files
	sageoxDir := filepath.Dir(filepath.Dir(q.path)) // .sageox/cache -> .sageox
	if !isSageoxInstalled(sageoxDir) {
		return nil // silently skip - SageOx not installed or was uninstalled
	}

	// ensure directory exists
	dir := filepath.Dir(q.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// check file size to prevent unbounded growth
	if info, err := os.Stat(q.path); err == nil && info.Size() > maxFileSize {
		// truncate old events - better to lose old data than grow forever
		_ = os.Truncate(q.path, 0)
	}

	// set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// append to file
	f, err := os.OpenFile(q.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// ReadAll reads all queued events from the file.
// Returns an empty slice if the file doesn't exist or is empty.
func (q *FileQueue) ReadAll() ([]Event, error) {
	if q == nil {
		return nil, nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	f, err := os.Open(q.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			// skip malformed lines
			continue
		}
		events = append(events, event)
	}

	return events, scanner.Err()
}

// Truncate removes all events from the queue file.
// Called after successful POST to clear transmitted events.
func (q *FileQueue) Truncate() error {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	return os.Truncate(q.path, 0)
}

// ShouldFlush returns true if the queue should be flushed.
// Triggers when: event count > threshold OR oldest event > age threshold.
func (q *FileQueue) ShouldFlush() bool {
	if q == nil {
		return false
	}

	events, err := q.ReadAll()
	if err != nil || len(events) == 0 {
		return false
	}

	// check count threshold
	if len(events) >= flushThresholdCount {
		return true
	}

	// check age threshold - oldest event is first in the list
	if len(events) > 0 && !events[0].Timestamp.IsZero() {
		age := time.Since(events[0].Timestamp)
		if age >= flushThresholdAge {
			return true
		}
	}

	return false
}

// Count returns the number of queued events.
func (q *FileQueue) Count() int {
	if q == nil {
		return 0
	}

	events, _ := q.ReadAll()
	return len(events)
}

// Path returns the path to the queue file.
func (q *FileQueue) Path() string {
	if q == nil {
		return ""
	}
	return q.path
}

// isSageoxInstalled checks if SageOx is properly installed in the given .sageox directory.
// Returns true if README.md or any .repo_* marker file exists.
func isSageoxInstalled(sageoxDir string) bool {
	// check for README.md
	if _, err := os.Stat(filepath.Join(sageoxDir, "README.md")); err == nil {
		return true
	}

	// check for .repo_* marker files
	entries, err := os.ReadDir(sageoxDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".repo_") {
			return true
		}
	}

	return false
}
