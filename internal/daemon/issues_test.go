package daemon

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ======
// IssueTracker Tests
// These tests verify the daemon's issue caching mechanism that signals
// when LLM reasoning is needed to resolve problems.
// ======

func TestIssueTracker_NeedsHelp_EmptyTracker(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	assert.False(t, tracker.NeedsHelp(), "empty tracker should not need help")
	assert.Equal(t, 0, tracker.Count())
	assert.Nil(t, tracker.GetIssues())
}

func TestIssueTracker_NeedsHelp_WithIssues(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Ledger has merge conflicts",
	})

	assert.True(t, tracker.NeedsHelp(), "tracker with issues should need help")
	assert.Equal(t, 1, tracker.Count())
}

func TestIssueTracker_Deduplication_SameTypeAndRepo(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Set initial issue
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Initial conflict",
	})

	// Set same (type, repo) again - should update, not duplicate
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityWarning, // severity changed
		Repo:     "ledger",
		Summary:  "Updated conflict",
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 1, "should deduplicate by (type, repo)")
	assert.Equal(t, SeverityWarning, issues[0].Severity, "should have updated severity")
	assert.Equal(t, "Updated conflict", issues[0].Summary, "should have updated summary")
}

func TestIssueTracker_Deduplication_DifferentRepos(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Same type, different repos - should NOT deduplicate
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Ledger conflict",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "team-context-abc",
		Summary:  "Team context conflict",
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 2, "different repos should not deduplicate")
}

func TestIssueTracker_Deduplication_DifferentTypes(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Same repo, different types - should NOT deduplicate
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Merge conflict",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMissingScaffolding,
		Severity: SeverityWarning,
		Repo:     "ledger",
		Summary:  "Missing AGENTS.md",
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 2, "different types should not deduplicate")
}

func TestIssueTracker_SeveritySorting(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Add in non-severity order
	tracker.SetIssue(DaemonIssue{
		Type:     "type1",
		Severity: SeverityWarning,
		Repo:     "repo1",
		Summary:  "Warning issue",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     "type2",
		Severity: SeverityCritical,
		Repo:     "repo2",
		Summary:  "Critical issue",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     "type3",
		Severity: SeverityError,
		Repo:     "repo3",
		Summary:  "Error issue",
	})

	issues := tracker.GetIssues()
	require.Len(t, issues, 3)

	// Should be sorted: critical, error, warning
	assert.Equal(t, SeverityCritical, issues[0].Severity, "critical should be first")
	assert.Equal(t, SeverityError, issues[1].Severity, "error should be second")
	assert.Equal(t, SeverityWarning, issues[2].Severity, "warning should be third")
}

func TestIssueTracker_ClearIssue(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Conflict",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMissingScaffolding,
		Severity: SeverityWarning,
		Repo:     "ledger",
		Summary:  "Missing file",
	})

	assert.Equal(t, 2, tracker.Count())

	// Clear only the merge conflict
	tracker.ClearIssue(IssueTypeMergeConflict, "ledger")

	assert.Equal(t, 1, tracker.Count())
	issues := tracker.GetIssues()
	assert.Equal(t, IssueTypeMissingScaffolding, issues[0].Type)
}

func TestIssueTracker_ClearIssue_NonExistent(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Conflict",
	})

	// Clear non-existent issue - should be no-op
	tracker.ClearIssue(IssueTypeMergeConflict, "other-repo")
	tracker.ClearIssue("non_existent_type", "ledger")

	assert.Equal(t, 1, tracker.Count(), "should not affect existing issues")
}

func TestIssueTracker_ClearRepo(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Add multiple issues for same repo
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Conflict",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMissingScaffolding,
		Severity: SeverityWarning,
		Repo:     "ledger",
		Summary:  "Missing file",
	})
	// Add issue for different repo
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "team-context-abc",
		Summary:  "Team conflict",
	})

	assert.Equal(t, 3, tracker.Count())

	// Clear all issues for ledger
	tracker.ClearRepo("ledger")

	assert.Equal(t, 1, tracker.Count())
	issues := tracker.GetIssues()
	assert.Equal(t, "team-context-abc", issues[0].Repo)
}

func TestIssueTracker_Clear(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	tracker.SetIssue(DaemonIssue{Type: "type1", Severity: SeverityError, Repo: "repo1"})
	tracker.SetIssue(DaemonIssue{Type: "type2", Severity: SeverityWarning, Repo: "repo2"})

	assert.Equal(t, 2, tracker.Count())

	tracker.Clear()

	assert.Equal(t, 0, tracker.Count())
	assert.False(t, tracker.NeedsHelp())
}

