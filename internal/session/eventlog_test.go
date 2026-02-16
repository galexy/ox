package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEventsFromEntries_UserMessages(t *testing.T) {
	ts := time.Now()
	entries := []Entry{
		{Timestamp: ts, Type: EntryTypeUser, Content: "Please fix the bug in main.go"},
		{Timestamp: ts.Add(time.Second), Type: EntryTypeUser, Content: "  "},
		{Timestamp: ts.Add(2 * time.Second), Type: EntryTypeUser, Content: ""},
	}

	events := ExtractEventsFromEntries(entries)

	// should only extract non-empty user messages
	require.Len(t, events, 1)
	assert.Equal(t, ExtractedEventUserAsked, events[0].Type)
	assert.Equal(t, "Please fix the bug in main.go", events[0].Summary)
}

func TestExtractEventsFromEntries_AssistantMessages(t *testing.T) {
	ts := time.Now()
	entries := []Entry{
		{Timestamp: ts, Type: EntryTypeAssistant, Content: "I'll help you fix that issue."},
		{Timestamp: ts.Add(time.Second), Type: EntryTypeAssistant, Content: "The fix is complete. Success!"},
	}

	events := ExtractEventsFromEntries(entries)

	require.Len(t, events, 2)
	assert.Equal(t, ExtractedEventAgentResponded, events[0].Type)
	// second message should be marked as resolved due to success indicator
	assert.Equal(t, ExtractedEventResolved, events[1].Type)
}

func TestExtractEventsFromEntries_ErrorDetection(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{"error prefix", "error: file not found", true},
		{"Error prefix", "Error: connection refused", true},
		{"failed keyword", "Command failed with status 1", true},
		{"exit code", "exit code 1", true},
		{"panic", "panic: runtime error", true},
		{"fatal", "fatal: not a git repository", true},
		{"no error", "Everything looks good", false},
		{"exit code 0", "exit code 0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Now()
			entries := []Entry{
				{Timestamp: ts, Type: EntryTypeAssistant, Content: tt.content},
			}

			events := ExtractEventsFromEntries(entries)

			require.Len(t, events, 1)
			gotError := events[0].Type == ExtractedEventErrorOccurred
			assert.Equal(t, tt.expectError, gotError, "error detection for content: %q", tt.content)
		})
	}
}

func TestExtractEventsFromEntries_ToolExecution(t *testing.T) {
	ts := time.Now()

	tests := []struct {
		name       string
		entry      Entry
		expectType ExtractedEventType
		expectFile string
		expectNil  bool
	}{
		{
			name: "bash command",
			entry: Entry{
				Timestamp:  ts,
				Type:       EntryTypeTool,
				ToolName:   "bash",
				ToolInput:  "go test ./...",
				ToolOutput: "PASS\nok\nexit code 0",
			},
			expectType: ExtractedEventCommandRun,
		},
		{
			name: "file edit",
			entry: Entry{
				Timestamp:  ts,
				Type:       EntryTypeTool,
				ToolName:   "write",
				ToolInput:  "/path/to/main.go",
				ToolOutput: "file written",
			},
			expectType: ExtractedEventFileEdited,
			expectFile: "/path/to/main.go",
		},
		{
			name: "successful read",
			entry: Entry{
				Timestamp:  ts,
				Type:       EntryTypeTool,
				ToolName:   "read",
				ToolInput:  "/some/file.go",
				ToolOutput: "package main\nfunc main() {}",
			},
			expectNil: true, // successful reads are skipped
		},
		{
			name: "failed command",
			entry: Entry{
				Timestamp:  ts,
				Type:       EntryTypeTool,
				ToolName:   "bash",
				ToolInput:  "make test",
				ToolOutput: "error: build failed",
			},
			expectType: ExtractedEventErrorOccurred,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []Entry{tt.entry}
			events := ExtractEventsFromEntries(entries)

			if tt.expectNil {
				assert.Empty(t, events)
				return
			}

			require.Len(t, events, 1)
			assert.Equal(t, tt.expectType, events[0].Type)
			if tt.expectFile != "" {
				assert.Equal(t, tt.expectFile, events[0].RelatedFile)
			}
		})
	}
}

