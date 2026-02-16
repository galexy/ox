// internal/tips/tips.go
package tips

import (
	"math/rand" //nolint:gosec // G404: tip selection doesn't need crypto-secure randomness

	"github.com/sageox/ox/internal/cli"
)

// TriggerMode determines when tips should be shown
type TriggerMode int

const (
	// AlwaysShow displays a tip every time (for login, status, init)
	AlwaysShow TriggerMode = iota
	// WhenMinimal displays a tip when command output is short
	WhenMinimal
	// RandomChance displays a tip ~10% of the time
	RandomChance
)

const (
	discoveryChance = 0.20 // 20% chance to show discovery tip
	randomChance    = 0.10 // 10% chance for RandomChance mode
)

// ShouldShow determines if a tip should be displayed based on mode and flags
// Parameters: mode, quietMode, tipsDisabled, jsonMode
func ShouldShow(mode TriggerMode, quietMode, tipsDisabled, jsonMode bool) bool {
	// suppression checks
	if quietMode || tipsDisabled || jsonMode {
		return false
	}

	switch mode {
	case AlwaysShow:
		return true
	case WhenMinimal:
		return true // caller determines if output was minimal
	case RandomChance:
		return rand.Float64() < randomChance
	}

	return false
}

// GetTip returns a random tip for the given command.
// Automatically selects from human or agent tip pools based on command.
// Has 20% chance to return a discovery tip instead of contextual.
func GetTip(command string) string {
	contextual := getContextualTips(command)
	general := getGeneralTips(command)

	// roll for discovery
	if rand.Float64() < discoveryChance {
		return getRandomFromPool(general)
	}

	// try contextual tips
	if tips, ok := contextual[command]; ok && len(tips) > 0 {
		return getRandomFromPool(tips)
	}

	// fallback to general
	return getRandomFromPool(general)
}

func getRandomFromPool(pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[rand.Intn(len(pool))]
}

// MaybeShow conditionally displays a tip based on mode and flags.
// Proactive health check tips take priority over educational tips when an issue is detected.
// Parameters: command, mode, quietMode, tipsDisabled, jsonMode
func MaybeShow(command string, mode TriggerMode, quietMode, tipsDisabled, jsonMode bool) {
	if !ShouldShow(mode, quietMode, tipsDisabled, jsonMode) {
		return
	}

	// try proactive health check first (50% chance, weighted selection)
	// actionable fix tips trump educational tips when issue detected
	if proactiveTip, hasIssue := MaybeRunProactiveCheck(); hasIssue {
		cli.PrintTip(proactiveTip)
		return
	}

	// fall back to regular contextual/educational tip
	tip := GetTip(command)
	if tip != "" {
		cli.PrintTip(tip)
	}
}
