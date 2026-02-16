// internal/tips/proactive.go
package tips

import (
	"math/rand" //nolint:gosec // G404: tip selection doesn't need crypto-secure randomness
)

// ProactiveCheck represents a lightweight health check that can be run
// during tip display to surface actionable issues.
type ProactiveCheck struct {
	Name   string      // display name for debugging
	Weight int         // selection weight (higher = more likely to be selected)
	Check  func() bool // returns true if issue is detected
	FixTip string      // tip to show if issue detected (shown to both humans and agents)
	Prereq func() bool // optional prerequisite - only run check if this returns true
}

// proactiveChecks holds all registered lightweight checks
var proactiveChecks []ProactiveCheck

// RegisterProactiveCheck adds a check to the proactive check pool.
// Called from cmd/ox init functions to register doctor checks.
func RegisterProactiveCheck(check ProactiveCheck) {
	proactiveChecks = append(proactiveChecks, check)
}

// proactiveCheckChance is the probability of running a proactive check
// during tip display. Higher than regular tips because actionable fixes
// have more value than educational tips.
const proactiveCheckChance = 0.50

// RunProactiveCheck selects and runs a weighted random check.
// Returns (tip, true) if an issue is detected, ("", false) otherwise.
// Tips are shown to both humans and agents since they're actionable.
func RunProactiveCheck() (string, bool) {
	if len(proactiveChecks) == 0 {
		return "", false
	}

	// filter to eligible checks (prereq passes or no prereq)
	eligible := make([]ProactiveCheck, 0, len(proactiveChecks))
	for _, c := range proactiveChecks {
		if c.Prereq == nil || c.Prereq() {
			eligible = append(eligible, c)
		}
	}

	if len(eligible) == 0 {
		return "", false
	}

	// weighted random selection
	totalWeight := 0
	for _, c := range eligible {
		totalWeight += c.Weight
	}

	if totalWeight == 0 {
		return "", false
	}

	pick := rand.Intn(totalWeight)
	cumulative := 0
	for _, c := range eligible {
		cumulative += c.Weight
		if pick < cumulative {
			// run this check
			if c.Check() {
				return c.FixTip, true
			}
			return "", false
		}
	}

	return "", false
}

// MaybeRunProactiveCheck runs a check with the configured probability.
// Returns (tip, true) if an issue was detected and should be shown.
func MaybeRunProactiveCheck() (string, bool) {
	if rand.Float64() >= proactiveCheckChance {
		return "", false
	}
	return RunProactiveCheck()
}

// ClearProactiveChecks removes all registered checks (for testing)
func ClearProactiveChecks() {
	proactiveChecks = nil
}

// ProactiveCheckCount returns the number of registered checks (for testing)
func ProactiveCheckCount() int {
	return len(proactiveChecks)
}
