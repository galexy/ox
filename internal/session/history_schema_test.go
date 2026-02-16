package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHistory_ValidComplete(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentType:     "claude-code",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
			CapturedAt:    time.Now(),
		},
		Entries: []HistoryEntry{
			{Timestamp: time.Now(), Type: "user", Content: "hello", Seq: 1, Source: "planning_history"},
			{Timestamp: time.Now(), Type: "assistant", Content: "hi there", Seq: 2, Source: "planning_history"},
		},
	}

	result := ValidateHistory(history)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 2, result.EntryCount)
	assert.True(t, result.HasMeta)
}

func TestValidateHistory_MissingMeta(t *testing.T) {
	history := &CapturedHistory{
		Meta: nil,
		Entries: []HistoryEntry{
			{Timestamp: time.Now(), Type: "user", Content: "hello", Seq: 1},
		},
	}

	result := ValidateHistory(history)

	assert.False(t, result.Valid)
	assert.False(t, result.HasMeta)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "missing _meta")
}

func TestValidateHistory_InvalidMeta(t *testing.T) {
	tests := []struct {
		name    string
		meta    *HistoryMeta
		wantErr string
	}{
		{
			name: "missing schema_version",
			meta: &HistoryMeta{
				SchemaVersion: "",
				AgentType:     "claude-code",
				AgentID:       "test-agent",
				Source:        "agent_reconstruction",
			},
			wantErr: "schema_version",
		},
		{
			name: "missing source",
			meta: &HistoryMeta{
				SchemaVersion: "1",
				AgentType:     "claude-code",
				AgentID:       "test-agent",
				Source:        "",
			},
			wantErr: "source",
		},
		{
			name: "missing agent_id",
			meta: &HistoryMeta{
				SchemaVersion: "1",
				AgentType:     "claude-code",
				AgentID:       "",
				Source:        "agent_reconstruction",
			},
			wantErr: "agent_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := &CapturedHistory{
				Meta:    tt.meta,
				Entries: []HistoryEntry{},
			}

			result := ValidateHistory(history)

			assert.False(t, result.Valid)
			require.NotEmpty(t, result.Errors)
			assert.Contains(t, result.Errors[0], tt.wantErr)
		})
	}
}

func TestValidateHistory_InvalidEntryType(t *testing.T) {
	tests := []struct {
		name      string
		entryType string
		wantErr   string
	}{
		{"invalid type", "unknown", "invalid"},
		{"typo user", "usr", "invalid"},
		{"uppercase", "USER", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := &CapturedHistory{
				Meta: &HistoryMeta{
					SchemaVersion: "1",
					AgentType:     "claude-code",
					AgentID:       "test-agent",
					Source:        "agent_reconstruction",
				},
				Entries: []HistoryEntry{
					{Timestamp: time.Now(), Type: tt.entryType, Content: "test", Seq: 1},
				},
			}

			result := ValidateHistory(history)

			assert.False(t, result.Valid)
			require.NotEmpty(t, result.Errors)
			assert.Contains(t, result.Errors[0], tt.wantErr)
		})
	}
}

func TestValidateHistory_NonMonotonicSeq(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{
			{Type: "user", Content: "first", Seq: 5},
			{Type: "assistant", Content: "second", Seq: 3},
		},
	}

	result := ValidateHistory(history)

	assert.False(t, result.Valid)
	found := false
	for _, err := range result.Errors {
		if strings.Contains(err, "not monotonically increasing") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected non-monotonic seq error")
}

func TestValidateHistory_DuplicateSeq(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{
			{Type: "user", Content: "first", Seq: 1},
			{Type: "assistant", Content: "second", Seq: 1},
		},
	}

	result := ValidateHistory(history)

	assert.False(t, result.Valid)
	found := false
	for _, err := range result.Errors {
		if strings.Contains(err, "duplicate seq") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected duplicate seq error")
}

func TestValidateHistory_EmptyContent(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{
			{Type: "user", Content: "", Seq: 1},
			{Type: "assistant", Content: "", Seq: 2},
		},
	}

	result := ValidateHistory(history)
	assert.True(t, result.Valid, "empty content should be allowed in ValidateHistory")
}

func TestValidateHistory_MissingSeq(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{
			{Type: "user", Content: "no seq"},
		},
	}

	result := ValidateHistory(history)
	assert.True(t, result.Valid)
}

func TestValidateHistory_PlanningHistory(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "manual",
			Source:        "planning_history",
		},
		Entries: []HistoryEntry{
			{Type: "user", Content: "prompt 1", Seq: 1, Source: "planning_history"},
			{Type: "assistant", Content: "response 1", Seq: 2, Source: "planning_history"},
		},
	}

	result := ValidateHistory(history)

	assert.True(t, result.Valid)
	assert.Equal(t, 2, result.EntryCount)
}