func TestIssueTracker_GlobalIssue_EmptyRepo(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Global issue (empty repo) - like auth expiring
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeAuthExpiring,
		Severity: SeverityWarning,
		Repo:     "", // global
		Summary:  "Auth token expires in 2 hours",
	})

	assert.True(t, tracker.NeedsHelp())
	issues := tracker.GetIssues()
	assert.Len(t, issues, 1)
	assert.Empty(t, issues[0].Repo)
}

func TestIssueTracker_GlobalIssue_Deduplication(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Set global auth issue twice
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeAuthExpiring,
		Severity: SeverityWarning,
		Repo:     "",
		Summary:  "Expires in 2 hours",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeAuthExpiring,
		Severity: SeverityError, // escalated
		Repo:     "",
		Summary:  "Expires in 30 minutes",
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 1, "global issues should deduplicate too")
	assert.Equal(t, SeverityError, issues[0].Severity)
}

func TestIssueTracker_MaxSeverity(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Empty tracker
	assert.Empty(t, tracker.MaxSeverity())

	// Add warning
	tracker.SetIssue(DaemonIssue{Type: "type1", Severity: SeverityWarning, Repo: "repo1"})
	assert.Equal(t, SeverityWarning, tracker.MaxSeverity())

	// Add error
	tracker.SetIssue(DaemonIssue{Type: "type2", Severity: SeverityError, Repo: "repo2"})
	assert.Equal(t, SeverityError, tracker.MaxSeverity())

	// Add critical
	tracker.SetIssue(DaemonIssue{Type: "type3", Severity: SeverityCritical, Repo: "repo3"})
	assert.Equal(t, SeverityCritical, tracker.MaxSeverity())

	// Remove critical
	tracker.ClearIssue("type3", "repo3")
	assert.Equal(t, SeverityError, tracker.MaxSeverity())
}

func TestIssueTracker_SinceTime_Preserved(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	originalTime := time.Now().Add(-1 * time.Hour)

	// Set with explicit Since time
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Original",
		Since:    originalTime,
	})

	// Update without Since - should preserve original
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityWarning,
		Repo:     "ledger",
		Summary:  "Updated",
		// Since not set
	})

	issues := tracker.GetIssues()
	assert.Equal(t, originalTime, issues[0].Since, "should preserve original Since time")
}

func TestIssueTracker_SinceTime_AutoSet(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	before := time.Now()

	// Set without Since - should auto-set
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Conflict",
	})

	after := time.Now()

	issues := tracker.GetIssues()
	assert.False(t, issues[0].Since.IsZero(), "Since should be auto-set")
	assert.True(t, issues[0].Since.After(before) || issues[0].Since.Equal(before))
	assert.True(t, issues[0].Since.Before(after) || issues[0].Since.Equal(after))
}

func TestIssueTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Concurrent writers and readers
	var wg sync.WaitGroup
	const numWriters = 10
	const numReaders = 20
	const iterations = 100

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tracker.SetIssue(DaemonIssue{
					Type:     IssueTypeMergeConflict,
					Severity: SeverityError,
					Repo:     "ledger", // all write to same key - tests dedup under contention
					Summary:  "Conflict",
				})
				tracker.ClearIssue(IssueTypeMergeConflict, "ledger")
			}
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = tracker.NeedsHelp()
				_ = tracker.GetIssues()
				_ = tracker.MaxSeverity()
				_ = tracker.Count()
			}
		}()
	}

	wg.Wait()
	// Test passes if no race detected (run with -race flag)
}

func TestIssueTracker_RapidSetClear(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Rapid set/clear cycles - should not leak or corrupt
	for i := 0; i < 1000; i++ {
		tracker.SetIssue(DaemonIssue{
			Type:     IssueTypeMergeConflict,
			Severity: SeverityError,
			Repo:     "ledger",
			Summary:  "Conflict",
		})
		tracker.ClearIssue(IssueTypeMergeConflict, "ledger")
	}

	assert.Equal(t, 0, tracker.Count(), "should be empty after set/clear cycles")
	assert.False(t, tracker.NeedsHelp())
}

func TestIssueTracker_GetIssues_ReturnsCopy(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Conflict",
	})

	// Get issues and modify the returned slice
	issues1 := tracker.GetIssues()
	issues1[0].Summary = "Modified"

	// Get again - should not reflect modification
	issues2 := tracker.GetIssues()
	assert.NotEqual(t, "Modified", issues2[0].Summary, "GetIssues should return a copy")
}

