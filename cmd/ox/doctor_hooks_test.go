package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOxCommands(t *testing.T) {
	tests := []struct {
		name     string
		hookCmd  string
		expected []string
	}{
		{
			name:     "simple ox agent prime",
			hookCmd:  "ox agent prime",
			expected: []string{"ox agent prime"},
		},
		{
			name:     "ox agent prime with shell conditional",
			hookCmd:  "if command -v ox >/dev/null 2>&1; then ox agent prime 2>&1; else echo 'not installed'; fi",
			expected: []string{"ox agent prime"},
		},
		{
			name:     "legacy ox prime command",
			hookCmd:  "ox prime",
			expected: []string{"ox prime"},
		},
		{
			name:     "ox prime with shell conditional",
			hookCmd:  "if command -v ox >/dev/null; then ox prime; fi",
			expected: []string{"ox prime"},
		},
		{
			name:     "multiple ox commands",
			hookCmd:  "ox agent prime && ox doctor",
			expected: []string{"ox agent prime", "ox doctor"},
		},
		{
			name:     "no ox commands",
			hookCmd:  "echo hello",
			expected: nil,
		},
		{
			name:     "ox version",
			hookCmd:  "ox version",
			expected: []string{"ox version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOxCommands(tt.hookCmd)
			require.Len(t, got, len(tt.expected), "extractOxCommands(%q) length mismatch", tt.hookCmd)
			for i, cmd := range got {
				assert.Equal(t, tt.expected[i], cmd, "extractOxCommands(%q)[%d] mismatch", tt.hookCmd, i)
			}
		})
	}
}

func TestIsValidOxCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		valid   bool
	}{
		{"ox agent prime", "ox agent prime", true},
		{"ox doctor", "ox doctor", true},
		{"ox init", "ox init", true},
		{"ox version", "ox version", true},
		{"ox prime (invalid)", "ox prime", false},
		{"ox guidance (invalid)", "ox guidance", false},
		{"ox welcome (invalid)", "ox welcome", false},
		{"ox agent prime with flags", "ox agent prime --user", true},
		{"ox nonexistent", "ox nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidOxCommand(tt.command)
			assert.Equal(t, tt.valid, got, "isValidOxCommand(%q) mismatch", tt.command)
		})
	}
}

func TestGetSuggestionForInvalidCommand(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		suggestion string
	}{
		{"ox prime -> ox agent prime", "ox prime", "ox agent prime"},
		{"ox prime with flags preserves them", "ox prime --user", "ox agent prime --user"},
		{"unknown command has no suggestion", "ox unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSuggestionForInvalidCommand(tt.command)
			assert.Equal(t, tt.suggestion, got, "getSuggestionForInvalidCommand(%q) mismatch", tt.command)
		})
	}
}

func TestValidateHookCommand(t *testing.T) {
	tests := []struct {
		name            string
		hookCmd         string
		expectValid     bool
		expectInvalid   []string
		expectSuggested map[string]string
	}{
		{
			name:          "valid ox agent prime",
			hookCmd:       "ox agent prime",
			expectValid:   true,
			expectInvalid: nil,
		},
		{
			name:          "valid ox agent prime with shell",
			hookCmd:       "if command -v ox >/dev/null; then ox agent prime; fi",
			expectValid:   true,
			expectInvalid: nil,
		},
		{
			name:          "invalid ox prime",
			hookCmd:       "ox prime",
			expectValid:   false,
			expectInvalid: []string{"ox prime"},
			expectSuggested: map[string]string{
				"ox prime": "ox agent prime",
			},
		},
		{
			name:          "invalid ox prime with shell",
			hookCmd:       "if command -v ox >/dev/null; then ox prime; fi",
			expectValid:   false,
			expectInvalid: []string{"ox prime"},
			expectSuggested: map[string]string{
				"ox prime": "ox agent prime",
			},
		},
		{
			name:        "no ox commands is valid",
			hookCmd:     "echo hello",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, invalid, suggestions := ValidateHookCommand(tt.hookCmd)
			assert.Equal(t, tt.expectValid, valid, "ValidateHookCommand(%q) valid mismatch", tt.hookCmd)
			assert.Len(t, invalid, len(tt.expectInvalid), "ValidateHookCommand(%q) invalid count mismatch", tt.hookCmd)
			for cmd, expected := range tt.expectSuggested {
				assert.Equal(t, expected, suggestions[cmd], "ValidateHookCommand(%q) suggestion for %q mismatch", tt.hookCmd, cmd)
			}
		})
	}
}

func TestCheckHookCommands_NoSettingsFile(t *testing.T) {
	// create temp home dir without settings.json
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	assert.True(t, result.skipped, "expected skipped result when no settings.json, got passed=%v warning=%v", result.passed, result.warning)
}

func TestCheckHookCommands_EmptyHooks(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with no hooks
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{},
	}
	data, _ := json.Marshal(settings)
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	assert.True(t, result.skipped, "expected skipped result when hooks empty, got passed=%v warning=%v", result.passed, result.warning)
}

func TestCheckHookCommands_ValidCommands(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with valid ox agent prime hook
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "if command -v ox >/dev/null 2>&1; then ox agent prime 2>&1; fi",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	assert.True(t, result.passed, "expected passed result for valid commands, got passed=%v warning=%v message=%s", result.passed, result.warning, result.message)
	assert.False(t, result.warning, "expected no warning for valid commands, got warning=true detail=%s", result.detail)
}

func TestCheckHookCommands_InvalidCommand(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with invalid ox prime hook (should be ox agent prime)
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "ox prime",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	assert.True(t, result.passed, "expected warning result (passed=true) for invalid commands")
	assert.True(t, result.warning, "expected warning result for invalid commands")
	assert.Equal(t, "1 invalid command(s)", result.message, "message mismatch")
	assert.NotEmpty(t, result.detail, "expected detail with suggestion")
	// check that suggestion is included
	assert.True(t, strings.Contains(result.detail, "ox agent prime"), "expected detail to suggest 'ox agent prime', got %q", result.detail)
}

func TestCheckHookCommands_MultipleInvalidCommands(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with multiple invalid hooks
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "ox prime",
						},
					},
				},
			},
			"PreCompact": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "ox deploy staging",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	assert.True(t, result.passed, "expected warning result (passed=true) for invalid commands")
	assert.True(t, result.warning, "expected warning result for invalid commands")
	assert.Equal(t, "2 invalid command(s)", result.message, "message mismatch")
}

func TestCheckHookCommands_MixedValidAndInvalid(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with both valid and invalid hooks
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "ox agent prime", // valid
						},
						map[string]interface{}{
							"type":    "command",
							"command": "ox prime", // invalid
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	// should warn because there's an invalid command, even though there's a valid one
	assert.True(t, result.passed, "expected warning result (passed=true) for mixed commands")
	assert.True(t, result.warning, "expected warning result for mixed commands")
	assert.Equal(t, "1 invalid command(s)", result.message, "message mismatch")
}

func TestCheckHookCommands_NonCommandHookType(t *testing.T) {
	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// write settings.json with non-command hook type
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "url", // not a command hook
							"command": "ox prime",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	result := checkHookCommands()

	// should skip because there are no command-type hooks
	assert.True(t, result.skipped, "expected skipped result for non-command hooks, got passed=%v warning=%v skipped=%v", result.passed, result.warning, result.skipped)
}

// NeedsHookUpgrade and UpgradeClaudeHooks were removed in favor of
// project-level hook installation via InstallProjectClaudeHooks.
// Legacy user-level hooks are no longer upgraded in-place.
