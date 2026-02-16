package uxfriction

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout captures stdout during f() execution and returns it.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

// captureStderr captures stderr during f() execution and returns it.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestEmitCorrection_JSONMode_AutoExecute(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionCommandRemap,
			Original:   "ox agent prine",
			Corrected:  "ox agent prime",
			Confidence: 0.95,
		},
		Action: ActionAutoExecute,
	}

	output := captureStdout(t, func() {
		EmitCorrection(result, true)
	})

	// should be valid JSON
	var parsed CorrectionOutput
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	// should have _corrected field for auto-execute
	require.NotNil(t, parsed.Corrected, "should have _corrected field")
	assert.Equal(t, "ox agent prine", parsed.Corrected.Was)
	assert.Equal(t, "ox agent prime", parsed.Corrected.Now)
	assert.Contains(t, parsed.Corrected.Note, "ox agent prime")

	// should NOT have _suggestion field
	assert.Nil(t, parsed.Suggestion, "should not have _suggestion for auto-execute")
}

func TestEmitCorrection_JSONMode_SuggestOnly(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionLevenshtein,
			Original:   "ox agent statsu",
			Corrected:  "ox agent status",
			Confidence: 0.75,
		},
		Action: ActionSuggestOnly,
	}

	output := captureStdout(t, func() {
		EmitCorrection(result, true)
	})

	// should be valid JSON
	var parsed CorrectionOutput
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	// should have _suggestion field for suggest-only
	require.NotNil(t, parsed.Suggestion, "should have _suggestion field")
	assert.Equal(t, "ox agent status", parsed.Suggestion.Try)
	assert.Contains(t, parsed.Suggestion.Note, "ox agent status")

	// should NOT have _corrected field
	assert.Nil(t, parsed.Corrected, "should not have _corrected for suggest-only")
}

func TestEmitCorrection_TextMode_AutoExecute(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionCommandRemap,
			Original:   "ox agent prine",
			Corrected:  "ox agent prime",
			Confidence: 0.95,
		},
		Action: ActionAutoExecute,
	}

	output := captureStderr(t, func() {
		EmitCorrection(result, false)
	})

	// should contain the correction indicator
	assert.Contains(t, output, "→")
	assert.Contains(t, output, "Correcting to:")
	assert.Contains(t, output, "ox agent prime")
}

func TestEmitCorrection_TextMode_SuggestOnly(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionLevenshtein,
			Original:   "ox agent statsu",
			Corrected:  "ox agent status",
			Confidence: 0.75,
		},
		Action: ActionSuggestOnly,
	}

	output := captureStderr(t, func() {
		EmitCorrection(result, false)
	})

	// should contain the suggestion prompt
	assert.Contains(t, output, "Did you mean?")
	assert.Contains(t, output, "ox agent status")
}

func TestEmitCorrection_NilResult(t *testing.T) {
	// should not panic or output anything with nil result
	stdout := captureStdout(t, func() {
		EmitCorrection(nil, true)
	})
	stderr := captureStderr(t, func() {
		EmitCorrection(nil, false)
	})

	assert.Empty(t, stdout, "nil result should produce no stdout")
	assert.Empty(t, stderr, "nil result should produce no stderr")
}

func TestEmitCorrection_NilSuggestion(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: nil,
		Action:     ActionSuggestOnly,
	}

	// should not panic or output anything with nil suggestion
	stdout := captureStdout(t, func() {
		EmitCorrection(result, true)
	})
	stderr := captureStderr(t, func() {
		EmitCorrection(result, false)
	})

	assert.Empty(t, stdout, "nil suggestion should produce no stdout")
	assert.Empty(t, stderr, "nil suggestion should produce no stderr")
}

