package session

import "errors"

// SessionError represents a structured error for JSON output.
// Provides machine-readable error information for coding agents.
type SessionError struct {
	Code      string `json:"code"`          // machine-readable error code (e.g., NOT_RECORDING)
	Message   string `json:"msg"`           // human-readable description
	Retryable bool   `json:"retry"`         // whether the operation can be retried
	Fix       string `json:"fix,omitempty"` // suggested fix command
}

// Error codes for session operations
const (
	ErrCodeNotRecording     = "NOT_RECORDING"
	ErrCodeAlreadyRecording = "ALREADY_RECORDING"
	ErrCodeNoSession        = "NO_SESSION"
	ErrCodeInvalidEntry     = "INVALID_ENTRY"
	ErrCodeStorageError     = "STORAGE_ERROR"
	ErrCodeAuthRequired     = "AUTH_REQUIRED"
)

// NewSessionError creates a new SessionError.
func NewSessionError(code, message string, retryable bool, fix string) *SessionError {
	return &SessionError{
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Fix:       fix,
	}
}

// Error implements the error interface.
func (e *SessionError) Error() string {
	return e.Message
}

// sentinel errors for the session package
var (
	// ErrSessionNotFound is returned when a session file cannot be located
	ErrSessionNotFound = errors.New("session not found")

	// ErrNoSessions is returned when no sessions exist in storage
	ErrNoSessions = errors.New("no sessions found")

	// ErrNoRawSessions is returned when no raw sessions exist
	ErrNoRawSessions = errors.New("no raw sessions found")

	// ErrEmptyPath is returned when a required path argument is empty
	ErrEmptyPath = errors.New("path cannot be empty")

	// ErrEmptyFilename is returned when a required filename argument is empty
	ErrEmptyFilename = errors.New("filename cannot be empty")

	// ErrNilEntry is returned when a nil entry is passed to a write operation
	ErrNilEntry = errors.New("entry cannot be nil")

	// ErrNilData is returned when nil data is passed to a write operation
	ErrNilData = errors.New("data cannot be nil")

	// ErrNilState is returned when a nil state is passed to a save operation
	ErrNilState = errors.New("recording state cannot be nil")

	// ErrSessionNotHydrated is returned when content files are not present locally.
	// The session's meta.json exists (from git pull) but content files need to be
	// downloaded from LFS blob storage via `ox session download`.
	ErrSessionNotHydrated = errors.New("session content not available locally: run 'ox session download' to fetch")
)