func TestSeverityRank(t *testing.T) {
	t.Parallel()

	assert.Greater(t, severityRank(SeverityCritical), severityRank(SeverityError))
	assert.Greater(t, severityRank(SeverityError), severityRank(SeverityWarning))
	assert.Greater(t, severityRank(SeverityWarning), severityRank("unknown"))
	assert.Equal(t, 0, severityRank(""))
	assert.Equal(t, 0, severityRank("info")) // info doesn't exist, should be lowest
}

// ======
// Integration-style tests for realistic scenarios
// ======

func TestIssueTracker_RealisticScenario_MergeConflictFlow(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// 1. Sync detects merge conflict
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Ledger has merge conflicts",
	})

	assert.True(t, tracker.NeedsHelp())
	assert.Equal(t, SeverityError, tracker.MaxSeverity())

	// 2. LLM resolves conflict, sync succeeds
	tracker.ClearIssue(IssueTypeMergeConflict, "ledger")

	assert.False(t, tracker.NeedsHelp())
	assert.Empty(t, tracker.MaxSeverity())
}

func TestIssueTracker_RealisticScenario_MultipleReposWithIssues(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Multiple repos with various issues
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMergeConflict,
		Severity: SeverityError,
		Repo:     "ledger",
		Summary:  "Ledger conflict",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeMissingScaffolding,
		Severity: SeverityWarning,
		Repo:     "team-context-frontend",
		Summary:  "Missing AGENTS.md",
	})
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeDiverged,
		Severity: SeverityCritical,
		Repo:     "team-context-backend",
		Summary:  "History diverged",
	})

	// Check state
	assert.True(t, tracker.NeedsHelp())
	assert.Equal(t, 3, tracker.Count())
	assert.Equal(t, SeverityCritical, tracker.MaxSeverity())

	// Issues should be sorted by severity
	issues := tracker.GetIssues()
	assert.Equal(t, SeverityCritical, issues[0].Severity)
	assert.Equal(t, SeverityError, issues[1].Severity)
	assert.Equal(t, SeverityWarning, issues[2].Severity)

	// Fix critical issue
	tracker.ClearIssue(IssueTypeDiverged, "team-context-backend")
	assert.Equal(t, SeverityError, tracker.MaxSeverity())

	// Fix error issue
	tracker.ClearIssue(IssueTypeMergeConflict, "ledger")
	assert.Equal(t, SeverityWarning, tracker.MaxSeverity())

	// Fix warning issue
	tracker.ClearIssue(IssueTypeMissingScaffolding, "team-context-frontend")
	assert.False(t, tracker.NeedsHelp())
}

func TestIssueTracker_RealisticScenario_SeverityEscalation(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Auth starts as warning (expires in 24h)
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeAuthExpiring,
		Severity: SeverityWarning,
		Repo:     "",
		Summary:  "Auth expires in 24 hours",
	})

	assert.Equal(t, SeverityWarning, tracker.MaxSeverity())

	// Later, escalates to error (expires in 1h)
	tracker.SetIssue(DaemonIssue{
		Type:     IssueTypeAuthExpiring,
		Severity: SeverityError,
		Repo:     "",
		Summary:  "Auth expires in 1 hour",
	})

	// Should be deduplicated and escalated
	assert.Equal(t, 1, tracker.Count())
	assert.Equal(t, SeverityError, tracker.MaxSeverity())
	assert.Equal(t, "Auth expires in 1 hour", tracker.GetIssues()[0].Summary)
}

