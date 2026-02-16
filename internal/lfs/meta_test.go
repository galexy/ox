package lfs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndReadSessionMeta(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "2026-01-06T14-32-ryan-Ox7f3a")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	meta := &SessionMeta{
		Version:     "1.0",
		SessionName: "2026-01-06T14-32-ryan-Ox7f3a",
		Username:    "ryan@sageox.ai",
		AgentID:     "Ox7f3a",
		AgentType:   "claude-code",
		Model:       "claude-sonnet-4",
		CreatedAt:   time.Date(2026, 1, 6, 14, 32, 0, 0, time.UTC),
		EntryCount:  42,
		Summary:     "Implemented LFS pipeline",
		Files: map[string]FileRef{
			"raw.jsonl":    {OID: "sha256:abc123", Size: 1024},
			"events.jsonl": {OID: "sha256:def456", Size: 512},
			"summary.md":   {OID: "sha256:ghi789", Size: 256},
			"session.md":   {OID: "sha256:jkl012", Size: 2048},
			"session.html": {OID: "sha256:mno345", Size: 4096},
		},
	}

	err := WriteSessionMeta(sessionDir, meta)
	require.NoError(t, err)

	// verify file exists
	_, err = os.Stat(filepath.Join(sessionDir, "meta.json"))
	require.NoError(t, err)

	// read back
	got, err := ReadSessionMeta(sessionDir)
	require.NoError(t, err)
	assert.Equal(t, meta.Version, got.Version)
	assert.Equal(t, meta.SessionName, got.SessionName)
	assert.Equal(t, meta.Username, got.Username)
	assert.Equal(t, meta.AgentID, got.AgentID)
	assert.Equal(t, meta.AgentType, got.AgentType)
	assert.Equal(t, meta.Model, got.Model)
	assert.Equal(t, meta.EntryCount, got.EntryCount)
	assert.Equal(t, meta.Summary, got.Summary)
	assert.Len(t, got.Files, 5)
	assert.Equal(t, "sha256:abc123", got.Files["raw.jsonl"].OID)
	assert.Equal(t, int64(1024), got.Files["raw.jsonl"].Size)
}

func TestWriteSessionMeta_Nil(t *testing.T) {
	err := WriteSessionMeta(t.TempDir(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil session meta")
}

func TestReadSessionMeta_NotFound(t *testing.T) {
	_, err := ReadSessionMeta(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meta.json not found")
}

func TestCheckHydrationStatus_Hydrated(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	// create content files
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "raw.jsonl"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte("data"), 0644))

	meta := &SessionMeta{
		Files: map[string]FileRef{
			"raw.jsonl":    {OID: "sha256:abc", Size: 4},
			"events.jsonl": {OID: "sha256:def", Size: 4},
		},
	}

	status := CheckHydrationStatus(sessionDir, meta)
	assert.Equal(t, HydrationStatusHydrated, status)
}

func TestCheckHydrationStatus_Dehydrated(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	meta := &SessionMeta{
		Files: map[string]FileRef{
			"raw.jsonl":    {OID: "sha256:abc", Size: 100},
			"events.jsonl": {OID: "sha256:def", Size: 200},
		},
	}

	status := CheckHydrationStatus(sessionDir, meta)
	assert.Equal(t, HydrationStatusDehydrated, status)
}

func TestCheckHydrationStatus_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "session")
	require.NoError(t, os.MkdirAll(sessionDir, 0755))

	// only create one of two files
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "raw.jsonl"), []byte("data"), 0644))

	meta := &SessionMeta{
		Files: map[string]FileRef{
			"raw.jsonl":    {OID: "sha256:abc", Size: 4},
			"events.jsonl": {OID: "sha256:def", Size: 200},
		},
	}

	status := CheckHydrationStatus(sessionDir, meta)
	assert.Equal(t, HydrationStatusPartial, status)
}

func TestCheckHydrationStatus_NilMeta(t *testing.T) {
	status := CheckHydrationStatus(t.TempDir(), nil)
	assert.Equal(t, HydrationStatusDehydrated, status)
}

func TestCheckHydrationStatus_EmptyFiles(t *testing.T) {
	meta := &SessionMeta{
		Files: map[string]FileRef{},
	}
	status := CheckHydrationStatus(t.TempDir(), meta)
	assert.Equal(t, HydrationStatusDehydrated, status)
}

func TestNewFileRef(t *testing.T) {
	content := []byte("test content")
	ref := NewFileRef(content)
	assert.Equal(t, int64(len(content)), ref.Size)
	assert.True(t, len(ref.OID) > 7)
	assert.Equal(t, "sha256:", ref.OID[:7])
}

func TestFileRef_BareOID(t *testing.T) {
	tests := []struct {
		oid      string
		expected string
	}{
		{"sha256:abc123", "abc123"},
		{"abc123", "abc123"},
		{"sha256:", "sha256:"},
		{"short", "short"},
	}

	for _, tt := range tests {
		ref := FileRef{OID: tt.oid}
		assert.Equal(t, tt.expected, ref.BareOID(), "BareOID for %q", tt.oid)
	}
}
