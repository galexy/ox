// Package frictionapi provides types and client for the friction telemetry API.
// This package is designed to be standalone with no internal dependencies to avoid
// import cycles between daemon, uxfriction, and other packages.
package frictionapi

import "time"

// FrictionEvent represents a single UX friction event.
// Events are anonymized and truncated before transmission.
type FrictionEvent struct {
	// Timestamp in ISO8601 UTC format.
	Timestamp string `json:"ts"`

	// Kind identifies the type of friction event.
	// Valid values: unknown-command, unknown-flag, invalid-arg, parse-error
	Kind string `json:"kind"`

	// Command is the top-level command (e.g., "agent" in "ox agent prime").
	Command string `json:"command,omitempty"`

	// Subcommand is the subcommand if applicable (e.g., "prime" in "ox agent prime").
	Subcommand string `json:"subcommand,omitempty"`

	// Actor identifies who triggered the event.
	// Valid values: human, agent
	Actor string `json:"actor"`

	// AgentType identifies the agent type when Actor is "agent".
	// Examples: claude-code, cursor
	AgentType string `json:"agent_type,omitempty"`

	// PathBucket categorizes the working directory (home, repo, other).
	PathBucket string `json:"path_bucket"`

	// Input is the redacted command input (max 500 chars).
	Input string `json:"input"`

	// ErrorMsg is the redacted error message (max 200 chars).
	ErrorMsg string `json:"error_msg"`

	// Suggestion is an optional suggested correction.
	Suggestion string `json:"suggestion,omitempty"`
}

// MaxInputLength is the maximum length for the Input field.
const MaxInputLength = 500

// MaxErrorLength is the maximum length for the ErrorMsg field.
const MaxErrorLength = 200

// Truncate enforces maximum field lengths.
func (f *FrictionEvent) Truncate() {
	if len(f.Input) > MaxInputLength {
		f.Input = f.Input[:MaxInputLength]
	}
	if len(f.ErrorMsg) > MaxErrorLength {
		f.ErrorMsg = f.ErrorMsg[:MaxErrorLength]
	}
}

// NewFrictionEvent creates a new FrictionEvent with the given kind and current timestamp.
func NewFrictionEvent(kind string) FrictionEvent {
	return FrictionEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Kind:      kind,
	}
}

// FailureKind represents types of CLI failures that create friction.
type FailureKind string

const (
	FailureUnknownCommand FailureKind = "unknown-command"
	FailureUnknownFlag    FailureKind = "unknown-flag"
	FailureInvalidArg     FailureKind = "invalid-arg"
	FailureParseError     FailureKind = "parse-error"
)

// Actor represents who triggered a friction event.
type Actor string

const (
	ActorHuman   Actor = "human"
	ActorAgent   Actor = "agent"
	ActorUnknown Actor = "unknown"
)

// CatalogData represents the friction catalog returned by the server.
// It contains learned corrections for common typos and command mistakes.
type CatalogData struct {
	// Version is the catalog version identifier (e.g., "v2026-01-16-001").
	Version string `json:"version"`

	// Commands contains full command remapping rules for renamed/restructured commands.
	Commands []CommandMapping `json:"commands,omitempty"`

	// Tokens contains single-token correction rules (typos, aliases).
	Tokens []TokenMapping `json:"tokens,omitempty"`
}

// CommandMapping represents a full command remapping rule.
// Used for renamed or restructured commands.
type CommandMapping struct {
	// Pattern is the normalized input pattern (no "ox" prefix, flags sorted).
	Pattern string `json:"pattern"`

	// Target is the correct command to suggest.
	Target string `json:"target"`

	// Count is how many times this pattern was seen.
	Count int `json:"count"`

	// Confidence is 0.0-1.0, derived from count and consistency.
	Confidence float64 `json:"confidence"`

	// Description is an optional explanation for humans.
	Description string `json:"description,omitempty"`
}

// TokenMapping represents a single-token correction rule.
// Used for typos and aliases.
type TokenMapping struct {
	// Pattern is the typo/alias (lowercase).
	Pattern string `json:"pattern"`

	// Target is the correct token.
	Target string `json:"target"`

	// Kind specifies which failure kind this applies to.
	Kind string `json:"kind"`

	// Count is how many times this pattern was seen.
	Count int `json:"count"`

	// Confidence is 0.0-1.0, derived from count and consistency.
	Confidence float64 `json:"confidence"`
}

// FrictionResponse represents the API response from friction event submission.
type FrictionResponse struct {
	// Accepted is the number of events successfully processed.
	Accepted int `json:"accepted"`

	// Catalog contains catalog data if updated or client version is stale.
	// nil if catalog version matches X-Catalog-Version header.
	Catalog *CatalogData `json:"catalog,omitempty"`
}
