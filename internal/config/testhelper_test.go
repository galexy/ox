package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateInitializedProject creates a temp directory with .sageox/ initialized.
// Use this when tests need a project that can save local configs, team contexts, etc.
// Returns the temp directory path (cleaned up automatically by t.TempDir()).
func CreateInitializedProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	require.NoError(t, os.MkdirAll(sageoxDir, 0755), "failed to create .sageox/")
	return tmpDir
}

// CreateInitializedProjectWithConfig creates a temp directory with .sageox/ and config.json.
// Use this when tests need a fully initialized project with project config.
// Returns the temp directory path.
func CreateInitializedProjectWithConfig(t *testing.T, cfg *ProjectConfig) string {
	t.Helper()
	tmpDir := CreateInitializedProject(t)

	if cfg == nil {
		cfg = &ProjectConfig{
			ProjectID:   "test_project",
			WorkspaceID: "test_workspace",
		}
	}

	require.NoError(t, SaveProjectConfig(tmpDir, cfg), "failed to save project config")
	return tmpDir
}

// RequireSageoxDir creates the .sageox/ directory in the given path.
// Use this to add initialization to an existing temp directory.
func RequireSageoxDir(t *testing.T, projectRoot string) {
	t.Helper()
	sageoxDir := filepath.Join(projectRoot, ".sageox")
	require.NoError(t, os.MkdirAll(sageoxDir, 0755), "failed to create .sageox/")
}
