package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sageox/ox/internal/endpoint"
)

// sageoxRemoteName is the expected git remote name for syncing .sageox
const sageoxRemoteName = "sageox"

// isSageoxGitRepo checks if .sageox directory is its own git repository.
// Returns true if .sageox/.git exists (either as directory or file for submodules).
func isSageoxGitRepo() bool {
	gitRoot := findGitRoot()
	if gitRoot == "" {
		return false
	}

	sageoxDir := filepath.Join(gitRoot, ".sageox")
	if _, err := os.Stat(sageoxDir); os.IsNotExist(err) {
		return false
	}

	// check for .git directory or file (submodules use a file pointing to parent's .git/modules)
	sageoxGitDir := filepath.Join(sageoxDir, ".git")
	_, err := os.Stat(sageoxGitDir)
	return err == nil
}

// checkSageoxRemote verifies that .sageox directory has a 'sageox' git remote.
// This check only applies when .sageox is its own git repository (submodule or separate repo).
// With fix=true, adds the 'sageox' remote pointing to the SageOx platform.
func checkSageoxRemote(fix bool) checkResult {
	gitRoot := findGitRoot()
	if gitRoot == "" {
		return SkippedCheck("sageox remote", "not in git repo", "")
	}

	sageoxDir := filepath.Join(gitRoot, ".sageox")
	if _, err := os.Stat(sageoxDir); os.IsNotExist(err) {
		return SkippedCheck("sageox remote", ".sageox/ missing", "")
	}

	// check if .sageox is its own git repository
	sageoxGitDir := filepath.Join(sageoxDir, ".git")
	if _, err := os.Stat(sageoxGitDir); os.IsNotExist(err) {
		// .sageox is not a separate git repo (part of parent repo)
		// this is the normal case, skip the check
		return SkippedCheck("sageox remote", ".sageox not a git repo", "")
	}

	// .sageox is a git repo, check for 'sageox' remote
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = sageoxDir
	output, err := cmd.Output()
	if err != nil {
		return WarningCheck("sageox remote", "failed to list remotes", err.Error())
	}

	// parse remotes to find 'sageox'
	if hasSageoxRemote(string(output)) {
		return PassedCheck("sageox remote", "configured")
	}

	// remote missing
	if fix {
		remoteURL := getSageoxRemoteURL()
		if remoteURL == "" {
			return WarningCheck("sageox remote", "missing",
				"Could not determine SageOx remote URL")
		}

		addCmd := exec.Command("git", "remote", "add", sageoxRemoteName, remoteURL)
		addCmd.Dir = sageoxDir
		if err := addCmd.Run(); err != nil {
			return FailedCheck("sageox remote", "add failed", err.Error())
		}
		return PassedCheck("sageox remote", fmt.Sprintf("added (%s)", remoteURL))
	}

	return WarningCheck("sageox remote", "missing",
		fmt.Sprintf("Run `ox doctor --fix` to add '%s' remote for .sageox sync", sageoxRemoteName))
}

// hasSageoxRemote checks if the git remote -v output contains a 'sageox' remote
func hasSageoxRemote(remoteOutput string) bool {
	lines := strings.Split(remoteOutput, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 && fields[0] == sageoxRemoteName {
			return true
		}
	}
	return false
}

// getSageoxRemoteURL returns the URL for the 'sageox' git remote.
// Uses the current SAGEOX_ENDPOINT to construct the git URL.
func getSageoxRemoteURL() string {
	ep := endpoint.Get()
	if ep == "" {
		return ""
	}

	// extract host from endpoint URL
	// e.g., "https://api.sageox.ai" -> "git@sageox.ai"
	host := strings.TrimPrefix(ep, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "api.")
	host = strings.Split(host, "/")[0] // remove any path
	host = strings.Split(host, ":")[0] // remove any port

	if host == "" {
		return ""
	}

	// construct git SSH URL for SageOx guidance sync
	// format: git@<host>:sageox/guidance.git
	return fmt.Sprintf("git@%s:sageox/guidance.git", host)
}
