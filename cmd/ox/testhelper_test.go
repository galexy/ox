package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sageox/ox/internal/config"
	"github.com/stretchr/testify/require"
)

// createInitializedProject creates a temp directory with .sageox/ initialized.
// Use this when tests need a project that can save local configs, team contexts, etc.
// Returns the temp directory path (cleaned up automatically by t.TempDir()).
func createInitializedProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	requireSageoxDir(t, tmpDir)
	return tmpDir
}

// createInitializedProjectWithConfig creates a temp directory with .sageox/ and config.json.
// Use this when tests need a fully initialized project with project config.
// Returns the temp directory path.
func createInitializedProjectWithConfig(t *testing.T, cfg *config.ProjectConfig) string {
	t.Helper()
	tmpDir := createInitializedProject(t)

	if cfg == nil {
		cfg = &config.ProjectConfig{
			ProjectID:   "test_project",
			WorkspaceID: "test_workspace",
		}
	}

	require.NoError(t, config.SaveProjectConfig(tmpDir, cfg), "failed to save project config")
	return tmpDir
}

// requireSageoxDir creates the .sageox/ directory in the given path.
// Use this to add initialization to an existing temp directory.
func requireSageoxDir(t *testing.T, projectRoot string) {
	t.Helper()
	sageoxDir := filepath.Join(projectRoot, ".sageox")
	require.NoError(t, os.MkdirAll(sageoxDir, 0755), "failed to create .sageox/")
}
