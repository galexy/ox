package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sageox/ox/internal/paths"
)

// SessionMarkerDir returns the per-user directory for session markers.
// Uses paths.TempDir()/sessions/ for user isolation. See TempDir() for
// why /tmp/<user>/sageox/ instead of /tmp/sageox/.
func SessionMarkerDir() string {
	return filepath.Join(paths.TempDir(), "sessions")
}

// SessionMarker represents the contents of a session marker file.
// Stored as JSON for structured data access.
type SessionMarker struct {
	AgentID         string    `json:"agent_id"`
	SessionID       string    `json:"session_id,omitempty"`    // ox session ID, not Claude's
	ClaudeSessionID string    `json:"claude_session_id"`       // Claude's session_id from hook JSON
	PrimedAt        time.Time `json:"primed_at"`               // when session was primed
	LastNotified    time.Time `json:"last_notified,omitempty"` // mtime of last context file check
}

// ClaudeHookInput represents the JSON input Claude Code passes to hooks via stdin.
// Claude Code passes a JSON object with session_id and hook_event_name fields.
type ClaudeHookInput struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
	// Other fields may be present but we only need session_id
}

// markerPath returns the path to the marker file for a given Claude session ID.
func markerPath(claudeSessionID string) string {
	// sanitize session ID to prevent path traversal
	sanitized := strings.ReplaceAll(claudeSessionID, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, "..", "_")
	return filepath.Join(SessionMarkerDir(), sanitized+".json")
}

// ReadSessionMarker reads a session marker from disk.
// Returns nil, nil if the marker doesn't exist.
func ReadSessionMarker(claudeSessionID string) (*SessionMarker, error) {
	if claudeSessionID == "" {
		return nil, nil
	}

	path := markerPath(claudeSessionID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read marker: %w", err)
	}

	var marker SessionMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil, fmt.Errorf("failed to parse marker: %w", err)
	}

	// ensure ClaudeSessionID is set (may not be in old files)
	if marker.ClaudeSessionID == "" {
		marker.ClaudeSessionID = claudeSessionID
	}

	return &marker, nil
}

// WriteSessionMarker writes a session marker to disk.
// Creates the marker directory if it doesn't exist.
// Uses atomic write pattern (temp file + rename) for safety.
func WriteSessionMarker(marker *SessionMarker) error {
	if marker.ClaudeSessionID == "" {
		return fmt.Errorf("claude session ID is required")
	}

	// ensure directory exists
	if err := os.MkdirAll(SessionMarkerDir(), 0700); err != nil {
		return fmt.Errorf("failed to create marker directory: %w", err)
	}

	path := markerPath(marker.ClaudeSessionID)

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal marker: %w", err)
	}

	// atomic write: temp file + rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write marker temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // clean up on failure
		return fmt.Errorf("failed to rename marker: %w", err)
	}

	return nil
}

// UpdateLastNotified updates the LastNotified field and writes to disk.
// On write failure, rolls back the in-memory value.
func (m *SessionMarker) UpdateLastNotified(t time.Time) error {
	oldValue := m.LastNotified
	m.LastNotified = t

	if err := WriteSessionMarker(m); err != nil {
		m.LastNotified = oldValue // rollback on failure
		return err
	}
	return nil
}

// DeleteSessionMarker removes a session marker from disk.
func DeleteSessionMarker(claudeSessionID string) error {
	if claudeSessionID == "" {
		return nil
	}
	path := markerPath(claudeSessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ReadClaudeHookInput reads Claude hook input from stdin.
// Returns nil if stdin is not available or doesn't contain valid JSON.
// This is designed to be non-blocking and fail gracefully.
func ReadClaudeHookInput() *ClaudeHookInput {
	// check if stdin has data (non-blocking check)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil
	}

	// check if stdin is a pipe or has data
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// stdin is a terminal, not a pipe - no hook input
		return nil
	}

	// read up to 4KB (hook input should be small)
	buf := make([]byte, 4096)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return nil
	}

	var input ClaudeHookInput
	if err := json.Unmarshal(buf[:n], &input); err != nil {
		return nil
	}

	// validate we got a session_id (required for marker keying)
	if input.SessionID == "" {
		return nil
	}

	return &input
}

// WriteToClaudeEnvFile writes environment variables to CLAUDE_ENV_FILE if available.
// This makes the variables available to subsequent Bash tool calls in the same session.
func WriteToClaudeEnvFile(vars map[string]string) error {
	envFilePath := os.Getenv("CLAUDE_ENV_FILE")
	if envFilePath == "" {
		return nil // not in Claude context or env file not available
	}

	file, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CLAUDE_ENV_FILE: %w", err)
	}
	defer file.Close()

	for key, value := range vars {
		// write as export statements for bash sourcing
		fmt.Fprintf(file, "export %s=%q\n", key, value)
	}

	return nil
}

// IsClaudeHookContext detects if we're running in a Claude Code hook context.
// Returns true if either:
// - CLAUDE_PROJECT_DIR is set (environment variable)
// - Stdin contains valid hook JSON with session_id
func IsClaudeHookContext() bool {
	// check CLAUDE_PROJECT_DIR (set by Claude Code)
	if os.Getenv("CLAUDE_PROJECT_DIR") != "" {
		return true
	}

	// check CLAUDECODE env var
	if os.Getenv("CLAUDECODE") == "1" {
		return true
	}

	// check CLAUDE_CODE_ENTRYPOINT
	if os.Getenv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return true
	}

	return false
}
