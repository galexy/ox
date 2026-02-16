package signature

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv configures environment for isolated testing
func setupTestEnv(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	// enable XDG mode and set temp directories for test isolation
	t.Setenv("OX_XDG_ENABLE", "1")
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	return tempDir
}

func TestGetMachineID_Creates(t *testing.T) {
	setupTestEnv(t)

	// first call should create machine ID
	machineID1, err := GetMachineID()
	require.NoError(t, err, "GetMachineID() error")

	assert.NotEmpty(t, machineID1, "GetMachineID() returned empty string")

	// verify machine ID file was created
	idPath, err := getMachineIDPath()
	require.NoError(t, err, "getMachineIDPath() error")
	_, err = os.Stat(idPath)
	assert.False(t, os.IsNotExist(err), "Machine ID file was not created")

	// second call should return same ID
	machineID2, err := GetMachineID()
	require.NoError(t, err, "GetMachineID() second call error")

	assert.Equal(t, machineID1, machineID2, "GetMachineID() returned different IDs")
}

func TestLoadCache_Empty(t *testing.T) {
	setupTestEnv(t)

	// load cache when no cache file exists
	cache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	require.NotNil(t, cache, "LoadCache() returned nil cache")
	assert.Empty(t, cache.Entries, "LoadCache() returned non-empty entries")
}

func TestSaveAndLoadCache(t *testing.T) {
	setupTestEnv(t)

	// create cache with entries
	originalCache := &VerificationCache{
		Entries: []CacheEntry{
			{
				FilePath:    "/path/to/file1.bin",
				ContentHash: "hash1",
				Verified:    true,
				Timestamp:   time.Now().Truncate(time.Second), // truncate to avoid nanosecond precision issues
			},
			{
				FilePath:    "/path/to/file2.bin",
				ContentHash: "hash2",
				Verified:    false,
				Timestamp:   time.Now().Truncate(time.Second),
			},
		},
	}

	// save cache
	require.NoError(t, SaveCache(originalCache), "SaveCache() error")

	// verify cache file was created
	cachePath, err := GetCachePath()
	require.NoError(t, err, "GetCachePath() error")
	_, err = os.Stat(cachePath)
	assert.False(t, os.IsNotExist(err), "Cache file was not created")

	// load cache
	loadedCache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	// verify entries match
	require.Len(t, loadedCache.Entries, len(originalCache.Entries))

	for i, entry := range loadedCache.Entries {
		original := originalCache.Entries[i]
		assert.Equal(t, original.FilePath, entry.FilePath, "Entry %d FilePath mismatch", i)
		assert.Equal(t, original.ContentHash, entry.ContentHash, "Entry %d ContentHash mismatch", i)
		assert.Equal(t, original.Verified, entry.Verified, "Entry %d Verified mismatch", i)
		assert.True(t, entry.Timestamp.Equal(original.Timestamp), "Entry %d Timestamp mismatch", i)
	}

	// verify HMAC is valid
	assert.NotEmpty(t, loadedCache.HMAC, "LoadCache() returned empty HMAC")

	machineID, err := GetMachineID()
	require.NoError(t, err, "GetMachineID() error")

	assert.True(t, verifyHMAC(loadedCache.Entries, loadedCache.HMAC, machineID), "LoadCache() returned cache with invalid HMAC")
}

func TestLoadCache_TamperedHMAC(t *testing.T) {
	setupTestEnv(t)

	// get machine ID to ensure it's created
	_, err := GetMachineID()
	require.NoError(t, err, "GetMachineID() error")

	// create cache file with invalid HMAC
	cachePath, err := GetCachePath()
	require.NoError(t, err, "GetCachePath() error")

	// ensure config directory exists
	configDir := filepath.Dir(cachePath)
	require.NoError(t, os.MkdirAll(configDir, 0700), "MkdirAll() error")

	tamperedCache := VerificationCache{
		Entries: []CacheEntry{
			{
				FilePath:    "/path/to/file.bin",
				ContentHash: "hash",
				Verified:    true,
				Timestamp:   time.Now(),
			},
		},
		HMAC: "0000000000000000000000000000000000000000000000000000000000000000", // invalid HMAC
	}

	data, err := json.MarshalIndent(tamperedCache, "", "  ")
	require.NoError(t, err, "json.Marshal() error")

	require.NoError(t, os.WriteFile(cachePath, data, 0600), "WriteFile() error")

	// load cache - should return empty cache due to invalid HMAC
	cache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	assert.Empty(t, cache.Entries, "LoadCache() should return empty entries for tampered cache")
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	setupTestEnv(t)

	// create cache file with invalid JSON
	cachePath, err := GetCachePath()
	require.NoError(t, err, "GetCachePath() error")

	// ensure config directory exists
	configDir := filepath.Dir(cachePath)
	require.NoError(t, os.MkdirAll(configDir, 0700), "MkdirAll() error")

	require.NoError(t, os.WriteFile(cachePath, []byte("{invalid json}"), 0600), "WriteFile() error")

	// load cache - should return empty cache due to invalid JSON
	cache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	assert.Empty(t, cache.Entries, "LoadCache() should return empty entries for invalid JSON")
}

func TestGetCachedResult_Hit(t *testing.T) {
	setupTestEnv(t)

	// create cache with entry
	cache := &VerificationCache{
		Entries: []CacheEntry{
			{
				FilePath:    "/path/to/file.bin",
				ContentHash: "abc123",
				Verified:    true,
				Timestamp:   time.Now(),
			},
		},
	}

	require.NoError(t, SaveCache(cache), "SaveCache() error")

	// lookup with matching hash
	result, err := GetCachedResult("/path/to/file.bin", "abc123")
	require.NoError(t, err, "GetCachedResult() error")

	require.NotNil(t, result, "GetCachedResult() returned nil, expected entry")

	assert.Equal(t, "/path/to/file.bin", result.FilePath)
	assert.Equal(t, "abc123", result.ContentHash)
	assert.True(t, result.Verified)
}

