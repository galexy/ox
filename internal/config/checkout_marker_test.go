package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutMarker_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	marker := &CheckoutMarker{
		Type:     "ledger",
		Endpoint: "https://api.sageox.ai",
		RepoID:   "repo-abc123",
	}

	err := SaveCheckoutMarker(tmpDir, marker)
	require.NoError(t, err)

	// verify file exists
	markerPath := filepath.Join(tmpDir, ".sageox", "checkout.json")
	_, err = os.Stat(markerPath)
	require.NoError(t, err, "marker file should exist")

	// verify .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".sageox", ".gitignore")
	_, err = os.Stat(gitignorePath)
	require.NoError(t, err, ".gitignore should exist")

	gitignoreContent, _ := os.ReadFile(gitignorePath)
	assert.Contains(t, string(gitignoreContent), "checkout.json")

	// load and verify
	loaded, err := LoadCheckoutMarker(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "ledger", loaded.Type)
	assert.Equal(t, "https://api.sageox.ai", loaded.Endpoint)
	assert.Equal(t, "repo-abc123", loaded.RepoID)
}

func TestCheckoutMarker_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	marker, err := LoadCheckoutMarker(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, marker, "should return nil for non-existent marker")
}

func TestCreateLedgerMarker(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateLedgerMarker(tmpDir, "https://api.test.sageox.ai", "repo-xyz789")
	require.NoError(t, err)

	loaded, err := LoadCheckoutMarker(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "ledger", loaded.Type)
	assert.Equal(t, "https://api.test.sageox.ai", loaded.Endpoint)
	assert.Equal(t, "repo-xyz789", loaded.RepoID)
	assert.Empty(t, loaded.TeamID)
	assert.False(t, loaded.CheckedOutAt.IsZero())
}

func TestCreateTeamContextMarker(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateTeamContextMarker(tmpDir, "https://api.sageox.ai", "team-eng-123")
	require.NoError(t, err)

	loaded, err := LoadCheckoutMarker(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "team-context", loaded.Type)
	assert.Equal(t, "https://api.sageox.ai", loaded.Endpoint)
	assert.Equal(t, "team-eng-123", loaded.TeamID)
	assert.Empty(t, loaded.RepoID)
	assert.False(t, loaded.CheckedOutAt.IsZero())
}

func TestValidateLedgerMarker_Match(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-abc123")
	require.NoError(t, err)

	err = ValidateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-abc123")
	assert.NoError(t, err)
}

func TestValidateLedgerMarker_EndpointMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateLedgerMarker(tmpDir, "https://api.test.sageox.ai", "repo-abc123")
	require.NoError(t, err)

	err = ValidateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-abc123")
	require.Error(t, err)

	var mismatch CheckoutMarkerMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, "https://api.test.sageox.ai", mismatch.MarkerEndpoint)
	assert.Equal(t, "https://api.sageox.ai", mismatch.CurrentEndpoint)
}

func TestValidateLedgerMarker_RepoIDMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-abc123")
	require.NoError(t, err)

	err = ValidateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-xyz789")
	require.Error(t, err)

	var mismatch CheckoutMarkerMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, "repo-abc123", mismatch.MarkerRepoID)
	assert.Equal(t, "repo-xyz789", mismatch.CurrentRepoID)
}

func TestValidateLedgerMarker_NoMarker(t *testing.T) {
	tmpDir := t.TempDir()

	// no marker created, should pass validation
	err := ValidateLedgerMarker(tmpDir, "https://api.sageox.ai", "repo-abc123")
	assert.NoError(t, err)
}

func TestValidateTeamContextMarker_Match(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateTeamContextMarker(tmpDir, "https://api.sageox.ai", "team-123")
	require.NoError(t, err)

	err = ValidateTeamContextMarker(tmpDir, "https://api.sageox.ai")
	assert.NoError(t, err)
}

func TestValidateTeamContextMarker_EndpointMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateTeamContextMarker(tmpDir, "https://api.test.sageox.ai", "team-123")
	require.NoError(t, err)

	err = ValidateTeamContextMarker(tmpDir, "https://api.sageox.ai")
	require.Error(t, err)

	var mismatch CheckoutMarkerMismatch
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, "https://api.test.sageox.ai", mismatch.MarkerEndpoint)
	assert.Equal(t, "https://api.sageox.ai", mismatch.CurrentEndpoint)
}

func TestCheckoutMarkerMismatch_Error(t *testing.T) {
	t.Run("endpoint mismatch", func(t *testing.T) {
		err := CheckoutMarkerMismatch{
			CheckoutPath:    "/path/to/ledger",
			MarkerType:      "ledger",
			MarkerEndpoint:  "https://api.test.sageox.ai",
			CurrentEndpoint: "https://api.sageox.ai",
		}
		msg := err.Error()
		assert.Contains(t, msg, "ledger")
		assert.Contains(t, msg, "api.test.sageox.ai")
		assert.Contains(t, msg, "api.sageox.ai")
	})

	t.Run("repo id mismatch", func(t *testing.T) {
		err := CheckoutMarkerMismatch{
			CheckoutPath:    "/path/to/ledger",
			MarkerType:      "ledger",
			MarkerEndpoint:  "https://api.sageox.ai",
			MarkerRepoID:    "repo-old",
			CurrentEndpoint: "https://api.sageox.ai",
			CurrentRepoID:   "repo-new",
		}
		msg := err.Error()
		assert.Contains(t, msg, "repo-old")
		assert.Contains(t, msg, "repo-new")
	})
}

func TestSaveCheckoutMarker_EmptyPath(t *testing.T) {
	marker := &CheckoutMarker{Type: "ledger"}
	err := SaveCheckoutMarker("", marker)
	assert.Error(t, err)
}

func TestSaveCheckoutMarker_NilMarker(t *testing.T) {
	err := SaveCheckoutMarker("/tmp", nil)
	assert.Error(t, err)
}

func TestLoadCheckoutMarker_EmptyPath(t *testing.T) {
	_, err := LoadCheckoutMarker("")
	assert.Error(t, err)
}
