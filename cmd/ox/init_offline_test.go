//go:build !short

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sageox/ox/internal/config"
	"github.com/sageox/ox/internal/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOfflineToOnlinePromotion_PreservesRepoID tests that re-running ox init
// on an offline-initialized repo preserves the existing repo_id
func TestOfflineToOnlinePromotion_PreservesRepoID(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// simulate offline initialization: create config without team_id
	existingRepoID := "repo_01jfk3mab123offline"
	offlineConfig := &config.ProjectConfig{
		ConfigVersion: config.CurrentConfigVersion,
		RepoID:        existingRepoID,
		// TeamID intentionally empty (offline mode)
	}
	require.NoError(t, config.SaveProjectConfig(tmpDir, offlineConfig), "failed to save offline config")

	// create existing marker file (simulating offline init)
	currentEndpoint := endpoint.Get()
	existingUUID := extractUUIDSuffix(existingRepoID)
	markerPath := filepath.Join(sageoxDir, ".repo_"+existingUUID)
	existingMarker := map[string]string{
		"repo_id":  existingRepoID,
		"type":     "git",
		"init_at":  "2025-01-01T00:00:00Z",
		"endpoint": currentEndpoint,
	}
	data, _ := json.Marshal(existingMarker)
	require.NoError(t, os.WriteFile(markerPath, data, 0644), "failed to write existing marker")

	// simulate re-running init (online) - detect existing markers
	existingMarkerRepoIDs, err := detectExistingRepoMarkers(sageoxDir)
	require.NoError(t, err, "detectExistingRepoMarkers failed")

	// verify existing repo_id is preserved
	require.Len(t, existingMarkerRepoIDs, 1, "expected 1 existing marker")

	assert.Equal(t, existingRepoID, existingMarkerRepoIDs[0], "preserved repo_id mismatch")

	// verify config still has the same repo_id
	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	assert.Equal(t, existingRepoID, cfg.RepoID, "config repo_id changed")
}

// TestOfflineToOnlinePromotion_TeamIDUpdate tests that team_id can be added
// to a previously offline-initialized repo
func TestOfflineToOnlinePromotion_TeamIDUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)

	// simulate offline initialization: create config without team_id
	existingRepoID := "repo_01jfk3mab456offline"
	offlineConfig := &config.ProjectConfig{
		ConfigVersion: config.CurrentConfigVersion,
		RepoID:        existingRepoID,
		// TeamID intentionally empty (offline mode)
	}
	require.NoError(t, config.SaveProjectConfig(tmpDir, offlineConfig), "failed to save offline config")

	// verify no team_id initially
	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	assert.Empty(t, cfg.TeamID, "expected empty team_id")

	// simulate online registration success by updating config with team_id
	// (this is what runInit does after successful API registration)
	cfg.TeamID = "team_abc123"
	require.NoError(t, config.SaveProjectConfig(tmpDir, cfg), "failed to save updated config")

	// verify team_id was added
	updatedCfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load updated config")

	assert.Equal(t, "team_abc123", updatedCfg.TeamID, "team_id mismatch")

	// verify repo_id is still preserved
	assert.Equal(t, existingRepoID, updatedCfg.RepoID, "repo_id changed during promotion")
}

// TestOfflineToOnlinePromotion_NoDuplicateMarkers tests that re-running init
// does not create duplicate marker files
func TestOfflineToOnlinePromotion_NoDuplicateMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create existing marker file (simulating offline init)
	currentEndpoint := endpoint.Get()
	existingRepoID := "repo_01jfk3mab789offline"
	existingUUID := extractUUIDSuffix(existingRepoID)
	markerPath := filepath.Join(sageoxDir, ".repo_"+existingUUID)
	existingMarker := map[string]string{
		"repo_id":  existingRepoID,
		"type":     "git",
		"init_at":  "2025-01-01T00:00:00Z",
		"endpoint": currentEndpoint,
	}
	data, _ := json.Marshal(existingMarker)
	require.NoError(t, os.WriteFile(markerPath, data, 0644), "failed to write existing marker")

	// detect existing markers (simulating start of re-init)
	existingMarkerRepoIDs, _ := detectExistingRepoMarkers(sageoxDir)
	markerAlreadyExists := len(existingMarkerRepoIDs) > 0

	// simulate the check in runInit that prevents creating new marker
	require.True(t, markerAlreadyExists, "expected marker to be detected as existing")

	// when markerAlreadyExists is true, runInit should NOT create a new marker
	// verify only one marker exists
	entries, err := os.ReadDir(sageoxDir)
	require.NoError(t, err, "failed to read .sageox")

	markerCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".repo_") {
			markerCount++
		}
	}

	assert.Equal(t, 1, markerCount, "expected exactly 1 marker file")
}

