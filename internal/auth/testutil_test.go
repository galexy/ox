package auth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// NewTestClient creates an AuthClient with isolated storage for parallel testing.
// Each test gets its own temp directory, enabling t.Parallel().
// Uses a nested path (tempDir/sageox) to match production structure where
// configDir is ~/.config/sageox and auth.json is at configDir/auth.json.
// This allows permission tests to verify directory creation with 0700.
func NewTestClient(t *testing.T) *AuthClient {
	t.Helper()
	return NewAuthClientWithDir(filepath.Join(t.TempDir(), "sageox"))
}

// CreateTestTokenForClient creates a test token and saves it to the given client
func CreateTestTokenForClient(t *testing.T, client *AuthClient, expiresIn time.Duration) *StoredToken {
	t.Helper()
	token := &StoredToken{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(expiresIn),
		TokenType:    "Bearer",
		Scope:        "user:profile sageox:write",
		UserInfo: UserInfo{
			UserID: "user-123",
			Email:  "test@example.com",
			Name:   "Test User",
		},
	}
	require.NoError(t, client.SaveToken(token), "failed to save test token")
	return token
}

// --- Legacy functions for backward compatibility during migration ---
// NOTE: These use global state (configDirOverride) and cannot be used with t.Parallel().
// Prefer NewTestClient(t) and AuthClient methods for new tests.

// setupTestDir creates a temporary directory and overrides the config path.
// DEPRECATED: Cannot be used with t.Parallel(). Use NewTestClient(t) instead.
func setupTestDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	// save original override value
	originalOverride := configDirOverride

	// set override for this test
	configDirOverride = tempDir

	// restore on cleanup
	t.Cleanup(func() {
		configDirOverride = originalOverride
	})

	return tempDir
}

// SetTestConfigDir sets a temporary config directory for testing.
// DEPRECATED: Use NewTestClient(t) instead.
func SetTestConfigDir(t *testing.T) func() {
	t.Helper()
	tempDir := t.TempDir()
	oldDir := configDirOverride
	configDirOverride = tempDir
	return func() {
		configDirOverride = oldDir
	}
}

// createTestToken returns a test token for testing (not saved).
// DEPRECATED: Use CreateTestTokenForClient(t, client, expiresIn) instead.
func createTestToken(expiresIn time.Duration) *StoredToken {
	return &StoredToken{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(expiresIn),
		TokenType:    "Bearer",
		Scope:        "user:profile sageox:write",
		UserInfo: UserInfo{
			UserID: "user-123",
			Email:  "test@example.com",
			Name:   "Test User",
		},
	}
}

// CreateTestToken creates a test token and saves it using global functions.
// DEPRECATED: Use CreateTestTokenForClient(t, client, expiresIn) instead.
func CreateTestToken(t *testing.T, expiresIn time.Duration) *StoredToken {
	t.Helper()
	token := createTestToken(expiresIn)
	require.NoError(t, SaveToken(token), "failed to save test token")
	return token
}
