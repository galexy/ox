package gitserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCheckoutPath(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // returns path to test
		wantErr   bool
		errTarget error
	}{
		{
			name: "non-existent path is valid",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-repo")
			},
			wantErr: false,
		},
		{
			name: "empty directory is valid",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "empty-dir")
				require.NoError(t, os.MkdirAll(dir, 0755))
				return dir
			},
			wantErr: false,
		},
		{
			name: "non-empty directory returns error",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "non-empty")
				require.NoError(t, os.MkdirAll(dir, 0755))
				// create a file inside
				require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644))
				return dir
			},
			wantErr:   true,
			errTarget: ErrPathExists,
		},
		{
			name: "file path returns error",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "file.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
				return filePath
			},
			wantErr:   true,
			errTarget: ErrPathExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			err := validateCheckoutPath(path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.True(t, errors.Is(err, tt.errTarget), "error = %v, want %v", err, tt.errTarget)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildAuthURL(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		creds   *GitCredentials
		wantURL string
		wantErr bool
	}{
		{
			name:    "https URL with credentials",
			repoURL: "https://git.example.com/user/repo.git",
			creds: &GitCredentials{
				Token:    "glpat-test-token",
				Username: "testuser",
			},
			wantURL: "https://oauth2:glpat-test-token@git.example.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "https URL without credentials",
			repoURL: "https://git.example.com/user/repo.git",
			creds:   nil,
			wantURL: "https://git.example.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "https URL with empty token",
			repoURL: "https://git.example.com/user/repo.git",
			creds: &GitCredentials{
				Token: "",
			},
			wantURL: "https://git.example.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "ssh URL unchanged",
			repoURL: "git@git.example.com:user/repo.git",
			creds: &GitCredentials{
				Token: "glpat-test-token",
			},
			wantURL: "git@git.example.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "URL with existing credentials",
			repoURL: "https://old:token@git.example.com/user/repo.git",
			creds: &GitCredentials{
				Token: "glpat-new-token",
			},
			wantURL: "https://oauth2:glpat-new-token@git.example.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "http localhost with credentials",
			repoURL: "http://localhost:8929/org/repo.git",
			creds: &GitCredentials{
				Token: "glpat-test-token",
			},
			wantURL: "http://oauth2:glpat-test-token@localhost:8929/org/repo.git",
			wantErr: false,
		},
		{
			name:    "http localhost without port",
			repoURL: "http://localhost/org/repo.git",
			creds: &GitCredentials{
				Token: "glpat-test-token",
			},
			wantURL: "http://oauth2:glpat-test-token@localhost/org/repo.git",
			wantErr: false,
		},
		{
			name:    "http 127.0.0.1 with credentials",
			repoURL: "http://127.0.0.1:8929/org/repo.git",
			creds: &GitCredentials{
				Token: "glpat-test-token",
			},
			wantURL: "http://oauth2:glpat-test-token@127.0.0.1:8929/org/repo.git",
			wantErr: false,
		},
		{
			name:    "http external host unchanged (security)",
			repoURL: "http://external-host.com/repo.git",
			creds: &GitCredentials{
				Token: "glpat-test-token",
			},
			wantURL: "http://external-host.com/repo.git",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAuthURL(tt.repoURL, tt.creds)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		wantIs bool
	}{
		{
			name:   "standard SSH URL",
			url:    "git@github.com:user/repo.git",
			wantIs: true,
		},
		{
			name:   "SSH URL with custom host",
			url:    "git@gitlab.example.com:group/repo.git",
			wantIs: true,
		},
		{
			name:   "HTTPS URL",
			url:    "https://github.com/user/repo.git",
			wantIs: false,
		},
		{
			name:   "HTTPS URL with credentials",
			url:    "https://user:token@github.com/user/repo.git",
			wantIs: false,
		},
		{
			name:   "file URL",
			url:    "file:///path/to/repo",
			wantIs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSSHURL(tt.url)
			assert.Equal(t, tt.wantIs, got)
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "URL with credentials",
			url:  "https://user:token@git.example.com/repo.git",
			want: "https://git.example.com/repo.git",
		},
		{
			name: "URL without credentials",
			url:  "https://git.example.com/repo.git",
			want: "https://git.example.com/repo.git",
		},
		{
			name: "SSH URL unchanged",
			url:  "git@github.com:user/repo.git",
			want: "git@github.com:user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultCheckoutPath(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		workDir  string
		want     string
	}{
		{
			name:     "with explicit workDir",
			repoName: "ledger",
			workDir:  "/home/user/projects/myrepo",
			want:     "/home/user/projects/ledger",
		},
		{
			name:     "team repo name",
			repoName: "team-acme-norms",
			workDir:  "/home/user/projects/myrepo",
			want:     "/home/user/projects/team-acme-norms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultCheckoutPath(tt.repoName, tt.workDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "HTTPS URL with .git suffix",
			url:  "https://git.example.com/user/my-repo.git",
			want: "my-repo",
		},
		{
			name: "HTTPS URL without .git suffix",
			url:  "https://git.example.com/user/my-repo",
			want: "my-repo",
		},
		{
			name: "SSH URL",
			url:  "git@github.com:user/my-repo.git",
			want: "my-repo",
		},
		{
			name: "SSH URL without .git",
			url:  "git@github.com:user/another-repo",
			want: "another-repo",
		},
		{
			name: "nested path",
			url:  "https://git.example.com/group/subgroup/my-repo.git",
			want: "my-repo",
		},
		{
			name: "empty string",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

// tests for deprecated functions (maintained for backwards compatibility)

func TestCheckoutTeamLedger_NoCredentials(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutTeamLedger(ctx, "team-123", "/tmp/test-team", nil)

	assert.True(t, errors.Is(err, ErrNoCredentials), "error = %v, want %v", err, ErrNoCredentials)
}

func TestCheckoutTeamLedger_RepoNotFound(t *testing.T) {
	tempDir := setupTestDir(t)

	// save credentials without team repo
	creds := GitCredentials{
		Token:     "test-token",
		ServerURL: "https://git.example.com",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Repos: map[string]RepoEntry{
			"team-alpha": {
				Name: "team-alpha",
				Type: "team-context",
				URL:  "https://git.example.com/teams/alpha.git",
			},
		},
	}
	require.NoError(t, SaveCredentialsForEndpoint("", creds))

	ctx := context.Background()
	checkoutPath := filepath.Join(tempDir, "team-checkout")
	err := CheckoutTeamLedger(ctx, "team-xyz", checkoutPath, nil)

	assert.True(t, errors.Is(err, ErrRepoNotFound), "error = %v, want %v", err, ErrRepoNotFound)
}

func TestCheckoutRepo_NoCredentials(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutRepo(ctx, "some-repo", "/tmp/test-repo", nil)

	assert.True(t, errors.Is(err, ErrNoCredentials), "error = %v, want %v", err, ErrNoCredentials)
}

func TestIsGitInstalled(t *testing.T) {
	// this should return true on most development machines
	// we don't fail if git is not installed since that's a valid state
	installed := IsGitInstalled()
	t.Logf("IsGitInstalled() = %v", installed)
}

func TestGetGitVersion(t *testing.T) {
	if !IsGitInstalled() {
		t.Skip("git not installed")
	}

	version, err := GetGitVersion()
	require.NoError(t, err)
	assert.NotEmpty(t, version)

	t.Logf("git version: %s", version)
}

func TestCheckoutTeamLedger_FindsByPartialMatch(t *testing.T) {
	tempDir := setupTestDir(t)

	// save credentials with team repo using different naming pattern
	creds := GitCredentials{
		Token:     "test-token",
		ServerURL: "https://git.example.com",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Repos: map[string]RepoEntry{
			"acme-corp-team-context": {
				Name: "acme-corp-team-context",
				Type: "team-context",
				URL:  "https://git.example.com/user/acme-corp-team-context.git",
			},
		},
	}
	require.NoError(t, SaveCredentialsForEndpoint("", creds))

	ctx := context.Background()
	checkoutPath := filepath.Join(tempDir, "team-checkout")

	// should find by partial match (acme-corp contains in key)
	err := CheckoutTeamLedger(ctx, "acme-corp", checkoutPath, nil)

	// will fail because clone won't work, but should NOT be ErrRepoNotFound
	assert.False(t, errors.Is(err, ErrRepoNotFound), "should have found repo by partial match, got %v", err)
}

func TestCheckoutOptions(t *testing.T) {
	// verify options struct works correctly
	opts := &CheckoutOptions{
		Depth:  1,
		Branch: "main",
	}

	assert.Equal(t, 1, opts.Depth)
	assert.Equal(t, "main", opts.Branch)
}

// tests for new URL-required functions

func TestCheckoutLedgerWithURL_EmptyURL(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutLedgerWithURL(ctx, "", "/tmp/test-ledger", nil)

	assert.True(t, errors.Is(err, ErrEmptyURL), "error = %v, want %v", err, ErrEmptyURL)
}

func TestCheckoutLedgerWithURL_NoCredentials(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutLedgerWithURL(ctx, "https://git.example.com/user/ledger.git", "/tmp/test-ledger", nil)

	assert.True(t, errors.Is(err, ErrNoCredentials), "error = %v, want %v", err, ErrNoCredentials)
}

func TestCheckoutTeamContextWithURL_EmptyURL(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutTeamContextWithURL(ctx, "team-123", "", "/tmp/test-team", nil)

	assert.True(t, errors.Is(err, ErrEmptyURL), "error = %v, want %v", err, ErrEmptyURL)
}

func TestCheckoutTeamContextWithURL_NoCredentials(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CheckoutTeamContextWithURL(ctx, "team-123", "https://git.example.com/team/context.git", "/tmp/test-team", nil)

	assert.True(t, errors.Is(err, ErrNoCredentials), "error = %v, want %v", err, ErrNoCredentials)
}

func TestCloneFromURL_EmptyURL(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CloneFromURL(ctx, "", "/tmp/test-repo", nil)

	assert.True(t, errors.Is(err, ErrEmptyURL), "error = %v, want %v", err, ErrEmptyURL)
}

func TestCloneFromURL_NoCredentials(t *testing.T) {
	setupTestDir(t)

	ctx := context.Background()
	err := CloneFromURL(ctx, "https://git.example.com/user/repo.git", "/tmp/test-repo", nil)

	assert.True(t, errors.Is(err, ErrNoCredentials), "error = %v, want %v", err, ErrNoCredentials)
}

func TestCheckoutLedgerWithURL_WithCredentials(t *testing.T) {
	tempDir := setupTestDir(t)

	// save credentials (needed for auth)
	creds := GitCredentials{
		Token:     "test-token",
		ServerURL: "https://git.example.com",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Repos:     map[string]RepoEntry{},
	}
	require.NoError(t, SaveCredentialsForEndpoint("", creds))

	ctx := context.Background()
	checkoutPath := filepath.Join(tempDir, "ledger-checkout")

	// this will fail at clone (no actual repo), but should pass URL and creds validation
	err := CheckoutLedgerWithURL(ctx, "https://git.example.com/user/ledger.git", checkoutPath, nil)

	// should fail with clone error, not URL or credentials error
	assert.False(t, errors.Is(err, ErrEmptyURL), "should not be ErrEmptyURL")
	assert.False(t, errors.Is(err, ErrNoCredentials), "should not be ErrNoCredentials")
	assert.True(t, errors.Is(err, ErrCloneFailed), "error = %v, want %v", err, ErrCloneFailed)
}

func TestCheckoutTeamContextWithURL_WithCredentials(t *testing.T) {
	tempDir := setupTestDir(t)

	// save credentials (needed for auth)
	creds := GitCredentials{
		Token:     "test-token",
		ServerURL: "https://git.example.com",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Repos:     map[string]RepoEntry{},
	}
	require.NoError(t, SaveCredentialsForEndpoint("", creds))

	ctx := context.Background()
	checkoutPath := filepath.Join(tempDir, "team-checkout")

	// this will fail at clone (no actual repo), but should pass URL and creds validation
	err := CheckoutTeamContextWithURL(ctx, "team-123", "https://git.example.com/team/context.git", checkoutPath, nil)

	// should fail with clone error, not URL or credentials error
	assert.False(t, errors.Is(err, ErrEmptyURL), "should not be ErrEmptyURL")
	assert.False(t, errors.Is(err, ErrNoCredentials), "should not be ErrNoCredentials")
	assert.True(t, errors.Is(err, ErrCloneFailed), "error = %v, want %v", err, ErrCloneFailed)
}