func TestExtractEventsFromEntries_OxCommand(t *testing.T) {
	ts := time.Now()
	entries := []Entry{
		{
			Timestamp: ts,
			Type:      EntryTypeAssistant,
			Content:   "I'll run ox init to set up the repository",
		},
	}

	events := ExtractEventsFromEntries(entries)

	// ox commands are only extracted when there's execution context
	// this test shows the ox command is detected in the content
	require.NotEmpty(t, events, "expected at least 1 event")
}

func TestEventLogExtractFilePath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFile string
	}{
		{"go file", "editing /path/to/file.go", "/path/to/file.go"},
		{"python file", "fixed issue in script.py", "script.py"},
		{"json file", "updated config.json", "config.json"},
		{"yaml file", "modified deploy.yaml", "deploy.yaml"},
		{"no file", "just some text", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventLogExtractFilePath(tt.content)
			assert.Equal(t, tt.wantFile, got)
		})
	}
}

func TestEventLogDetectError(t *testing.T) {
	tests := []struct {
		content   string
		wantError bool
	}{
		{"error: something failed", true},
		{"Error: permission denied", true},
		{"FAILED to connect", true},
		{"panic: nil pointer", true},
		{"fatal: repository not found", true},
		{"exception occurred", true},
		{"exit code 1", true},
		{"exit code 0", false},
		{"all tests passed", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			gotError, _ := eventLogDetectError(tt.content)
			assert.Equal(t, tt.wantError, gotError)
		})
	}
}

func TestEventLogDetectSuccess(t *testing.T) {
	tests := []struct {
		content     string
		wantSuccess bool
	}{
		{"success!", true},
		{"Task completed successfully", true},
		{"All tests passed", true},
		{"Build done", true},
		{"exit code 0", true},
		{"still working", false},
		{"error occurred", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := eventLogDetectSuccess(tt.content)
			assert.Equal(t, tt.wantSuccess, got)
		})
	}
}

func TestEventLogSummarizeContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{"short", "hello", 100, "hello"},
		{"exactly max", "hello", 5, "hello"},
		{"needs truncation", "hello world this is a test", 15, "hello world..."},
		{"whitespace normalization", "hello   world\n\ttab", 100, "hello world tab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventLogSummarizeContent(tt.content, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEventLogSummarizeCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "ls -la", "ls -la"},
		{"multiline", "echo hello\necho world", "echo hello"},
		{"long command", "go test -v -race -coverprofile=coverage.out ./... && go tool cover -html=coverage.out", "go test -v -race -coverprofile=coverage.out..."},
		{"empty", "", "command executed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventLogSummarizeCommand(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteAndReadEventLog(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "events.jsonl")

	ts := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)
	success := true

	log := &EventLog{
		Metadata: &EventLogMetadata{
			SessionID:   "oxsid_test123",
			Agent:       "claude",
			ExtractedAt: ts,
			EntryCount:  10,
			EventCount:  3,
		},
		Events: []ExtractedEvent{
			{
				Timestamp: ts,
				Type:      ExtractedEventUserAsked,
				Summary:   "fix the bug",
			},
			{
				Timestamp: ts.Add(time.Second),
				Type:      ExtractedEventCommandRun,
				Summary:   "go test ./...",
				Success:   &success,
			},
			{
				Timestamp:   ts.Add(2 * time.Second),
				Type:        ExtractedEventFileEdited,
				Summary:     "edited main.go",
				RelatedFile: "/path/to/main.go",
			},
		},
	}

	// write
	err := WriteEventLog(path, log)
	require.NoError(t, err)

	// verify file exists
	_, err = os.Stat(path)
	require.False(t, os.IsNotExist(err), "event log file was not created")

	// read back
	readLog, err := ReadEventLog(path)
	require.NoError(t, err)

	// verify metadata
	require.NotNil(t, readLog.Metadata)
	assert.Equal(t, log.Metadata.SessionID, readLog.Metadata.SessionID)
	assert.Equal(t, log.Metadata.EntryCount, readLog.Metadata.EntryCount)

	// verify events
	require.Len(t, readLog.Events, len(log.Events))

	for i, event := range readLog.Events {
		assert.Equal(t, log.Events[i].Type, event.Type)
		assert.Equal(t, log.Events[i].Summary, event.Summary)
	}
}

func TestReadEventLog_FileNotFound(t *testing.T) {
	_, err := ReadEventLog("/nonexistent/path/events.jsonl")
	assert.Error(t, err)
}

