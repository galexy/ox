package uninstall

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSageoxFiles_NonexistentDirectory(t *testing.T) {
	// create temp directory without .sageox
	tmpDir := t.TempDir()

	items, err := FindSageoxFiles(tmpDir)
	require.NoError(t, err, "expected no error for nonexistent directory")
	assert.Nil(t, items, "expected nil items for nonexistent directory")
}

func TestFindSageoxFiles_EmptyDirectory(t *testing.T) {
	// create temp git repo with empty .sageox
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// create empty .sageox directory
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	items, err := FindSageoxFiles(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, items, "expected 0 items for empty directory")
}

func TestFindSageoxFiles_WithFiles(t *testing.T) {
	// create temp git repo with files in .sageox
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// create .sageox directory with files
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	// create a tracked file
	trackedFile := filepath.Join(sageoxDir, "config.json")
	err = os.WriteFile(trackedFile, []byte(`{"version":"1.0"}`), 0644)
	require.NoError(t, err, "failed to create config.json")

	// add to git
	cmd = exec.Command("git", "add", ".sageox/config.json")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "git add failed")

	// create an untracked file
	untrackedFile := filepath.Join(sageoxDir, "temp.log")
	err = os.WriteFile(untrackedFile, []byte("log data"), 0644)
	require.NoError(t, err, "failed to create temp.log")

	// create a subdirectory with a file
	cacheDir := filepath.Join(sageoxDir, "cache")
	err = os.Mkdir(cacheDir, 0755)
	require.NoError(t, err, "failed to create cache dir")

	cacheFile := filepath.Join(cacheDir, "data.json")
	err = os.WriteFile(cacheFile, []byte(`{"cached":true}`), 0644)
	require.NoError(t, err, "failed to create cache file")

	items, err := FindSageoxFiles(tmpDir)
	require.NoError(t, err)

	// should find: config.json (tracked), temp.log (untracked), cache/ (dir), cache/data.json (untracked)
	assert.GreaterOrEqual(t, len(items), 3, "expected at least 3 items")

	// verify tracked status
	var foundTracked, foundUntracked bool
	for _, item := range items {
		if item.RelPath == ".sageox/config.json" {
			foundTracked = true
			assert.True(t, item.IsTracked, "config.json should be tracked")
			assert.False(t, item.IsDir, "config.json should not be a directory")
			assert.NotZero(t, item.Size, "config.json should have non-zero size")
		}
		if item.RelPath == ".sageox/temp.log" {
			foundUntracked = true
			assert.False(t, item.IsTracked, "temp.log should not be tracked")
		}
	}

	assert.True(t, foundTracked, "did not find tracked config.json")
	assert.True(t, foundUntracked, "did not find untracked temp.log")
}

func TestRemoveSageoxDir_NonexistentDirectory(t *testing.T) {
	// create temp directory without .sageox
	tmpDir := t.TempDir()

	// should not error when directory doesn't exist
	err := RemoveSageoxDir(tmpDir, false)
	assert.NoError(t, err, "expected no error for nonexistent directory")
}

func TestRemoveSageoxDir_DryRun(t *testing.T) {
	// create temp git repo with .sageox
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// create .sageox with a file
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	testFile := filepath.Join(sageoxDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err, "failed to create test file")

	// dry run should not remove anything
	err = RemoveSageoxDir(tmpDir, true)
	require.NoError(t, err, "dry run failed")

	// verify directory still exists
	_, err = os.Stat(sageoxDir)
	assert.False(t, os.IsNotExist(err), "dry run should not remove directory")

	// verify file still exists
	_, err = os.Stat(testFile)
	assert.False(t, os.IsNotExist(err), "dry run should not remove files")
}

func TestRemoveSageoxDir_UntrackedFiles(t *testing.T) {
	// create temp git repo with untracked files in .sageox
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// create .sageox with files
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	testFile := filepath.Join(sageoxDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err, "failed to create test file")

	// remove should work
	err = RemoveSageoxDir(tmpDir, false)
	require.NoError(t, err, "remove failed")

	// verify directory is gone
	_, err = os.Stat(sageoxDir)
	assert.True(t, os.IsNotExist(err), "directory should be removed")
}

func TestRemoveSageoxDir_TrackedFiles(t *testing.T) {
	// create temp git repo with tracked files in .sageox
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// configure git user for commits (must specify tmpDir to avoid polluting real repo)
	emailCmd := exec.Command("git", "config", "user.email", "test@example.com")
	emailCmd.Dir = tmpDir
	emailCmd.Run()
	nameCmd := exec.Command("git", "config", "user.name", "Test User")
	nameCmd.Dir = tmpDir
	nameCmd.Run()

	// create .sageox with a tracked file
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	testFile := filepath.Join(sageoxDir, "config.json")
	err = os.WriteFile(testFile, []byte(`{"test":true}`), 0644)
	require.NoError(t, err, "failed to create test file")

	// add and commit
	cmd = exec.Command("git", "add", ".sageox/config.json")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "git add failed")

	cmd = exec.Command("git", "commit", "-m", "add config")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "git commit failed")

	// remove should work
	err = RemoveSageoxDir(tmpDir, false)
	require.NoError(t, err, "remove failed")

	// verify directory is gone
	_, err = os.Stat(sageoxDir)
	assert.True(t, os.IsNotExist(err), "directory should be removed")

	// verify git status shows the deletion
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	require.NoError(t, err, "git status failed")

	statusOutput := string(output)
	assert.NotEmpty(t, statusOutput, "expected git status to show deletion")
}

func TestRemoveSageoxDir_MixedTrackedUntracked(t *testing.T) {
	// create temp git repo with both tracked and untracked files
	tmpDir := t.TempDir()

	// initialize git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// configure git user (must specify tmpDir to avoid polluting real repo)
	emailCmd := exec.Command("git", "config", "user.email", "test@example.com")
	emailCmd.Dir = tmpDir
	emailCmd.Run()
	nameCmd := exec.Command("git", "config", "user.name", "Test User")
	nameCmd.Dir = tmpDir
	nameCmd.Run()

	// create .sageox
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	err := os.Mkdir(sageoxDir, 0755)
	require.NoError(t, err, "failed to create .sageox")

	// create tracked file
	trackedFile := filepath.Join(sageoxDir, "config.json")
	err = os.WriteFile(trackedFile, []byte(`{"version":"1.0"}`), 0644)
	require.NoError(t, err, "failed to create tracked file")

	cmd = exec.Command("git", "add", ".sageox/config.json")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "git add failed")

	cmd = exec.Command("git", "commit", "-m", "add config")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err, "git commit failed")

	// create untracked file
	untrackedFile := filepath.Join(sageoxDir, "temp.log")
	err = os.WriteFile(untrackedFile, []byte("log"), 0644)
	require.NoError(t, err, "failed to create untracked file")

	// remove should handle both
	err = RemoveSageoxDir(tmpDir, false)
	require.NoError(t, err, "remove failed")

	// verify directory is completely gone
	_, err = os.Stat(sageoxDir)
	assert.True(t, os.IsNotExist(err), "directory should be completely removed")
}