func TestIssueTracker_RequiresConfirm(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Merge conflicts require confirmation (human approval for resolution)
	tracker.SetIssue(DaemonIssue{
		Type:            IssueTypeMergeConflict,
		Severity:        SeverityError,
		Repo:            "ledger",
		Summary:         "Ledger has merge conflicts",
		RequiresConfirm: true,
	})

	// Auth expiring does not require confirmation (can auto-refresh)
	tracker.SetIssue(DaemonIssue{
		Type:            IssueTypeAuthExpiring,
		Severity:        SeverityWarning,
		Repo:            "",
		Summary:         "Auth expires soon",
		RequiresConfirm: false,
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 2)

	// Find merge conflict issue (should be first due to higher severity)
	var mergeIssue, authIssue *DaemonIssue
	for i := range issues {
		if issues[i].Type == IssueTypeMergeConflict {
			mergeIssue = &issues[i]
		}
		if issues[i].Type == IssueTypeAuthExpiring {
			authIssue = &issues[i]
		}
	}

	require.NotNil(t, mergeIssue)
	require.NotNil(t, authIssue)

	assert.True(t, mergeIssue.RequiresConfirm, "merge conflict should require confirmation")
	assert.False(t, authIssue.RequiresConfirm, "auth expiring should not require confirmation")
}

func TestIssueTracker_RequiresConfirm_PreservedOnUpdate(t *testing.T) {
	t.Parallel()

	tracker := NewIssueTracker()

	// Set issue with RequiresConfirm
	tracker.SetIssue(DaemonIssue{
		Type:            IssueTypeMergeConflict,
		Severity:        SeverityError,
		Repo:            "ledger",
		Summary:         "Initial conflict",
		RequiresConfirm: true,
	})

	// Update same issue - RequiresConfirm should be updated to new value
	tracker.SetIssue(DaemonIssue{
		Type:            IssueTypeMergeConflict,
		Severity:        SeverityWarning, // changed
		Repo:            "ledger",
		Summary:         "Updated conflict",
		RequiresConfirm: false, // changed
	})

	issues := tracker.GetIssues()
	assert.Len(t, issues, 1)
	assert.False(t, issues[0].RequiresConfirm, "RequiresConfirm should be updated")
	assert.Equal(t, SeverityWarning, issues[0].Severity)
}

// ======
// Helper function tests
// ======

func TestDaemonIssue_FormatLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		issue           DaemonIssue
		includeSeverity bool
		expected        string
	}{
		{
			name: "basic issue with severity",
			issue: DaemonIssue{
				Severity: SeverityError,
				Repo:     "ledger",
				Summary:  "Has merge conflicts",
			},
			includeSeverity: true,
			expected:        "[error] ledger: Has merge conflicts",
		},
		{
			name: "basic issue without severity",
			issue: DaemonIssue{
				Severity: SeverityWarning,
				Repo:     "ledger",
				Summary:  "Has merge conflicts",
			},
			includeSeverity: false,
			expected:        "ledger: Has merge conflicts",
		},
		{
			name: "issue with confirm required",
			issue: DaemonIssue{
				Severity:        SeverityError,
				Repo:            "ledger",
				Summary:         "Has merge conflicts",
				RequiresConfirm: true,
			},
			includeSeverity: true,
			expected:        "[error] ledger: Has merge conflicts [CONFIRM REQUIRED]",
		},
		{
			name: "global issue (no repo)",
			issue: DaemonIssue{
				Severity: SeverityWarning,
				Repo:     "",
				Summary:  "Auth expires soon",
			},
			includeSeverity: true,
			expected:        "[warning] Auth expires soon",
		},
		{
			name: "global issue with confirm required",
			issue: DaemonIssue{
				Severity:        SeverityCritical,
				Repo:            "",
				Summary:         "Re-authentication needed",
				RequiresConfirm: true,
			},
			includeSeverity: false,
			expected:        "Re-authentication needed [CONFIRM REQUIRED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.FormatLine(tt.includeSeverity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasConfirmRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		issues   []DaemonIssue
		expected bool
	}{
		{
			name:     "empty slice",
			issues:   []DaemonIssue{},
			expected: false,
		},
		{
			name:     "nil slice",
			issues:   nil,
			expected: false,
		},
		{
			name: "no confirm required",
			issues: []DaemonIssue{
				{Type: "type1", RequiresConfirm: false},
				{Type: "type2", RequiresConfirm: false},
			},
			expected: false,
		},
		{
			name: "one confirm required",
			issues: []DaemonIssue{
				{Type: "type1", RequiresConfirm: false},
				{Type: "type2", RequiresConfirm: true},
			},
			expected: true,
		},
		{
			name: "all confirm required",
			issues: []DaemonIssue{
				{Type: "type1", RequiresConfirm: true},
				{Type: "type2", RequiresConfirm: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasConfirmRequired(tt.issues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxIssueSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		issues   []DaemonIssue
		expected string
	}{
		{
			name:     "empty slice",
			issues:   []DaemonIssue{},
			expected: "",
		},
		{
			name:     "nil slice",
			issues:   nil,
			expected: "",
		},
		{
			name: "single warning",
			issues: []DaemonIssue{
				{Severity: SeverityWarning},
			},
			expected: SeverityWarning,
		},
		{
			name: "warning and error",
			issues: []DaemonIssue{
				{Severity: SeverityWarning},
				{Severity: SeverityError},
			},
			expected: SeverityError,
		},
		{
			name: "all severities",
			issues: []DaemonIssue{
				{Severity: SeverityWarning},
				{Severity: SeverityCritical},
				{Severity: SeverityError},
			},
			expected: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxIssueSeverity(tt.issues)
			assert.Equal(t, tt.expected, result)
		})
	}
}
