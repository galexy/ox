package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	RegisterDoctorCheck(&DoctorCheck{
		Slug:        CheckSlugSessionCommit,
		Name:        "staged session commit",
		Category:    "Sessions",
		FixLevel:    FixLevelSuggested,
		Description: "Commits staged session files in the ledger repository",
		Run:         func(fix bool) checkResult { return checkSessionCommit(fix) },
	})
}

// checkSessionCommit checks for staged session files in the ledger and optionally commits them.
// This is a FixLevelSuggested check - it only performs the commit when fix=true.
func checkSessionCommit(fix bool) checkResult {
	ledgerPath := getLedgerPath()
	if ledgerPath == "" {
		return SkippedCheck("staged session commit", "no ledger found", "")
	}

	if !isGitRepo(ledgerPath) {
		return SkippedCheck("staged session commit", "ledger not a git repo", "")
	}

	// check for staged files in sessions/ directory
	stagedFiles, err := getStagedSessionFiles(ledgerPath)
	if err != nil {
		return SkippedCheck("staged session commit", "failed to check staged files", "")
	}

	if len(stagedFiles) == 0 {
		// no staged files - check passes (nothing to do)
		return PassedCheck("staged session commit", "no staged sessions")
	}

	// there are staged session files
	sessionCount := len(stagedFiles)
	sessionIDs := extractSessionIDs(stagedFiles)

	if !fix {
		// report that there are staged files that could be committed
		msg := fmt.Sprintf("%d staged session(s)", sessionCount)
		return WarningCheck("staged session commit", msg,
			fmt.Sprintf("Run `ox doctor --fix` to commit %d staged session file(s)", sessionCount))
	}

	// fix=true: commit the staged files
	commitMsg := buildSessionCommitMessage(sessionIDs)
	if err := commitStagedSessions(ledgerPath, commitMsg); err != nil {
		return FailedCheck("staged session commit",
			"commit failed",
			fmt.Sprintf("Error: %v", err))
	}

	return PassedCheck("staged session commit",
		fmt.Sprintf("committed %d session(s)", sessionCount))
}

// getStagedSessionFiles returns the list of staged files in the ledger's sessions/ directory.
func getStagedSessionFiles(ledgerPath string) ([]string, error) {
	// git diff --cached --name-only -- sessions/
	cmd := exec.Command("git", "-C", ledgerPath, "diff", "--cached", "--name-only", "--", "sessions/")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// extractSessionIDs extracts session identifiers from staged file paths.
// Session files have format: sessions/YYYY-MM-DD-HH-MM-username-sessionid.jsonl
func extractSessionIDs(files []string) []string {
	var sessionIDs []string
	seen := make(map[string]bool)

	for _, file := range files {
		base := filepath.Base(file)
		if !strings.HasSuffix(base, ".jsonl") {
			continue
		}

		// extract session ID from filename
		// format: YYYY-MM-DD-HH-MM-username-sessionid.jsonl
		parts := strings.Split(strings.TrimSuffix(base, ".jsonl"), "-")
		if len(parts) >= 7 {
			// session ID is typically the last part (e.g., "Oxa7b3")
			sessionID := parts[len(parts)-1]
			if !seen[sessionID] {
				seen[sessionID] = true
				sessionIDs = append(sessionIDs, sessionID)
			}
		}
	}
	return sessionIDs
}

// buildSessionCommitMessage creates a commit message for session files.
// Reuses the same format as cmd/ox/session_commit.go for consistency.
func buildSessionCommitMessage(sessionIDs []string) string {
	timestamp := time.Now().Format("2006-01-02")

	if len(sessionIDs) == 1 {
		return fmt.Sprintf("Add session %s-%s", timestamp, sessionIDs[0])
	}
	if len(sessionIDs) > 1 {
		return fmt.Sprintf("Add %d sessions %s", len(sessionIDs), timestamp)
	}
	return fmt.Sprintf("Update sessions %s", timestamp)
}

// commitStagedSessions commits the staged session files in the ledger.
func commitStagedSessions(ledgerPath, commitMsg string) error {
	cmd := exec.Command("git", "-C", ledgerPath, "commit", "--no-verify", "-m", commitMsg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// check if nothing to commit (shouldn't happen since we checked for staged files)
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit: %w: %s", err, string(output))
	}
	return nil
}
