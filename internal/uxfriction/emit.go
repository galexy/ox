package uxfriction

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sageox/ox/pkg/agentx"
)

// CRITICAL: HUMAN REVIEW REQUIRED
// The emit formats below are intentionally designed for agent learning.
// Changes to these structures can break agent parsing and learning behavior.
// Always get human review before modifying output formats.

// CorrectionMeta contains metadata about a command correction that was auto-executed.
//
// TEACHING PATTERN: This appears in stdout so agents see it in context and learn
// the correct command for future invocations. The underscore prefix (_corrected)
// indicates this is metadata, not command output.
type CorrectionMeta struct {
	// Was is the original command that was typed (incorrect).
	Was string `json:"was"`

	// Now is the corrected command that was executed.
	Now string `json:"now"`

	// Note is an actionable instruction for next time (e.g., "Use 'ox agent prime' next time").
	Note string `json:"note"`
}

// SuggestionMeta contains metadata about a command suggestion (not auto-executed).
// Used when confidence is too low for auto-execute or auto-execute is disabled.
type SuggestionMeta struct {
	// Try is the suggested command to try.
	Try string `json:"try"`

	// Note is an actionable instruction for next time.
	Note string `json:"note"`
}

// CorrectionOutput wraps command output with correction/suggestion metadata.
//
// TEACHING PATTERN: The _corrected field is intentionally in stdout so agents
// see it in their context and learn the correct command for future invocations.
// Do NOT move to stderr - agents need to see this in their context window.
//
// JSON structure examples:
//
// Auto-executed correction:
//
//	{"_corrected": {"was": "ox agent prine", "now": "ox agent prime", "note": "..."}, "result": {...}}
//
// Suggestion (not auto-executed):
//
//	{"_suggestion": {"try": "ox agent prime", "note": "..."}, "error": "unknown command"}
type CorrectionOutput struct {
	// Corrected is present when the command was auto-executed with correction.
	Corrected *CorrectionMeta `json:"_corrected,omitempty"`

	// Suggestion is present when a correction was suggested but not auto-executed.
	Suggestion *SuggestionMeta `json:"_suggestion,omitempty"`

	// Result contains the command output (when command succeeded).
	Result any `json:"result,omitempty"`

	// Error contains the error message (when command failed).
	Error string `json:"error,omitempty"`
}

// EmitCorrection outputs the correction/suggestion for the caller to learn from.
// For agents (JSON mode): writes structured JSON to stdout.
// For humans (text mode): writes human-friendly text to stderr.
func EmitCorrection(result *AutoExecuteResult, jsonMode bool) {
	if result == nil || result.Suggestion == nil {
		return
	}

	if jsonMode || agentx.IsAgentContext() {
		emitCorrectionJSON(result)
	} else {
		emitCorrectionText(result)
	}
}

// emitCorrectionJSON outputs structured JSON for agent learning.
func emitCorrectionJSON(result *AutoExecuteResult) {
	var output CorrectionOutput

	if result.Action == ActionAutoExecute {
		output.Corrected = &CorrectionMeta{
			Was:  result.Suggestion.Original,
			Now:  result.Suggestion.Corrected,
			Note: fmt.Sprintf("Use '%s' next time", result.Suggestion.Corrected),
		}
	} else {
		output.Suggestion = &SuggestionMeta{
			Try:  result.Suggestion.Corrected,
			Note: fmt.Sprintf("Use '%s' next time", result.Suggestion.Corrected),
		}
	}

	data, err := json.Marshal(output)
	if err != nil {
		return
	}

	// TEACHING PATTERN: Write to stdout so agents see it in their context
	fmt.Fprintln(os.Stdout, string(data))
}

// emitCorrectionText outputs human-friendly text to stderr.
func emitCorrectionText(result *AutoExecuteResult) {
	if result.Action == ActionAutoExecute {
		fmt.Fprintf(os.Stderr, "→ Correcting to: %s\n", result.Suggestion.Corrected)
	} else {
		fmt.Fprintf(os.Stderr, "Did you mean?\n    %s\n", result.Suggestion.Corrected)
	}
}

// WrapOutputWithCorrection wraps successful command output with correction metadata.
// This teaches the agent what the correct command is for next time.
// Use this when the corrected command succeeds and you want to include its output.
func WrapOutputWithCorrection(result *AutoExecuteResult, commandOutput any) *CorrectionOutput {
	if result == nil || result.Suggestion == nil {
		return &CorrectionOutput{Result: commandOutput}
	}

	return &CorrectionOutput{
		Corrected: &CorrectionMeta{
			Was:  result.Suggestion.Original,
			Now:  result.Suggestion.Corrected,
			Note: fmt.Sprintf("Use '%s' next time", result.Suggestion.Corrected),
		},
		Result: commandOutput,
	}
}

// EmitSuggestionError outputs an error with a suggestion for agent learning.
// For suggest-only cases where we don't auto-execute.
func EmitSuggestionError(result *AutoExecuteResult, errorMsg string, jsonMode bool) {
	if result == nil || result.Suggestion == nil {
		return
	}

	if jsonMode || agentx.IsAgentContext() {
		output := &CorrectionOutput{
			Error: errorMsg,
			Suggestion: &SuggestionMeta{
				Try:  result.Suggestion.Corrected,
				Note: fmt.Sprintf("Use '%s' next time", result.Suggestion.Corrected),
			},
		}

		data, err := json.Marshal(output)
		if err != nil {
			return
		}
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		emitCorrectionText(result)
	}
}
