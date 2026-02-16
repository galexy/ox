package session

import (
	"regexp"
	"strings"
)

// PlanMarker represents a type of plan marker found in content.
type PlanMarker string

const (
	// PlanMarkerNone indicates no plan markers were found
	PlanMarkerNone PlanMarker = ""

	// PlanMarkerPlan indicates a "## Plan" header was found
	PlanMarkerPlan PlanMarker = "plan"

	// PlanMarkerImplementationPlan indicates a "## Implementation Plan" header was found
	PlanMarkerImplementationPlan PlanMarker = "implementation_plan"

	// PlanMarkerFinalPlan indicates a "## Final Plan" header was found
	PlanMarkerFinalPlan PlanMarker = "final_plan"

	// PlanMarkerIsPlan indicates the entry has is_plan:true metadata
	PlanMarkerIsPlan PlanMarker = "is_plan"
)

// planHeaderPatterns maps regex patterns to their corresponding marker types.
// Patterns are checked in order; more specific patterns should come first.
var planHeaderPatterns = []struct {
	pattern *regexp.Regexp
	marker  PlanMarker
}{
	// final plan should match before generic plan
	{regexp.MustCompile(`(?i)^##\s+final\s+plan\b`), PlanMarkerFinalPlan},
	// implementation plan should match before generic plan
	{regexp.MustCompile(`(?i)^##\s+implementation\s+plan\b`), PlanMarkerImplementationPlan},
	// generic plan header
	{regexp.MustCompile(`(?i)^##\s+plan\b`), PlanMarkerPlan},
}

// PlanEntry represents an entry that contains plan content.
type PlanEntry struct {
	// Entry is the original session entry
	Entry SessionEntry

	// Marker indicates which plan marker was detected
	Marker PlanMarker

	// IsPlanFlag indicates whether the entry had is_plan:true metadata
	IsPlanFlag bool
}

// ExtractPlan finds the plan from a slice of session entries.
// Priority order:
//  1. Entry with is_plan:true metadata (highest priority)
//  2. Entry with "## Final Plan" header
//  3. Entry with "## Implementation Plan" header
//  4. Entry with "## Plan" header
//  5. Last assistant message (fallback)
//
// If multiple entries match the same marker type, the last one is returned.
// Returns nil if entries is empty or no suitable content is found.
func ExtractPlan(entries []SessionEntry) *PlanEntry {
	if len(entries) == 0 {
		return nil
	}

	var (
		lastIsPlan           *PlanEntry
		lastFinalPlan        *PlanEntry
		lastImplementation   *PlanEntry
		lastPlan             *PlanEntry
		lastAssistantMessage *PlanEntry
	)

	for i := range entries {
		entry := &entries[i]

		// only consider assistant messages for plan content
		if entry.Type != SessionEntryTypeAssistant {
			continue
		}

		// track last assistant message as fallback
		lastAssistantMessage = &PlanEntry{
			Entry:  *entry,
			Marker: PlanMarkerNone,
		}

		// check for plan markers in content
		marker := DetectPlanMarkers(entry.Content)

		switch marker {
		case PlanMarkerFinalPlan:
			lastFinalPlan = &PlanEntry{
				Entry:  *entry,
				Marker: marker,
			}
		case PlanMarkerImplementationPlan:
			lastImplementation = &PlanEntry{
				Entry:  *entry,
				Marker: marker,
			}
		case PlanMarkerPlan:
			lastPlan = &PlanEntry{
				Entry:  *entry,
				Marker: marker,
			}
		}
	}

	// check for is_plan metadata marker (simulated through content check)
	// in real usage, this would be checked via entry metadata
	for i := range entries {
		entry := &entries[i]
		if entry.Type == SessionEntryTypeAssistant && hasIsPlanMarker(entry.Content) {
			lastIsPlan = &PlanEntry{
				Entry:      *entry,
				Marker:     PlanMarkerIsPlan,
				IsPlanFlag: true,
			}
		}
	}

	// return in priority order
	if lastIsPlan != nil {
		return lastIsPlan
	}
	if lastFinalPlan != nil {
		return lastFinalPlan
	}
	if lastImplementation != nil {
		return lastImplementation
	}
	if lastPlan != nil {
		return lastPlan
	}

	return lastAssistantMessage
}

// hasIsPlanMarker checks if content contains the is_plan:true marker.
// This is typically embedded in the content as a comment or metadata indicator.
func hasIsPlanMarker(content string) bool {
	// check for common patterns indicating is_plan metadata
	// these patterns allow for variations in formatting
	patterns := []string{
		"is_plan:true",
		"is_plan: true",
		`"is_plan":true`,
		`"is_plan": true`,
	}

	contentLower := strings.ToLower(content)
	for _, p := range patterns {
		if strings.Contains(contentLower, p) {
			return true
		}
	}
	return false
}

// DetectPlanMarkers scans content for plan header markers.
// Returns the marker type found, or PlanMarkerNone if no markers detected.
// Checks line by line to find headers at the start of lines.
func DetectPlanMarkers(content string) PlanMarker {
	if content == "" {
		return PlanMarkerNone
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, ph := range planHeaderPatterns {
			if ph.pattern.MatchString(trimmed) {
				return ph.marker
			}
		}
	}

	return PlanMarkerNone
}

// ExtractMermaidDiagrams finds all mermaid code blocks in content.
// Returns a slice of diagram contents (without the ```mermaid markers).
// Handles multiple diagrams and ignores malformed/unclosed blocks.
func ExtractMermaidDiagrams(content string) []string {
	if content == "" {
		return nil
	}

	var diagrams []string

	// use a simple state machine to find mermaid blocks
	// this handles edge cases better than regex alone
	lines := strings.Split(content, "\n")
	var (
		inBlock    bool
		blockLines []string
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			// check for start of mermaid block
			if strings.HasPrefix(trimmed, "```mermaid") {
				inBlock = true
				blockLines = nil
				continue
			}
		} else {
			// check for end of block
			if trimmed == "```" {
				// close the block and save if non-empty
				if len(blockLines) > 0 {
					diagram := strings.TrimSpace(strings.Join(blockLines, "\n"))
					if diagram != "" {
						diagrams = append(diagrams, diagram)
					}
				}
				inBlock = false
				blockLines = nil
				continue
			}
			// accumulate block content
			blockLines = append(blockLines, line)
		}
	}

	// unclosed blocks are ignored (malformed)

	return diagrams
}

// ExtractAllMermaidFromEntries finds all unique mermaid diagrams across entries.
// Returns deduplicated list of diagrams.
func ExtractAllMermaidFromEntries(entries []SessionEntry) []string {
	seen := make(map[string]bool)
	var diagrams []string

	for _, entry := range entries {
		for _, d := range ExtractMermaidDiagrams(entry.Content) {
			if !seen[d] {
				seen[d] = true
				diagrams = append(diagrams, d)
			}
		}
	}

	return diagrams
}
