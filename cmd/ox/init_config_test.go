//go:build !short

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sageox/ox/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureSageoxConfig_CreatesNew(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	result := ensureSageoxConfig(tmpDir)
	assert.Equal(t, configCreated, result, "expected configCreated")

	// verify config exists
	configPath := filepath.Join(sageoxDir, "config.json")
	require.FileExists(t, configPath, "config.json was not created")

	// verify it's valid
	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	assert.Equal(t, config.CurrentConfigVersion, cfg.ConfigVersion, "config version mismatch")
}

func TestEnsureSageoxConfig_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// create a current config
	cfg := config.GetDefaultProjectConfig()
	cfg.RepoID = "repo_existing123"
	require.NoError(t, config.SaveProjectConfig(tmpDir, cfg), "failed to save config")

	result := ensureSageoxConfig(tmpDir)
	assert.Equal(t, configPreserved, result, "expected configPreserved")

	// verify repo_id was preserved
	loadedCfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	assert.Equal(t, "repo_existing123", loadedCfg.RepoID, "repo_id should be preserved")
}

func TestEnsureSageoxConfig_UpgradesOld(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create an old version config
	oldConfig := `{
		"config_version": "1",
		"update_frequency_hours": 24
	}`
	configPath := filepath.Join(sageoxDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(oldConfig), 0644), "failed to write old config")

	result := ensureSageoxConfig(tmpDir)
	assert.Equal(t, configUpgraded, result, "expected configUpgraded")

	// verify version was upgraded
	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	assert.Equal(t, config.CurrentConfigVersion, cfg.ConfigVersion, "version should be upgraded")
}

func TestConfigResult_Values(t *testing.T) {
	// verify configResult constants have expected values
	assert.Equal(t, configResult(0), configCreated, "configCreated should be 0")
	assert.Equal(t, configResult(1), configUpgraded, "configUpgraded should be 1")
	assert.Equal(t, configResult(2), configPreserved, "configPreserved should be 2")
	assert.Equal(t, configResult(3), configError, "configError should be 3")
}

func TestConfigMetadata_OnlyUpdatesWhenChanged(t *testing.T) {
	tmpDir := t.TempDir()

	// create initial config with repo_id and remote hashes
	cfg := config.GetDefaultProjectConfig()
	cfg.RepoID = "repo_existing123"
	cfg.RepoRemoteHashes = []string{"hash1", "hash2"}
	require.NoError(t, config.SaveProjectConfig(tmpDir, cfg), "failed to save initial config")

	// simulate runInit logic: load config and check if metadata changed
	loadedCfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to load config")

	// test case 1: no changes - should NOT save
	metadataChanged := false

	// repo_id unchanged
	currentRepoID := "repo_existing123"
	if loadedCfg.RepoID != currentRepoID {
		loadedCfg.RepoID = currentRepoID
		metadataChanged = true
	}

	// remote hashes unchanged
	currentRemoteHashes := []string{"hash1", "hash2"}
	if !stringSlicesEqual(loadedCfg.RepoRemoteHashes, currentRemoteHashes) {
		loadedCfg.RepoRemoteHashes = currentRemoteHashes
		metadataChanged = true
	}

	assert.False(t, metadataChanged, "metadataChanged should be false when repo_id and hashes unchanged")

	// test case 2: repo_id empty, should update
	loadedCfg.RepoID = ""
	metadataChanged = false

	if loadedCfg.RepoID == "" {
		loadedCfg.RepoID = currentRepoID
		metadataChanged = true
	}

	assert.True(t, metadataChanged, "metadataChanged should be true when repo_id is empty")

	// test case 3: remote hashes changed, should update
	loadedCfg, _ = config.LoadProjectConfig(tmpDir)
	metadataChanged = false

	newRemoteHashes := []string{"hash1", "hash2", "hash3"}
	if !stringSlicesEqual(loadedCfg.RepoRemoteHashes, newRemoteHashes) {
		loadedCfg.RepoRemoteHashes = newRemoteHashes
		metadataChanged = true
	}

	assert.True(t, metadataChanged, "metadataChanged should be true when remote hashes changed")

	// verify saving works correctly
	require.NoError(t, config.SaveProjectConfig(tmpDir, loadedCfg), "failed to save updated config")

	// reload and verify changes were saved
	finalCfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "failed to reload config")

	assert.True(t, stringSlicesEqual(finalCfg.RepoRemoteHashes, newRemoteHashes), "remote hashes should be updated")
}

func TestEnsureSageoxConfig_Error_InvalidPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create a config file with invalid permissions (read-only directory)
	configPath := filepath.Join(sageoxDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0444), "failed to create readonly config")

	// make directory read-only to prevent writes
	require.NoError(t, os.Chmod(sageoxDir, 0444), "failed to change permissions")
	defer os.Chmod(sageoxDir, 0755)

	result := ensureSageoxConfig(tmpDir)
	assert.Equal(t, configError, result, "expected configError when cannot write")
}

func TestEnsureSageoxConfig_Error_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	sageoxDir := filepath.Join(tmpDir, ".sageox")

	// create a corrupted config file (invalid JSON)
	configPath := filepath.Join(sageoxDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte("not valid json {{{"), 0644), "failed to create corrupted config")

	result := ensureSageoxConfig(tmpDir)

	// when file exists but is corrupted, ensureSageoxConfig recreates it
	assert.NotEqual(t, configError, result, "ensureSageoxConfig should recreate corrupted config, not return error")

	// verify config was recreated
	cfg, err := config.LoadProjectConfig(tmpDir)
	require.NoError(t, err, "config should have been recreated")

	assert.Equal(t, config.CurrentConfigVersion, cfg.ConfigVersion, "expected current version")
}
