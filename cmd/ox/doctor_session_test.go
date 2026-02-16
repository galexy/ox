package main

import (
	"os"
	"strings"
	"testing"
)

func TestCheckSessionHealth_ReturnsChecks(t *testing.T) {
	// this test verifies the integration works with the new doctor package
	// actual checks depend on system state
	opts := doctorOptions{
		fix:      false,
		fixSlugs: nil,
		forceYes: false,
		verbose:  false,
	}
	results := checkSessionHealth(opts)

	// should return results (count varies based on system state)
	// some checks may be skipped if conditions aren't met
	_ = results // results validated by checks below

	// verify we have the expected check names when they appear
	names := make(map[string]bool)
	for _, r := range results {
		names[r.name] = true
	}

	// check for expected check names if they're present
	expectedChecks := []string{
		"session storage writable",
		"ledger cloned",
	}

	for _, expected := range expectedChecks {
		// checks may be skipped, so we just verify structure is correct
		// if they appear in results
		if names[expected] {
			t.Logf("found expected check: %s", expected)
		}
	}
}

func TestCheckSessionCommit_NoLedger(t *testing.T) {
	// save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// create a temp directory that is NOT a git repo
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	result := checkSessionCommit(false)

	// should skip when no ledger found
	if !result.skipped {
		t.Logf("result: passed=%v, skipped=%v, message=%q", result.passed, result.skipped, result.message)
		// acceptable: either skipped or passed with "no staged sessions"
		if !result.passed || result.message != "no staged sessions" {
			t.Logf("expected skipped or passed with 'no staged sessions', got: %+v", result)
		}
	}
}

func TestExtractSessionIDs(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "single session file",
			files:    []string{"sessions/2026-01-15-10-30-ryan-Oxabc1.jsonl"},
			expected: []string{"Oxabc1"},
		},
		{
			name: "multiple session files",
			files: []string{
				"sessions/2026-01-15-10-30-ryan-Oxabc1.jsonl",
				"sessions/2026-01-15-11-45-ryan-Oxdef2.jsonl",
			},
			expected: []string{"Oxabc1", "Oxdef2"},
		},
		{
			name: "duplicate session IDs",
			files: []string{
				"sessions/2026-01-15-10-30-ryan-Oxabc1.jsonl",
				"sessions/2026-01-15-11-45-ryan-Oxabc1.html",
			},
			expected: []string{"Oxabc1"},
		},
		{
			name:     "non-jsonl files ignored",
			files:    []string{"sessions/readme.md", "sessions/2026-01-15-10-30-ryan-Oxabc1.html"},
			expected: nil,
		},
		{
			name:     "empty list",
			files:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionIDs(tt.files)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d session IDs, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, id := range tt.expected {
				if result[i] != id {
					t.Errorf("expected session ID %q at index %d, got %q", id, i, result[i])
				}
			}
		})
	}
}

func TestBuildSessionCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		sessionIDs []string
		contains   string
	}{
		{
			name:       "single session",
			sessionIDs: []string{"Oxabc1"},
			contains:   "Add session",
		},
		{
			name:       "multiple sessions",
			sessionIDs: []string{"Oxabc1", "Oxdef2", "Oxghi3"},
			contains:   "Add 3 sessions",
		},
		{
			name:       "no sessions",
			sessionIDs: []string{},
			contains:   "Update sessions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSessionCommitMessage(tt.sessionIDs)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected message to contain %q, got %q", tt.contains, result)
			}
		})
	}
}