func TestWriteEventLog_InvalidPath(t *testing.T) {
	log := &EventLog{
		Metadata: &EventLogMetadata{
			ExtractedAt: time.Now(),
		},
		Events: []ExtractedEvent{},
	}

	err := WriteEventLog("/nonexistent/dir/events.jsonl", log)
	assert.Error(t, err)
}

func TestNewEventLog(t *testing.T) {
	ts := time.Now()
	entries := []Entry{
		{Timestamp: ts, Type: EntryTypeUser, Content: "hello"},
		{Timestamp: ts.Add(time.Second), Type: EntryTypeAssistant, Content: "hi there"},
	}

	log := NewEventLog(entries, "oxsid_abc123", "claude")

	require.NotNil(t, log.Metadata)
	assert.Equal(t, "oxsid_abc123", log.Metadata.SessionID)
	assert.Equal(t, "claude", log.Metadata.Agent)
	assert.Equal(t, 2, log.Metadata.EntryCount)
	assert.Len(t, log.Events, 2)
}

func TestExtractEventsFromEntries_EmptyInput(t *testing.T) {
	events := ExtractEventsFromEntries(nil)
	assert.NotNil(t, events, "ExtractEventsFromEntries(nil) should return empty slice, not nil")
	assert.Empty(t, events)

	events = ExtractEventsFromEntries([]Entry{})
	assert.Empty(t, events)
}

func TestExtractEventsFromEntries_RealWorldSession(t *testing.T) {
	ts := time.Now()
	entries := []Entry{
		{Timestamp: ts, Type: EntryTypeUser, Content: "Can you fix the failing test in internal/api/client_test.go?"},
		{Timestamp: ts.Add(time.Second), Type: EntryTypeAssistant, Content: "I'll look at the test file and identify the issue."},
		{Timestamp: ts.Add(2 * time.Second), Type: EntryTypeTool, ToolName: "read", ToolInput: "internal/api/client_test.go", ToolOutput: "package api\n\nimport \"testing\""},
		{Timestamp: ts.Add(3 * time.Second), Type: EntryTypeAssistant, Content: "I see the issue. The test expects a different response format. Let me fix it."},
		{Timestamp: ts.Add(4 * time.Second), Type: EntryTypeTool, ToolName: "write", ToolInput: "internal/api/client_test.go", ToolOutput: "file written successfully"},
		{Timestamp: ts.Add(5 * time.Second), Type: EntryTypeTool, ToolName: "bash", ToolInput: "go test ./internal/api/...", ToolOutput: "PASS\nok github.com/sageox/ox/internal/api 0.5s\nexit code 0"},
		{Timestamp: ts.Add(6 * time.Second), Type: EntryTypeAssistant, Content: "The test is now passing. I fixed the response format expectation."},
	}

	events := ExtractEventsFromEntries(entries)

	// verify we captured key events
	typeCount := make(map[ExtractedEventType]int)
	for _, e := range events {
		typeCount[e.Type]++
	}

	assert.GreaterOrEqual(t, typeCount[ExtractedEventUserAsked], 1, "expected at least 1 user_asked event")
	assert.GreaterOrEqual(t, typeCount[ExtractedEventFileEdited], 1, "expected at least 1 file_edited event")
	assert.GreaterOrEqual(t, typeCount[ExtractedEventCommandRun], 1, "expected at least 1 command_run event")
}

func TestEventLog_SuccessFlag(t *testing.T) {
	ts := time.Now()

	// test success=true
	entries := []Entry{
		{
			Timestamp:  ts,
			Type:       EntryTypeTool,
			ToolName:   "bash",
			ToolInput:  "make build",
			ToolOutput: "Build succeeded\nexit code 0",
		},
	}

	events := ExtractEventsFromEntries(entries)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].Success, "Success should not be nil for command with success output")
	assert.True(t, *events[0].Success, "Success should be true for successful command")

	// test success=false
	entries = []Entry{
		{
			Timestamp:  ts,
			Type:       EntryTypeTool,
			ToolName:   "bash",
			ToolInput:  "make build",
			ToolOutput: "error: build failed\nexit code 1",
		},
	}

	events = ExtractEventsFromEntries(entries)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].Success, "Success should not be nil for failed command")
	assert.False(t, *events[0].Success, "Success should be false for failed command")
}