// TestOfflineConfig_HasNoTeamID verifies detection of offline-initialized repos
func TestOfflineConfig_HasNoTeamID(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)

	// create config without team_id (offline mode)
	offlineConfig := &config.ProjectConfig{
		ConfigVersion: config.CurrentConfigVersion,
		RepoID:        "repo_offline123",
		// TeamID intentionally empty
	}
	require.NoError(t, config.SaveProjectConfig(tmpDir, offlineConfig), "failed to save config")

	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	// this is how ox doctor detects offline mode
	isOffline := cfg.TeamID == ""
	assert.True(t, isOffline, "expected config to be detected as offline (no team_id)")
}

// TestOnlineConfig_HasTeamID verifies detection of online-registered repos
func TestOnlineConfig_HasTeamID(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)

	// create config with team_id (online mode)
	onlineConfig := &config.ProjectConfig{
		ConfigVersion: config.CurrentConfigVersion,
		RepoID:        "repo_online123",
		TeamID:        "team_xyz789",
	}
	require.NoError(t, config.SaveProjectConfig(tmpDir, onlineConfig), "failed to save config")

	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	// this is how ox doctor detects online mode
	isOnline := cfg.TeamID != ""
	assert.True(t, isOnline, "expected config to be detected as online (has team_id)")
}

// TestDetectExistingRepoMarkers_MultipleMarkers tests detection of concurrent inits
// (multiple .repo_* files created by different developers)
func TestDetectExistingRepoMarkers_MultipleMarkersOffline(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create multiple marker files (simulating concurrent inits from different devs)
	currentEndpoint := endpoint.Get()
	markers := []struct {
		repoID string
		initAt string
	}{
		{"repo_01aaa111", "2025-01-01T00:00:00Z"}, // oldest (should be first after sort)
		{"repo_01ccc333", "2025-01-03T00:00:00Z"}, // newest
		{"repo_01bbb222", "2025-01-02T00:00:00Z"}, // middle
	}

	for _, m := range markers {
		uuid := extractUUIDSuffix(m.repoID)
		markerPath := filepath.Join(sageoxDir, ".repo_"+uuid)
		data, _ := json.Marshal(map[string]string{
			"repo_id":  m.repoID,
			"type":     "git",
			"init_at":  m.initAt,
			"endpoint": currentEndpoint,
		})
		require.NoError(t, os.WriteFile(markerPath, data, 0644), "failed to write marker %s", m.repoID)
	}

	// detect all markers
	repoIDs, err := detectExistingRepoMarkers(sageoxDir)
	require.NoError(t, err, "detectExistingRepoMarkers failed")

	// should find all 3 markers
	require.Len(t, repoIDs, 3, "expected 3 repo IDs")

	// should be sorted (UUIDv7 is time-sortable, so alphabetical = chronological)
	assert.Equal(t, "repo_01aaa111", repoIDs[0], "expected first repo_id to be oldest")
}

// TestDetectExistingRepoMarkers_SortOrderOffline verifies markers are sorted by repo_id
// (UUIDv7 is time-sortable, so alphabetical order = chronological order)
func TestDetectExistingRepoMarkers_SortOrderOffline(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create markers in reverse order to test sorting
	currentEndpoint := endpoint.Get()
	repoIDs := []string{"repo_zzz999", "repo_aaa111", "repo_mmm555"}
	for _, id := range repoIDs {
		uuid := extractUUIDSuffix(id)
		markerPath := filepath.Join(sageoxDir, ".repo_"+uuid)
		data, _ := json.Marshal(map[string]string{"repo_id": id, "endpoint": currentEndpoint})
		require.NoError(t, os.WriteFile(markerPath, data, 0644), "failed to write marker")
	}

	detected, err := detectExistingRepoMarkers(sageoxDir)
	require.NoError(t, err, "detectExistingRepoMarkers failed")

	// verify sorted order
	expected := []string{"repo_aaa111", "repo_mmm555", "repo_zzz999"}
	for i, id := range expected {
		assert.Equal(t, id, detected[i], "position %d mismatch", i)
	}
}