func TestGetCachedResult_Miss_DifferentHash(t *testing.T) {
	setupTestEnv(t)

	// create cache with entry
	cache := &VerificationCache{
		Entries: []CacheEntry{
			{
				FilePath:    "/path/to/file.bin",
				ContentHash: "abc123",
				Verified:    true,
				Timestamp:   time.Now(),
			},
		},
	}

	require.NoError(t, SaveCache(cache), "SaveCache() error")

	// lookup with different hash
	result, err := GetCachedResult("/path/to/file.bin", "xyz789")
	require.NoError(t, err, "GetCachedResult() error")

	assert.Nil(t, result, "GetCachedResult() returned entry, expected nil")
}

func TestGetCachedResult_Miss_DifferentPath(t *testing.T) {
	setupTestEnv(t)

	// create cache with entry
	cache := &VerificationCache{
		Entries: []CacheEntry{
			{
				FilePath:    "/path/to/file.bin",
				ContentHash: "abc123",
				Verified:    true,
				Timestamp:   time.Now(),
			},
		},
	}

	require.NoError(t, SaveCache(cache), "SaveCache() error")

	// lookup with path not in cache
	result, err := GetCachedResult("/different/path.bin", "abc123")
	require.NoError(t, err, "GetCachedResult() error")

	assert.Nil(t, result, "GetCachedResult() returned entry, expected nil")
}

func TestSetCachedResult_New(t *testing.T) {
	setupTestEnv(t)

	// add new entry
	require.NoError(t, SetCachedResult("/path/to/file.bin", "abc123", true), "SetCachedResult() error")

	// verify entry was added
	result, err := GetCachedResult("/path/to/file.bin", "abc123")
	require.NoError(t, err, "GetCachedResult() error")

	require.NotNil(t, result, "GetCachedResult() returned nil, expected entry")

	assert.Equal(t, "/path/to/file.bin", result.FilePath)
	assert.Equal(t, "abc123", result.ContentHash)
	assert.True(t, result.Verified)
}

func TestSetCachedResult_Update(t *testing.T) {
	setupTestEnv(t)

	// add initial entry
	require.NoError(t, SetCachedResult("/path/to/file.bin", "abc123", false), "SetCachedResult() initial error")

	// verify initial state
	result1, err := GetCachedResult("/path/to/file.bin", "abc123")
	require.NoError(t, err, "GetCachedResult() error")
	require.NotNil(t, result1, "GetCachedResult() returned nil")
	assert.False(t, result1.Verified, "Initial entry Verified should be false")
	timestamp1 := result1.Timestamp

	// sleep to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// update existing entry
	require.NoError(t, SetCachedResult("/path/to/file.bin", "abc123", true), "SetCachedResult() update error")

	// verify entry was updated
	result2, err := GetCachedResult("/path/to/file.bin", "abc123")
	require.NoError(t, err, "GetCachedResult() error")

	require.NotNil(t, result2, "GetCachedResult() returned nil after update")

	assert.True(t, result2.Verified, "Updated entry Verified should be true")
	assert.True(t, result2.Timestamp.After(timestamp1), "Updated entry Timestamp should be after original")

	// verify only one entry exists (not duplicated)
	cache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	assert.Len(t, cache.Entries, 1, "expected 1 entry (no duplicates)")
}

func TestSetCachedResult_MultipleEntries(t *testing.T) {
	setupTestEnv(t)

	// add multiple different entries
	entries := []struct {
		path     string
		hash     string
		verified bool
	}{
		{"/path/to/file1.bin", "hash1", true},
		{"/path/to/file2.bin", "hash2", false},
		{"/path/to/file1.bin", "hash3", true}, // same path, different hash
	}

	for _, entry := range entries {
		require.NoError(t, SetCachedResult(entry.path, entry.hash, entry.verified), "SetCachedResult(%s, %s) error", entry.path, entry.hash)
	}

	// verify all entries exist
	for _, entry := range entries {
		result, err := GetCachedResult(entry.path, entry.hash)
		require.NoError(t, err, "GetCachedResult(%s, %s) error", entry.path, entry.hash)
		require.NotNil(t, result, "GetCachedResult(%s, %s) returned nil", entry.path, entry.hash)
		assert.Equal(t, entry.verified, result.Verified, "GetCachedResult(%s, %s) Verified mismatch", entry.path, entry.hash)
	}

	// verify total count
	cache, err := LoadCache()
	require.NoError(t, err, "LoadCache() error")

	assert.Len(t, cache.Entries, len(entries))
}

func TestCachePath_ConsistentLocation(t *testing.T) {
	setupTestEnv(t)

	// get cache path multiple times
	path1, err := GetCachePath()
	require.NoError(t, err, "GetCachePath() error")

	path2, err := GetCachePath()
	require.NoError(t, err, "GetCachePath() error")

	assert.Equal(t, path1, path2, "GetCachePath() returned different paths")

	// verify path is in expected location
	assert.True(t, filepath.IsAbs(path1), "GetCachePath() returned relative path: %s", path1)
	// path should contain "sageox" somewhere (either ~/.sageox/config/ or ~/.config/sageox/)
	assert.Contains(t, path1, "sageox", "GetCachePath() not in sageox directory")
	assert.Equal(t, "verification-cache.json", filepath.Base(path1), "GetCachePath() unexpected filename")
}
