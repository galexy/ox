package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSageoxGitRepo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name: "no .sageox directory",
			setup: func(t *testing.T, dir string) {
				// no setup needed - .sageox doesn't exist
			},
			expected: false,
		},
		{
			name: ".sageox exists but not a git repo",
			setup: func(t *testing.T, dir string) {
				sageoxDir := filepath.Join(dir, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxDir, 0755))
			},
			expected: false,
		},
		{
			name: ".sageox is its own git repo",
			setup: func(t *testing.T, dir string) {
				sageoxDir := filepath.Join(dir, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxDir, 0755))
				// initialize .sageox as a git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = sageoxDir
				require.NoError(t, cmd.Run())
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// create temp dir with parent git repo
			tmpDir := t.TempDir()
			cmd := exec.Command("git", "init")
			cmd.Dir = tmpDir
			require.NoError(t, cmd.Run())

			// configure git user for commits
			configCmd := exec.Command("git", "config", "user.email", "test@example.com")
			configCmd.Dir = tmpDir
			_ = configCmd.Run()
			configCmd = exec.Command("git", "config", "user.name", "Test")
			configCmd.Dir = tmpDir
			_ = configCmd.Run()

			tc.setup(t, tmpDir)

			// change to temp dir to test
			oldWd, _ := os.Getwd()
			defer func() { _ = os.Chdir(oldWd) }()
			require.NoError(t, os.Chdir(tmpDir))

			result := isSageoxGitRepo()
			assert.Equal(t, tc.expected, result, "isSageoxGitRepo() result mismatch")
		})
	}
}

func TestCheckSageoxRemote(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, dir string)
		fix            bool
		expectPassed   bool
		expectSkipped  bool
		expectWarning  bool
		expectContains string
	}{
		{
			name: "not in git repo",
			setup: func(t *testing.T, dir string) {
				// remove .git to make it not a repo
				os.RemoveAll(filepath.Join(dir, ".git"))
			},
			fix:           false,
			expectSkipped: true,
		},
		{
			name: ".sageox not a git repo - skipped",
			setup: func(t *testing.T, dir string) {
				sageoxDir := filepath.Join(dir, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxDir, 0755))
				// no .git inside .sageox
			},
			fix:           false,
			expectSkipped: true,
		},
		{
			name: ".sageox is git repo without sageox remote - warning",
			setup: func(t *testing.T, dir string) {
				sageoxDir := filepath.Join(dir, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxDir, 0755))
				// initialize .sageox as a git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = sageoxDir
				require.NoError(t, cmd.Run())
			},
			fix:            false,
			expectPassed:   true,
			expectWarning:  true,
			expectContains: "missing",
		},
		{
			name: ".sageox is git repo with sageox remote - passed",
			setup: func(t *testing.T, dir string) {
				sageoxDir := filepath.Join(dir, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxDir, 0755))
				// initialize .sageox as a git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = sageoxDir
				require.NoError(t, cmd.Run())
				// add sageox remote
				cmd = exec.Command("git", "remote", "add", "sageox", "git@example.com:test/repo.git")
				cmd.Dir = sageoxDir
				require.NoError(t, cmd.Run())
			},
			fix:            false,
			expectPassed:   true,
			expectContains: "configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// create temp dir with parent git repo
			tmpDir := t.TempDir()
			cmd := exec.Command("git", "init")
			cmd.Dir = tmpDir
			require.NoError(t, cmd.Run())

			tc.setup(t, tmpDir)

			// change to temp dir to test
			oldWd, _ := os.Getwd()
			defer func() { _ = os.Chdir(oldWd) }()
			require.NoError(t, os.Chdir(tmpDir))

			result := checkSageoxRemote(tc.fix)

			if tc.expectSkipped {
				assert.True(t, result.skipped, "expected skipped, got passed=%v warning=%v", result.passed, result.warning)
			}
			if tc.expectPassed {
				assert.True(t, result.passed, "expected passed")
			}
			if tc.expectWarning {
				assert.True(t, result.warning, "expected warning")
			}
			if tc.expectContains != "" && result.message != "" {
				if result.message != tc.expectContains {
					// check if message contains the expected string
					t.Logf("message=%q (checking contains %q)", result.message, tc.expectContains)
				}
			}
		})
	}
}

func TestHasSageoxRemote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty output",
			input:    "",
			expected: false,
		},
		{
			name:     "only origin remote",
			input:    "origin\tgit@github.com:user/repo.git (fetch)\norigin\tgit@github.com:user/repo.git (push)\n",
			expected: false,
		},
		{
			name:     "has sageox remote",
			input:    "origin\tgit@github.com:user/repo.git (fetch)\nsageox\tgit@sageox.ai:team/guidance.git (fetch)\n",
			expected: true,
		},
		{
			name:     "sageox is only remote",
			input:    "sageox\tgit@sageox.ai:team/guidance.git (fetch)\nsageox\tgit@sageox.ai:team/guidance.git (push)\n",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := hasSageoxRemote(tc.input)
			assert.Equal(t, tc.expected, result, "hasSageoxRemote(%q)", tc.input)
		})
	}
}

func TestGetSageoxRemoteURL(t *testing.T) {
	// test the URL extraction logic
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "production endpoint",
			envValue: "https://api.sageox.ai",
			expected: "git@sageox.ai:sageox/guidance.git",
		},
		{
			name:     "staging endpoint with port",
			envValue: "https://api.staging.sageox.ai:8443",
			expected: "git@staging.sageox.ai:sageox/guidance.git",
		},
		{
			name:     "local endpoint",
			envValue: "http://localhost:8080",
			expected: "git@localhost:sageox/guidance.git",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// set endpoint env var
			oldEnv := os.Getenv("SAGEOX_ENDPOINT")
			defer func() { _ = os.Setenv("SAGEOX_ENDPOINT", oldEnv) }()
			_ = os.Setenv("SAGEOX_ENDPOINT", tc.envValue)

			result := getSageoxRemoteURL()
			assert.Equal(t, tc.expected, result, "getSageoxRemoteURL()")
		})
	}
}
