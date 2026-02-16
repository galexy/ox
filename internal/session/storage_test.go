package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_Save(t *testing.T) {
	storage := NewMemoryStorage()

	meta := &StoreMeta{
		Version:   "1.0",
		AgentID:   "Oxa1b2",
		AgentType: "claude-code",
	}

	entries := []map[string]any{
		{"type": "user", "content": "hello"},
		{"type": "assistant", "content": "hi there"},
	}

	err := storage.Save("test.jsonl", "raw", meta, entries)
	require.NoError(t, err)
	assert.Equal(t, 1, storage.Count())
}

func TestMemoryStorage_Load(t *testing.T) {
	storage := NewMemoryStorage()

	meta := &StoreMeta{
		Version:   "1.0",
		AgentID:   "Oxa1b2",
		AgentType: "claude-code",
		Model:     "claude-sonnet-4",
	}

	entries := []map[string]any{
		{"type": "user", "content": "hello"},
		{"type": "assistant", "content": "hi there"},
	}

	err := storage.Save("test.jsonl", "raw", meta, entries)
	require.NoError(t, err)

	loaded, err := storage.Load("test.jsonl")
	require.NoError(t, err)

	assert.Equal(t, "Oxa1b2", loaded.Meta.AgentID)
	assert.Equal(t, "claude-sonnet-4", loaded.Meta.Model)
	assert.Len(t, loaded.Entries, 2)
}

func TestMemoryStorage_LoadNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	_, err := storage.Load("nonexistent.jsonl")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryStorage_List(t *testing.T) {
	storage := NewMemoryStorage()

	// save multiple sessions with slight delays
	for i := 0; i < 3; i++ {
		meta := &StoreMeta{Version: "1.0"}
		entries := []map[string]any{{"type": "user", "content": "test"}}

		filename := []string{"a.jsonl", "b.jsonl", "c.jsonl"}[i]
		sessionType := []string{"raw", "events", "raw"}[i]

		err := storage.Save(filename, sessionType, meta, entries)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // ensure different mod times
	}

	list, err := storage.List()
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// verify sorted by date descending (newest first)
	assert.Equal(t, "c.jsonl", list[0].Filename)
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()

	storage.Save("test.jsonl", "raw", &StoreMeta{}, nil)

	assert.True(t, storage.Exists("test.jsonl"))

	err := storage.Delete("test.jsonl")
	require.NoError(t, err)

	assert.False(t, storage.Exists("test.jsonl"))

	// delete non-existent should error
	err = storage.Delete("nonexistent.jsonl")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryStorage_Exists(t *testing.T) {
	storage := NewMemoryStorage()

	assert.False(t, storage.Exists("test.jsonl"))

	storage.Save("test.jsonl", "raw", &StoreMeta{}, nil)

	assert.True(t, storage.Exists("test.jsonl"))
}

func TestMemoryStorage_GetLatest(t *testing.T) {
	storage := NewMemoryStorage()

	// empty storage
	_, err := storage.GetLatest()
	assert.ErrorIs(t, err, ErrNoSessions)

	// add sessions
	storage.Save("old.jsonl", "raw", &StoreMeta{}, nil)
	time.Sleep(10 * time.Millisecond)
	storage.Save("new.jsonl", "raw", &StoreMeta{}, nil)

	latest, err := storage.GetLatest()
	require.NoError(t, err)
	assert.Equal(t, "new.jsonl", latest.Filename)
}

func TestMemoryStorage_Clear(t *testing.T) {
	storage := NewMemoryStorage()

	storage.Save("a.jsonl", "raw", &StoreMeta{}, nil)
	storage.Save("b.jsonl", "raw", &StoreMeta{}, nil)

	assert.Equal(t, 2, storage.Count())

	storage.Clear()

	assert.Equal(t, 0, storage.Count())
}

func TestMemoryStorage_Concurrent(t *testing.T) {
	storage := NewMemoryStorage()

	// concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			meta := &StoreMeta{AgentID: "concurrent"}
			entries := []map[string]any{{"n": n}}
			storage.Save("test.jsonl", "raw", meta, entries)
			done <- true
		}(i)
	}

	// wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// should have 1 session (last write wins)
	assert.Equal(t, 1, storage.Count())

	// concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			storage.Load("test.jsonl")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStorage_FileStorage_Interface(t *testing.T) {
	// verify Store implements Storage
	tempDir := t.TempDir()
	store, err := NewStore(tempDir)
	require.NoError(t, err)

	// type assertion should work
	var _ Storage = store
}

func TestStorage_Save_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStore(tempDir)
	require.NoError(t, err)

	meta := &StoreMeta{
		Version:   "1.0",
		AgentID:   "Oxtest",
		AgentType: "claude-code",
		Model:     "claude-opus-4",
	}

	entries := []map[string]any{
		{"type": "user", "content": "hello world"},
		{"type": "assistant", "content": "hi there!"},
	}

	err = store.Save("integration-test.jsonl", "raw", meta, entries)
	require.NoError(t, err)

	// load it back
	loaded, err := store.Load("integration-test.jsonl")
	require.NoError(t, err)

	assert.Equal(t, "Oxtest", loaded.Meta.AgentID)
	assert.Equal(t, "claude-opus-4", loaded.Meta.Model)
	assert.Len(t, loaded.Entries, 2)
}