func TestValidateHistory_TimeRangeCalculation(t *testing.T) {
	now := time.Now()
	earliest := now.Add(-1 * time.Hour)
	latest := now

	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{
			{Timestamp: now.Add(-30 * time.Minute), Type: "user", Content: "middle", Seq: 1},
			{Timestamp: earliest, Type: "assistant", Content: "earliest", Seq: 2},
			{Timestamp: latest, Type: "user", Content: "latest", Seq: 3},
		},
	}

	timeRange := history.ComputeTimeRange()

	require.NotNil(t, timeRange)
	assert.Equal(t, earliest, timeRange.Earliest)
	assert.Equal(t, latest, timeRange.Latest)
}

func TestValidateHistory_TimeRangeEmpty(t *testing.T) {
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			AgentID:       "test-agent",
			Source:        "agent_reconstruction",
		},
		Entries: []HistoryEntry{},
	}

	timeRange := history.ComputeTimeRange()
	assert.Nil(t, timeRange)
	assert.Equal(t, time.Duration(0), history.Duration())
}

func TestValidateHistoryEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   *HistoryEntry
		wantErr error
	}{
		{"nil entry", nil, ErrNilEntry},
		{"valid user", &HistoryEntry{Type: "user", Content: "hello"}, nil},
		{"valid assistant", &HistoryEntry{Type: "assistant", Content: "hello"}, nil},
		{"valid system", &HistoryEntry{Type: "system", Content: "context"}, nil},
		{"valid tool", &HistoryEntry{Type: "tool", ToolName: "bash", Content: "ls"}, nil},
		{"tool without tool_name", &HistoryEntry{Type: "tool", Content: "output"}, errors.New("tool entries require tool_name")},
		{"empty type", &HistoryEntry{Type: "", Content: "hello"}, errors.New("entry type is required")},
		{"invalid type", &HistoryEntry{Type: "invalid", Content: "hello"}, errors.New("invalid entry type")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHistoryEntry(tt.entry)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHistoryMeta(t *testing.T) {
	tests := []struct {
		name    string
		meta    *HistoryMeta
		wantErr error
	}{
		{"nil meta", nil, ErrHistoryMissingMeta},
		{"valid meta", &HistoryMeta{SchemaVersion: "1", AgentID: "test", Source: "test"}, nil},
		{"missing schema", &HistoryMeta{SchemaVersion: "", AgentID: "test", Source: "test"}, ErrHistoryInvalidMeta},
		{"missing source", &HistoryMeta{SchemaVersion: "1", AgentID: "test", Source: ""}, ErrHistoryInvalidMeta},
		{"missing agent_id", &HistoryMeta{SchemaVersion: "1", AgentID: "", Source: "test"}, ErrHistoryInvalidMeta},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHistoryMeta(tt.meta)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseHistoryEntry(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantMeta  bool
		wantEntry bool
		wantErr   bool
	}{
		{"valid meta", `{"_meta":{"schema_version":"1","agent_id":"test","source":"test"}}`, true, false, false},
		{"valid entry", `{"type":"user","content":"hello","seq":1}`, false, true, false},
		{"invalid json", `{invalid}`, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, entry, err := ParseHistoryEntry([]byte(tt.line))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.wantMeta {
				assert.NotNil(t, meta)
			}
			if tt.wantEntry {
				assert.NotNil(t, entry)
			}
		})
	}
}

func TestValidateHistoryJSONLReader(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","agent_id":"test","source":"test"}}
{"type":"user","content":"hello","seq":1}
{"type":"assistant","content":"hi","seq":2}`

	history, err := ValidateHistoryJSONLReader(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history.Meta)
	assert.Equal(t, "1", history.Meta.SchemaVersion)
	require.Len(t, history.Entries, 2)
}

func TestValidateHistoryJSONL(t *testing.T) {
	tests := []struct {
		name      string
		jsonl     string
		wantValid bool
	}{
		{
			name: "valid history",
			jsonl: `{"_meta":{"schema_version":"1","agent_id":"test","source":"test"}}
{"type":"user","content":"hello","seq":1}`,
			wantValid: true,
		},
		{
			name:      "missing meta",
			jsonl:     `{"type":"user","content":"hello","seq":1}`,
			wantValid: false,
		},
		{
			name: "invalid type",
			jsonl: `{"_meta":{"schema_version":"1","agent_id":"test","source":"test"}}
{"type":"bogus","content":"hello","seq":1}`,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateHistoryJSONL(tt.jsonl)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, result.Valid)
		})
	}
}

func TestValidateHistoryFile(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid.jsonl")
	validContent := `{"_meta":{"schema_version":"1","agent_id":"test","source":"test"}}
{"type":"user","content":"hello","seq":1}`

	err := os.WriteFile(validPath, []byte(validContent), 0644)
	require.NoError(t, err)

	result, err := ValidateHistoryFile(validPath)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, 1, result.EntryCount)
}

func TestValidateHistoryFile_NotFound(t *testing.T) {
	result, err := ValidateHistoryFile("/nonexistent/path/file.jsonl")

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "open history file")
}

func TestHistoryEntryToSessionEntry(t *testing.T) {
	now := time.Now()
	entry := &HistoryEntry{
		Timestamp:  now,
		Type:       "tool",
		Content:    "output",
		ToolName:   "bash",
		ToolInput:  "ls",
		ToolOutput: "file1",
	}

	session := entry.ToSessionEntry()

	assert.Equal(t, now, session.Timestamp)
	assert.Equal(t, SessionEntryTypeTool, session.Type)
	assert.Equal(t, "bash", session.ToolName)
}

func TestHistoryEntryFromSessionEntry(t *testing.T) {
	now := time.Now()
	session := SessionEntry{
		Timestamp: now,
		Type:      SessionEntryTypeUser,
		Content:   "hello",
	}

	entry := HistoryEntryFromSessionEntry(session, 42, "planning_history")

	assert.Equal(t, now, entry.Timestamp)
	assert.Equal(t, "user", entry.Type)
	assert.Equal(t, 42, entry.Seq)
	assert.Equal(t, "planning_history", entry.Source)
}

func TestNewHistoryMeta(t *testing.T) {
	meta := NewHistoryMeta("test-agent", "agent_reconstruction")

	assert.Equal(t, HistorySchemaVersion, meta.SchemaVersion)
	assert.Equal(t, "test-agent", meta.AgentID)
	assert.Equal(t, "agent_reconstruction", meta.Source)
	assert.False(t, meta.CapturedAt.IsZero())
}

func TestNewHistoryEntry(t *testing.T) {
	entry := NewHistoryEntry(5, "user", "hello")

	assert.Equal(t, 5, entry.Seq)
	assert.Equal(t, "user", entry.Type)
	assert.Equal(t, "hello", entry.Content)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestIsValidEntryType(t *testing.T) {
	assert.True(t, IsValidEntryType("user"))
	assert.True(t, IsValidEntryType("assistant"))
	assert.True(t, IsValidEntryType("system"))
	assert.True(t, IsValidEntryType("tool"))
	assert.False(t, IsValidEntryType("invalid"))
	assert.False(t, IsValidEntryType(""))
}

func TestCapturedHistory_EntryCount(t *testing.T) {
	history := &CapturedHistory{
		Meta:    &HistoryMeta{SchemaVersion: "1", AgentID: "test", Source: "test"},
		Entries: []HistoryEntry{{}, {}, {}},
	}

	assert.Equal(t, 3, history.EntryCount())
}

func TestCapturedHistory_GetEntriesByType(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{Type: "user"},
			{Type: "assistant"},
			{Type: "user"},
		},
	}

	users := history.GetEntriesByType("user")
	assert.Len(t, users, 2)
}

func TestCapturedHistory_GetPlanEntries(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{IsPlan: true},
			{IsPlan: false},
			{IsPlan: true},
		},
	}

	plans := history.GetPlanEntries()
	assert.Len(t, plans, 2)
}

func TestCapturedHistory_Duration(t *testing.T) {
	now := time.Now()
	history := &CapturedHistory{
		Meta: &HistoryMeta{
			TimeRange: &HistoryTimeRange{
				Earliest: now.Add(-1 * time.Hour),
				Latest:   now,
			},
		},
	}

	assert.Equal(t, time.Hour, history.Duration())
}

func TestCapturedHistory_Duration_NilMeta(t *testing.T) {
	history := &CapturedHistory{Meta: nil}
	assert.Equal(t, time.Duration(0), history.Duration())
}

func TestCapturedHistory_HasPlanningHistory(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{Source: "planning_history"},
			{Source: ""},
		},
	}

	assert.True(t, history.HasPlanningHistory())
}

func TestCapturedHistory_HasPlanningHistory_None(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{Source: ""},
		},
	}

	assert.False(t, history.HasPlanningHistory())
}

func TestCapturedHistory_PlanningHistoryCount(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{Source: "planning_history"},
			{Source: "planning_history"},
			{Source: ""},
		},
	}

	assert.Equal(t, 2, history.PlanningHistoryCount())
}

func TestCapturedHistory_ToSessionEntries(t *testing.T) {
	history := &CapturedHistory{
		Entries: []HistoryEntry{
			{Type: "user", Content: "hello"},
			{Type: "assistant", Content: "hi"},
		},
	}

	sessions := history.ToSessionEntries()

	require.Len(t, sessions, 2)
	assert.Equal(t, SessionEntryTypeUser, sessions[0].Type)
	assert.Equal(t, SessionEntryTypeAssistant, sessions[1].Type)
}
