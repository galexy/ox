//go:build !short

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sageox/ox/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAuthTestDir creates a temporary directory and configures auth to use it.
// It calls skipIntegration to skip with -short flag.
func setupAuthTestDir(t *testing.T) string {
	skipIntegration(t)
	tempDir := t.TempDir()

	// enable XDG mode and set temp directories for test isolation
	// without OX_XDG_ENABLE, the paths package uses cached HOME from previous tests
	t.Setenv("OX_XDG_ENABLE", "1")
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	return tempDir
}

// createAuthFile creates an auth file with the specified permissions
func createAuthFile(t *testing.T, perm os.FileMode) string {
	t.Helper()

	authPath, err := auth.GetAuthFilePath()
	require.NoError(t, err, "auth.GetAuthFilePath() error")

	// ensure directory exists
	configDir := filepath.Dir(authPath)
	require.NoError(t, os.MkdirAll(configDir, 0700), "os.MkdirAll() error")

	// create auth file with test content
	testContent := []byte(`{
  "access_token": "test-token",
  "refresh_token": "test-refresh",
  "expires_at": "2025-12-31T23:59:59Z",
  "token_type": "Bearer",
  "scope": "openid profile email",
  "user_info": {
    "user_id": "user123",
    "email": "user@example.com",
    "name": "Test User"
  }
}`)

	require.NoError(t, os.WriteFile(authPath, testContent, perm), "os.WriteFile() error")

	return authPath
}

func TestCheckAuthFilePermissions_NoAuthFile(t *testing.T) {
	setupAuthTestDir(t)

	result := checkAuthFilePermissions(false)

	assert.True(t, result.skipped, "checkAuthFilePermissions() skipped should be true when auth file doesn't exist")
	assert.Equal(t, "Auth file", result.name, "name mismatch")
	assert.Equal(t, "not logged in", result.message, "message mismatch")
}

func TestCheckAuthFilePermissions_CorrectPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	setupAuthTestDir(t)
	createAuthFile(t, 0600)

	result := checkAuthFilePermissions(false)

	assert.True(t, result.passed, "checkAuthFilePermissions() passed should be true for correct permissions")
	assert.Equal(t, "Permissions", result.name, "name mismatch")
	assert.Equal(t, "0600 (secure)", result.message, "message mismatch")
	assert.False(t, result.skipped, "skipped should be false")
}

func TestCheckAuthFilePermissions_WrongPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	setupAuthTestDir(t)
	createAuthFile(t, 0644)

	result := checkAuthFilePermissions(false)

	assert.False(t, result.passed, "checkAuthFilePermissions() passed should be false for wrong permissions")
	assert.Equal(t, "Permissions", result.name, "name mismatch")
	assert.Equal(t, "insecure 0644", result.message, "message mismatch")
	assert.Equal(t, "Run `ox doctor --fix`", result.detail, "detail mismatch")
	assert.False(t, result.skipped, "skipped should be false")
}

func TestCheckAuthFilePermissions_WrongPermissionsWithFix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	setupAuthTestDir(t)
	authPath := createAuthFile(t, 0644)

	// verify initial wrong permissions
	info, err := os.Stat(authPath)
	require.NoError(t, err, "os.Stat() error")
	require.Equal(t, os.FileMode(0644), info.Mode().Perm(), "initial permissions mismatch")

	// run check with fix enabled
	result := checkAuthFilePermissions(true)

	assert.True(t, result.passed, "checkAuthFilePermissions(fix=true) passed should be true after fixing")
	assert.Equal(t, "Permissions", result.name, "name mismatch")
	assert.Equal(t, "fixed to 0600", result.message, "message mismatch")

	// verify permissions were actually fixed
	info, err = os.Stat(authPath)
	require.NoError(t, err, "os.Stat() after fix error")

	actualMode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	assert.Equal(t, expectedMode, actualMode, "permissions after fix mismatch")
}

func TestCheckAuthFilePermissions_StatError(t *testing.T) {
	setupAuthTestDir(t)

	// create auth file in a directory we'll remove
	authPath := createAuthFile(t, 0600)
	configDir := filepath.Dir(authPath)

	// remove the directory to trigger stat error
	require.NoError(t, os.RemoveAll(configDir), "os.RemoveAll() error")

	result := checkAuthFilePermissions(false)

	// on unix, stat of non-existent file returns IsNotExist, which is handled as "not logged in"
	// but on some systems or with corrupted paths, we might get other errors
	if !result.skipped && !result.passed {
		// if not skipped, should be a failure
		assert.Equal(t, "Auth file", result.name, "name mismatch")
		assert.True(t, result.message == "stat failed" || result.message == "not logged in", "message should be 'stat failed' or 'not logged in'")
	} else if result.skipped {
		// skipped is also acceptable for non-existent file
		assert.Equal(t, "not logged in", result.message, "message mismatch")
	}
}

func TestCheckAuthFilePermissions_DirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	setupAuthTestDir(t)
	authPath := createAuthFile(t, 0600)

	// change directory permissions to be insecure
	configDir := filepath.Dir(authPath)
	require.NoError(t, os.Chmod(configDir, 0755), "os.Chmod(dir) error")

	// check should only care about file permissions, not directory
	result := checkAuthFilePermissions(false)

	assert.True(t, result.passed, "checkAuthFilePermissions() passed should be true (only file permissions matter)")
}

func TestCheckAuthFilePermissions_NotLoggedIn(t *testing.T) {
	// use temp directory that won't have auth file
	tempDir := t.TempDir()
	// test with random value - any non-empty string enables XDG mode
	t.Setenv("OX_XDG_ENABLE", fmt.Sprintf("test-%d", time.Now().UnixNano()))
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	result := checkAuthFilePermissions(false)

	assert.True(t, result.skipped, "checkAuthFilePermissions() skipped should be true when not logged in")
	assert.Equal(t, "Auth file", result.name, "name mismatch")
	assert.Equal(t, "not logged in", result.message, "message mismatch")
}
