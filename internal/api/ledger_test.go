package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetLedgerStatus tests - focused on real failure modes and API contract
// Tests removed as "test theater": TestGetLedgerStatus_EmptyResponseBody, TestGetLedgerStatus_NoAuthToken,
// TestGetLedgerStatus_UserAgentHeader, TestGetLedgerStatus_HandlesTrailingSlash, TestGetLedgerStatus_URLPath,
// TestGetLedgerStatus_MinimalResponse, TestGetLedgerStatus_WithMessage, TestGetLedgerStatus_HTTP500_EmptyBody,
// TestGetLedgerStatus_Timeout (tests Go stdlib, not our code)

func TestGetLedgerStatus_Success(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/api/v1/repos/repo_123/ledger-status"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "ready",
			"repo_url": "https://git.sageox.io/user/ledger.git",
			"repo_id": 42,
			"created_at": "2025-01-15T10:30:00Z"
		}`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "test-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "ready", resp.Status)
	assert.Equal(t, "https://git.sageox.io/user/ledger.git", resp.RepoURL)
	assert.Equal(t, 42, resp.RepoID)
}

func TestGetLedgerStatus_Pending(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "pending",
			"repo_url": "",
			"repo_id": 0,
			"created_at": "2025-01-15T10:30:00Z",
			"message": "Ledger is being provisioned, please wait..."
		}`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "test-token",
	}

	resp, err := client.GetLedgerStatus("repo_456")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "pending", resp.Status)
	assert.Empty(t, resp.RepoURL)
	assert.Equal(t, "Ledger is being provisioned, please wait...", resp.Message)
}

func TestGetLedgerStatus_NotFound(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"ledger not found"}`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "test-token",
	}

	resp, err := client.GetLedgerStatus("repo_nonexistent")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrLedgerNotFound), "expected ErrLedgerNotFound, got %v", err)
	assert.Contains(t, err.Error(), "repo_nonexistent")
}

func TestGetLedgerStatus_Unauthorized(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "expired-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrUnauthorized), "expected ErrUnauthorized, got %v", err)
}

func TestGetLedgerStatus_Forbidden(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"access denied"}`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "valid-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrForbidden), "expected ErrForbidden, got %v", err)
}

func TestGetLedgerStatus_ServerError(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "valid-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "500")
}

func TestGetLedgerStatus_NetworkError(t *testing.T) {
	t.Parallel()
	client := &RepoClient{
		baseURL:    "http://localhost:99999", // invalid port
		httpClient: &http.Client{Timeout: 1 * time.Second},
		version:    "test-version",
		authToken:  "valid-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "network error")
}

func TestGetLedgerStatus_MalformedJSON(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json {{{`))
	}))
	defer mockServer.Close()

	client := &RepoClient{
		baseURL:    mockServer.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		version:    "test-version",
		authToken:  "valid-token",
	}

	resp, err := client.GetLedgerStatus("repo_123")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "decode")
}
