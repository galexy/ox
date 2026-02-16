package session

import (
	"strings"
)

// SessionFilterMode represents the level of session recording.
// Values: "none", "all"
type SessionFilterMode string

const (
	SessionFilterModeNone SessionFilterMode = "none" // no automatic sessions
	SessionFilterModeAll  SessionFilterMode = "all"  // all coding agent sessions
)

// IsValid returns true if the mode is a recognized value.
func (m SessionFilterMode) IsValid() bool {
	switch m {
	case SessionFilterModeNone, SessionFilterModeAll, "":
		return true
	}
	return false
}

// ShouldRecord returns true if this mode enables any recording.
func (m SessionFilterMode) ShouldRecord() bool {
	return m != SessionFilterModeNone && m != ""
}

// noise commands that are always filtered out (unless they fail)
var noiseCommands = []string{
	"ls", "pwd", "cd", "clear",
	"cat", "head", "tail", "less", "more",
	"echo", "printf",
	"which", "whereis", "type",
	"env", "export",
}

// IsNoiseCommand checks if a command is noise (low-value unless it fails).
func IsNoiseCommand(cmd string) bool {
	cmdLower := strings.ToLower(strings.TrimSpace(cmd))
	for _, pattern := range noiseCommands {
		if strings.HasPrefix(cmdLower, pattern+" ") || cmdLower == pattern {
			return true
		}
	}
	return false
}

// FilterEvents filters events based on the specified mode.
// mode="all": Remove noise, keep everything meaningful
func FilterEvents(events []ExtractedEvent, mode SessionFilterMode) []ExtractedEvent {
	if len(events) == 0 {
		return events
	}

	switch mode {
	case SessionFilterModeNone, "":
		return nil // no events when mode is none
	case SessionFilterModeAll:
		return filterNoise(events)
	default:
		return filterNoise(events) // default to all mode behavior
	}
}

// filterNoise removes low-value events from the list.
func filterNoise(events []ExtractedEvent) []ExtractedEvent {
	filtered := make([]ExtractedEvent, 0, len(events))
	for _, event := range events {
		if !isNoiseEvent(event) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// isNoiseEvent checks if an event is noise (should be filtered out).
func isNoiseEvent(event ExtractedEvent) bool {
	// errors are never noise
	if event.Type == ExtractedEventErrorOccurred {
		return false
	}

	// commands that are noise patterns
	if event.Type == ExtractedEventCommandRun {
		if IsNoiseCommand(event.Summary) {
			// unless it failed
			if event.Success == nil || *event.Success {
				return true
			}
		}
	}

	return false
}