func TestEmitSuggestionError_JSONMode(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionLevenshtein,
			Original:   "ox agent statsu",
			Corrected:  "ox agent status",
			Confidence: 0.75,
		},
		Action: ActionSuggestOnly,
	}

	output := captureStdout(t, func() {
		EmitSuggestionError(result, "unknown command: statsu", true)
	})

	// should be valid JSON
	var parsed CorrectionOutput
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	// should have error field
	assert.Equal(t, "unknown command: statsu", parsed.Error)

	// should have _suggestion field
	require.NotNil(t, parsed.Suggestion, "should have _suggestion field")
	assert.Equal(t, "ox agent status", parsed.Suggestion.Try)
	assert.Contains(t, parsed.Suggestion.Note, "ox agent status")
}

func TestEmitSuggestionError_TextMode(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionLevenshtein,
			Original:   "ox agent statsu",
			Corrected:  "ox agent status",
			Confidence: 0.75,
		},
		Action: ActionSuggestOnly,
	}

	output := captureStderr(t, func() {
		EmitSuggestionError(result, "unknown command: statsu", false)
	})

	// should contain the suggestion (delegates to emitCorrectionText)
	assert.Contains(t, output, "Did you mean?")
	assert.Contains(t, output, "ox agent status")
}

func TestEmitSuggestionError_NilResult(t *testing.T) {
	stdout := captureStdout(t, func() {
		EmitSuggestionError(nil, "some error", true)
	})
	stderr := captureStderr(t, func() {
		EmitSuggestionError(nil, "some error", false)
	})

	assert.Empty(t, stdout, "nil result should produce no stdout")
	assert.Empty(t, stderr, "nil result should produce no stderr")
}

func TestEmitSuggestionError_NilSuggestion(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: nil,
		Action:     ActionSuggestOnly,
	}

	stdout := captureStdout(t, func() {
		EmitSuggestionError(result, "some error", true)
	})
	stderr := captureStderr(t, func() {
		EmitSuggestionError(result, "some error", false)
	})

	assert.Empty(t, stdout, "nil suggestion should produce no stdout")
	assert.Empty(t, stderr, "nil suggestion should produce no stderr")
}

func TestWrapOutputWithCorrection(t *testing.T) {
	tests := []struct {
		name          string
		result        *AutoExecuteResult
		commandOutput any
		wantCorrected bool
		wantWas       string
		wantNow       string
		wantResult    any
	}{
		{
			name: "wraps output with correction metadata",
			result: &AutoExecuteResult{
				Suggestion: &Suggestion{
					Original:  "ox agent prine",
					Corrected: "ox agent prime",
				},
				Action: ActionAutoExecute,
			},
			commandOutput: map[string]string{"agent_id": "Oxa123"},
			wantCorrected: true,
			wantWas:       "ox agent prine",
			wantNow:       "ox agent prime",
			wantResult:    map[string]string{"agent_id": "Oxa123"},
		},
		{
			name:          "nil result returns just output",
			result:        nil,
			commandOutput: map[string]string{"status": "ok"},
			wantCorrected: false,
			wantResult:    map[string]string{"status": "ok"},
		},
		{
			name: "nil suggestion returns just output",
			result: &AutoExecuteResult{
				Suggestion: nil,
				Action:     ActionSuggestOnly,
			},
			commandOutput: "plain string output",
			wantCorrected: false,
			wantResult:    "plain string output",
		},
		{
			name: "handles string output",
			result: &AutoExecuteResult{
				Suggestion: &Suggestion{
					Original:  "ox status",
					Corrected: "ox doctor",
				},
				Action: ActionAutoExecute,
			},
			commandOutput: "health check passed",
			wantCorrected: true,
			wantWas:       "ox status",
			wantNow:       "ox doctor",
			wantResult:    "health check passed",
		},
		{
			name: "handles nil output",
			result: &AutoExecuteResult{
				Suggestion: &Suggestion{
					Original:  "ox clean",
					Corrected: "ox cleanup",
				},
				Action: ActionAutoExecute,
			},
			commandOutput: nil,
			wantCorrected: true,
			wantWas:       "ox clean",
			wantNow:       "ox cleanup",
			wantResult:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := WrapOutputWithCorrection(tt.result, tt.commandOutput)

			require.NotNil(t, output, "output should not be nil")

			if tt.wantCorrected {
				require.NotNil(t, output.Corrected, "should have Corrected field")
				assert.Equal(t, tt.wantWas, output.Corrected.Was)
				assert.Equal(t, tt.wantNow, output.Corrected.Now)
				assert.Contains(t, output.Corrected.Note, tt.wantNow)
			} else {
				assert.Nil(t, output.Corrected, "should not have Corrected field")
			}

			assert.Equal(t, tt.wantResult, output.Result)
		})
	}
}

