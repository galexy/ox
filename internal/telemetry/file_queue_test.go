package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestProject creates a valid SageOx project structure for testing
func setupTestProject(t *testing.T, projectRoot string) {
	t.Helper()
	sageoxDir := filepath.Join(projectRoot, ".sageox")
	err := os.MkdirAll(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox dir")
	// create README.md marker file
	readmePath := filepath.Join(sageoxDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# SageOx"), 0644)
	require.NoError(t, err, "failed to create README.md")
}

func TestFileQueue_Append(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	event := Event{
		Type:    EventGuidanceFetch,
		Path:    "infra/aws/s3",
		Success: true,
	}

	err := q.Append(event)
	require.NoError(t, err, "Append failed")

	// verify file was created
	_, err = os.Stat(q.Path())
	assert.False(t, os.IsNotExist(err), "queue file was not created")
}

func TestFileQueue_ReadAll(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	// append multiple events
	events := []Event{
		{Type: EventSessionStart, AgentID: "abc123", Success: true},
		{Type: EventGuidanceFetch, Path: "infra/aws", Success: true},
		{Type: EventAttributionShown, Path: "security/iam", Success: true},
	}

	for _, e := range events {
		err := q.Append(e)
		require.NoError(t, err, "Append failed")
	}

	// read all events
	read, err := q.ReadAll()
	require.NoError(t, err, "ReadAll failed")

	assert.Len(t, read, len(events))

	// verify event types match
	for i, e := range read {
		assert.Equal(t, events[i].Type, e.Type, "event %d type mismatch", i)
	}
}

func TestFileQueue_Truncate(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	// append an event
	_ = q.Append(Event{Type: EventGuidanceFetch, Success: true})

	// verify event is there
	events, _ := q.ReadAll()
	require.Len(t, events, 1, "expected 1 event before truncate")

	// truncate
	err := q.Truncate()
	require.NoError(t, err, "Truncate failed")

	// verify empty
	events, _ = q.ReadAll()
	assert.Empty(t, events, "expected 0 events after truncate")
}

func TestFileQueue_ShouldFlush_CountThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	// add events up to threshold
	for i := 0; i < flushThresholdCount-1; i++ {
		_ = q.Append(Event{Type: EventGuidanceFetch, Success: true})
	}

	assert.False(t, q.ShouldFlush(), "ShouldFlush returned true before threshold")

	// add one more to hit threshold
	_ = q.Append(Event{Type: EventGuidanceFetch, Success: true})

	assert.True(t, q.ShouldFlush(), "ShouldFlush returned false at threshold")
}

func TestFileQueue_ShouldFlush_AgeThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	// add an old event
	oldEvent := Event{
		Type:      EventGuidanceFetch,
		Timestamp: time.Now().Add(-2 * time.Hour), // 2 hours old
		Success:   true,
	}
	_ = q.Append(oldEvent)

	assert.True(t, q.ShouldFlush(), "ShouldFlush returned false for old event")
}

func TestFileQueue_ShouldFlush_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	q := NewFileQueue(tmpDir)

	assert.False(t, q.ShouldFlush(), "ShouldFlush returned true for empty queue")
}

func TestFileQueue_NilSafe(t *testing.T) {
	var q *FileQueue

	// all operations should be safe on nil
	err := q.Append(Event{})
	assert.NoError(t, err, "Append on nil returned error")

	events, err := q.ReadAll()
	assert.NoError(t, err, "ReadAll on nil returned error")
	assert.Nil(t, events, "ReadAll on nil should return nil events")

	err = q.Truncate()
	assert.NoError(t, err, "Truncate on nil returned error")

	assert.False(t, q.ShouldFlush(), "ShouldFlush on nil returned true")
	assert.Equal(t, 0, q.Count(), "Count on nil should be 0")
	assert.Empty(t, q.Path(), "Path on nil should be empty")
}

func TestFileQueue_Count(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestProject(t, tmpDir)
	q := NewFileQueue(tmpDir)

	assert.Equal(t, 0, q.Count(), "Count on empty queue should be 0")

	_ = q.Append(Event{Type: EventGuidanceFetch})
	_ = q.Append(Event{Type: EventSessionStart})

	assert.Equal(t, 2, q.Count(), "Count after 2 appends should be 2")
}

func TestFileQueue_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "deep", "nested", "project")
	setupTestProject(t, deepPath) // this creates .sageox/README.md
	q := NewFileQueue(deepPath)

	err := q.Append(Event{Type: EventGuidanceFetch, Success: true})
	require.NoError(t, err, "Append failed")

	// verify cache directory structure was created
	expectedDir := filepath.Join(deepPath, ".sageox", "cache")
	_, err = os.Stat(expectedDir)
	assert.False(t, os.IsNotExist(err), "directory %s was not created", expectedDir)
}

func TestNewFileQueue_EmptyProjectRoot(t *testing.T) {
	q := NewFileQueue("")
	assert.Nil(t, q, "NewFileQueue with empty projectRoot should return nil")
}
