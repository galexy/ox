//go:build !short

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectIntoFile_NewInjection(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "AGENTS.md")

	// create file without ox prime
	require.NoError(t, os.WriteFile(filePath, []byte("# Existing content\n"), 0644), "failed to write file")

	section := "\n" + OxPrimeLine + "\n"
	status, err := injectIntoFile(filePath, section, "AGENTS.md")
	require.NoError(t, err, "injectIntoFile failed")

	assert.Equal(t, injectedNew, status, "expected injectedNew")

	// verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err, "failed to read file")

	assert.Contains(t, string(content), OxPrimeLine, "expected OxPrimeLine to be present")
}

func TestInjectIntoFile_AlreadyPresent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "AGENTS.md")

	// create file with ox prime already present
	content := "# Existing content\n" + OxPrimeLine + "\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644), "failed to write file")

	section := "\n" + OxPrimeLine + "\n"
	status, err := injectIntoFile(filePath, section, "AGENTS.md")
	require.NoError(t, err, "injectIntoFile failed")

	assert.Equal(t, alreadyPresent, status, "expected alreadyPresent")
}

func TestInjectIntoFile_Upgrade(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "AGENTS.md")

	// create file with legacy reference
	legacyContent := "# Existing content\nRun ox prime at session start\n"
	require.NoError(t, os.WriteFile(filePath, []byte(legacyContent), 0644), "failed to write file")

	section := "\n" + OxPrimeLine + "\n"
	status, err := injectIntoFile(filePath, section, "AGENTS.md")
	require.NoError(t, err, "injectIntoFile failed")

	assert.Equal(t, injectedUpgrade, status, "expected injectedUpgrade")

	// verify new content has canonical line
	content, err := os.ReadFile(filePath)
	require.NoError(t, err, "failed to read file")

	assert.Contains(t, string(content), OxPrimeLine, "expected OxPrimeLine to be present after upgrade")
}

func TestInjectStatus_Values(t *testing.T) {
	// verify injectStatus constants have expected values
	assert.Equal(t, injectStatus(0), injectedNew, "injectedNew should be 0")
	assert.Equal(t, injectStatus(1), alreadyPresent, "alreadyPresent should be 1")
	assert.Equal(t, injectStatus(2), injectedUpgrade, "injectedUpgrade should be 2")
	assert.Equal(t, injectStatus(3), symlinkCreated, "symlinkCreated should be 3")
}

func TestInjectOxPrime_NeitherExists(t *testing.T) {
	tmpDir := t.TempDir()

	results, err := injectOxPrime(tmpDir)
	require.NoError(t, err, "injectOxPrime failed")

	// should create AGENTS.md + CLAUDE.md symlink
	require.GreaterOrEqual(t, len(results), 2, "expected AGENTS.md + CLAUDE.md results")

	agentsResult := results[0]
	assert.Equal(t, "AGENTS.md", agentsResult.file, "expected first result to be AGENTS.md")
	assert.Equal(t, injectedNew, agentsResult.status, "expected injectedNew")

	// verify AGENTS.md has both markers (header is PRIMARY mechanism for agent priming)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	require.FileExists(t, agentsPath, "AGENTS.md was not created")
	content, err := os.ReadFile(agentsPath)
	require.NoError(t, err, "failed to read AGENTS.md")
	assert.Contains(t, string(content), OxPrimeLine, "expected OxPrimeLine (footer)")
	assert.Contains(t, string(content), OxPrimeCheckMarker, "expected ox:prime-check (header)")

	// verify CLAUDE.md always created as symlink (agents read CLAUDE.md, so it must exist)
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	info, err := os.Lstat(claudePath)
	require.NoError(t, err, "CLAUDE.md was not created")
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "expected CLAUDE.md to be a symlink to AGENTS.md")
}

func TestInjectOxPrime_ClaudeExists(t *testing.T) {
	tmpDir := t.TempDir()

	// create CLAUDE.md
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	require.NoError(t, os.WriteFile(claudePath, []byte("# Existing CLAUDE.md\n"), 0644), "failed to create CLAUDE.md")

	results, err := injectOxPrime(tmpDir)
	require.NoError(t, err, "injectOxPrime failed")

	// should inject into CLAUDE.md and create symlink
	require.GreaterOrEqual(t, len(results), 1, "expected at least one result")

	// verify CLAUDE.md has both markers
	content, err := os.ReadFile(claudePath)
	require.NoError(t, err, "failed to read CLAUDE.md")

	assert.Contains(t, string(content), OxPrimeLine, "expected CLAUDE.md to contain OxPrimeLine (footer)")
	assert.Contains(t, string(content), OxPrimeCheckMarker, "expected CLAUDE.md to contain ox:prime-check (header)")

	// verify AGENTS.md symlink was created
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	info, err := os.Lstat(agentsPath)
	require.NoError(t, err, "failed to stat AGENTS.md")

	assert.True(t, info.Mode()&os.ModeSymlink != 0, "expected AGENTS.md to be a symlink")
}

func TestInjectOxPrime_BothExist(t *testing.T) {
	tmpDir := t.TempDir()

	// create both files
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	require.NoError(t, os.WriteFile(agentsPath, []byte("# AGENTS.md\n"), 0644), "failed to create AGENTS.md")

	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	require.NoError(t, os.WriteFile(claudePath, []byte("# CLAUDE.md\n"), 0644), "failed to create CLAUDE.md")

	results, err := injectOxPrime(tmpDir)
	require.NoError(t, err, "injectOxPrime failed")

	// should update both files
	require.Len(t, results, 2, "expected 2 results")

	// verify both contain both markers (header + footer)
	for _, path := range []string{agentsPath, claudePath} {
		content, err := os.ReadFile(path)
		require.NoError(t, err, "failed to read %s", path)

		assert.Contains(t, string(content), OxPrimeLine, "expected %s to contain OxPrimeLine (footer)", path)
		assert.Contains(t, string(content), OxPrimeCheckMarker, "expected %s to contain ox:prime-check (header)", path)
	}
}

func TestInjectOxPrime_WithClaudeCodeDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// create .claude directory to simulate Claude Code usage
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755), "failed to create .claude dir")

	results, err := injectOxPrime(tmpDir)
	require.NoError(t, err, "injectOxPrime failed")

	// should create AGENTS.md and symlink CLAUDE.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	require.FileExists(t, agentsPath, "AGENTS.md was not created")

	// verify AGENTS.md has both markers
	content, err := os.ReadFile(agentsPath)
	require.NoError(t, err, "failed to read AGENTS.md")
	assert.Contains(t, string(content), OxPrimeLine, "expected OxPrimeLine (footer)")
	assert.Contains(t, string(content), OxPrimeCheckMarker, "expected ox:prime-check (header)")

	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	info, err := os.Lstat(claudePath)
	require.NoError(t, err, "failed to stat CLAUDE.md")

	assert.True(t, info.Mode()&os.ModeSymlink != 0, "expected CLAUDE.md to be a symlink when .claude directory exists")

	// verify symlink target
	target, err := os.Readlink(claudePath)
	require.NoError(t, err, "failed to read symlink")

	assert.Equal(t, "AGENTS.md", target, "expected symlink target to be AGENTS.md")

	// check results include symlink creation
	hasSymlinkResult := false
	for _, r := range results {
		if r.file == "CLAUDE.md" && r.status == symlinkCreated {
			hasSymlinkResult = true
			break
		}
	}

	assert.True(t, hasSymlinkResult, "expected results to include CLAUDE.md symlink creation")
}