func TestWrapOutputWithCorrection_JSONSerialization(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Original:  "ox agent prine --review",
			Corrected: "ox agent prime --review",
		},
		Action: ActionAutoExecute,
	}
	commandOutput := map[string]any{
		"agent_id": "Oxa7b3c4d5",
		"status":   "registered",
	}

	output := WrapOutputWithCorrection(result, commandOutput)

	// should serialize to valid JSON
	data, err := json.Marshal(output)
	require.NoError(t, err, "should serialize to JSON")

	// parse back and verify structure
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "should parse JSON")

	// verify _corrected field
	corrected, ok := parsed["_corrected"].(map[string]any)
	require.True(t, ok, "should have _corrected as object")
	assert.Equal(t, "ox agent prine --review", corrected["was"])
	assert.Equal(t, "ox agent prime --review", corrected["now"])
	assert.Contains(t, corrected["note"], "ox agent prime --review")

	// verify result field
	resultField, ok := parsed["result"].(map[string]any)
	require.True(t, ok, "should have result as object")
	assert.Equal(t, "Oxa7b3c4d5", resultField["agent_id"])
	assert.Equal(t, "registered", resultField["status"])
}

func TestCorrectionOutput_OmitsEmptyFields(t *testing.T) {
	// verify omitempty works correctly
	output := &CorrectionOutput{
		Result: "test",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	// should not include _corrected or _suggestion or error
	assert.NotContains(t, string(data), "_corrected")
	assert.NotContains(t, string(data), "_suggestion")
	assert.NotContains(t, string(data), "error")
}

func TestEmitCorrection_AllSuggestionTypes(t *testing.T) {
	suggestionTypes := []SuggestionType{
		SuggestionCommandRemap,
		SuggestionTokenFix,
		SuggestionLevenshtein,
	}

	for _, st := range suggestionTypes {
		t.Run(string(st), func(t *testing.T) {
			result := &AutoExecuteResult{
				Suggestion: &Suggestion{
					Type:       st,
					Original:   "ox test",
					Corrected:  "ox test-fixed",
					Confidence: 0.9,
				},
				Action: ActionAutoExecute,
			}

			// JSON mode
			jsonOutput := captureStdout(t, func() {
				EmitCorrection(result, true)
			})
			assert.NotEmpty(t, jsonOutput, "JSON output should not be empty")

			var parsed CorrectionOutput
			err := json.Unmarshal([]byte(strings.TrimSpace(jsonOutput)), &parsed)
			require.NoError(t, err, "should be valid JSON")

			// text mode
			textOutput := captureStderr(t, func() {
				EmitCorrection(result, false)
			})
			assert.NotEmpty(t, textOutput, "text output should not be empty")
			assert.Contains(t, textOutput, "ox test-fixed")
		})
	}
}

func TestEmitCorrection_NoteFormat(t *testing.T) {
	result := &AutoExecuteResult{
		Suggestion: &Suggestion{
			Type:       SuggestionCommandRemap,
			Original:   "ox daemons list --every",
			Corrected:  "ox daemons show --all",
			Confidence: 0.95,
		},
		Action: ActionAutoExecute,
	}

	output := captureStdout(t, func() {
		EmitCorrection(result, true)
	})

	var parsed CorrectionOutput
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	require.NoError(t, err)

	// note should be actionable instruction
	assert.Equal(t, "Use 'ox daemons show --all' next time", parsed.Corrected.Note)
}
