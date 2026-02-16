package tips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterProactiveCheck(t *testing.T) {
	// clear any existing checks
	ClearProactiveChecks()

	// register a test check
	RegisterProactiveCheck(ProactiveCheck{
		Name:   "test check",
		Weight: 10,
		Check:  func() bool { return true },
		FixTip: "Test tip",
	})

	assert.Equal(t, 1, ProactiveCheckCount())

	// cleanup
	ClearProactiveChecks()
}

func TestRunProactiveCheck_NoChecks(t *testing.T) {
	ClearProactiveChecks()

	tip, hasIssue := RunProactiveCheck()
	assert.False(t, hasIssue, "expected no issue with no checks registered")
	assert.Empty(t, tip, "expected empty tip")
}

func TestRunProactiveCheck_IssueDetected(t *testing.T) {
	ClearProactiveChecks()

	RegisterProactiveCheck(ProactiveCheck{
		Name:   "always fails",
		Weight: 10,
		Check:  func() bool { return true }, // issue detected
		FixTip: "Fix this issue",
	})

	tip, hasIssue := RunProactiveCheck()
	assert.True(t, hasIssue, "expected issue to be detected")
	assert.Equal(t, "Fix this issue", tip)

	ClearProactiveChecks()
}

func TestRunProactiveCheck_NoIssue(t *testing.T) {
	ClearProactiveChecks()

	RegisterProactiveCheck(ProactiveCheck{
		Name:   "always passes",
		Weight: 10,
		Check:  func() bool { return false }, // no issue
		FixTip: "Fix this issue",
	})

	tip, hasIssue := RunProactiveCheck()
	assert.False(t, hasIssue, "expected no issue")
	assert.Empty(t, tip)

	ClearProactiveChecks()
}

func TestRunProactiveCheck_PrereqFails(t *testing.T) {
	ClearProactiveChecks()

	RegisterProactiveCheck(ProactiveCheck{
		Name:   "prereq fails",
		Weight: 10,
		Prereq: func() bool { return false }, // prereq fails
		Check:  func() bool { return true },  // would detect issue
		FixTip: "Should not see this",
	})

	tip, hasIssue := RunProactiveCheck()
	assert.False(t, hasIssue, "expected no issue when prereq fails")
	assert.Empty(t, tip, "expected empty tip when prereq fails")

	ClearProactiveChecks()
}

func TestRunProactiveCheck_WeightedSelection(t *testing.T) {
	ClearProactiveChecks()

	// register two checks with different weights
	// high weight check passes, low weight check fails
	RegisterProactiveCheck(ProactiveCheck{
		Name:   "high weight passes",
		Weight: 100,
		Check:  func() bool { return false }, // no issue
		FixTip: "High weight tip",
	})
	RegisterProactiveCheck(ProactiveCheck{
		Name:   "low weight fails",
		Weight: 1,
		Check:  func() bool { return true }, // issue detected
		FixTip: "Low weight tip",
	})

	// run many times - high weight should be selected most often
	// but when low weight is selected, it should show its tip
	foundLowWeightTip := false
	for i := 0; i < 1000; i++ {
		tip, hasIssue := RunProactiveCheck()
		if hasIssue && tip == "Low weight tip" {
			foundLowWeightTip = true
			break
		}
	}

	assert.True(t, foundLowWeightTip, "low weight check should eventually be selected")

	ClearProactiveChecks()
}
