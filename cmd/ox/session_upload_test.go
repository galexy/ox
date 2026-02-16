package main

import (
	"testing"

	"github.com/sageox/ox/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestCheckUploadAccess_NoRepoID(t *testing.T) {
	// bare temp dir with no .sageox/config.json → GetRepoID returns "" → fail-open returns nil
	tmpDir := t.TempDir()
	err := checkUploadAccess(tmpDir)
	assert.NoError(t, err, "should fail-open when no repo ID exists")
}

func TestCheckUploadAccess_NoProjectConfig(t *testing.T) {
	// nonexistent directory → GetRepoID returns "" → fail-open returns nil
	err := checkUploadAccess("/nonexistent/path/that/does/not/exist")
	assert.NoError(t, err, "should fail-open when project root does not exist")
}

func TestCheckUploadAccess_EmptyRepoID(t *testing.T) {
	// initialized project but config has no repo_id → fail-open returns nil
	tmpDir := createInitializedProject(t)
	err := checkUploadAccess(tmpDir)
	assert.NoError(t, err, "should fail-open when config exists but repo_id is empty")
}

func TestCheckUploadAccess_RepoIDButNoAuth(t *testing.T) {
	// project with repo_id set but no valid auth token → fail-open returns nil
	// (auth.GetTokenForEndpoint will fail for a fake endpoint)
	tmpDir := createInitializedProjectWithConfig(t, &config.ProjectConfig{
		RepoID:   "repo_01test1234567890",
		Endpoint: "https://fake.example.com",
	})
	err := checkUploadAccess(tmpDir)
	assert.NoError(t, err, "should fail-open when auth token is unavailable")
}
