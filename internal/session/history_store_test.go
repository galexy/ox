package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCapturedHistory_NilHistory(t *testing.T) {
	_, err := StoreCapturedHistory(nil, "test-agent", false)
	if err != ErrHistoryNilHistory {
		t.Errorf("expected ErrHistoryNilHistory, got %v", err)
	}
}

func TestStoreCapturedHistory_EmptyEntries(t *testing.T) {
	history := &CapturedHistory{
		Meta:    &HistoryMeta{SchemaVersion: "1", Source: "test", AgentID: "test"},
		Entries: []HistoryEntry{},
	}
	_, err := StoreCapturedHistory(history, "test-agent", false)
	if err != ErrHistoryEmptyEntries {
		t.Errorf("expected ErrHistoryEmptyEntries, got %v", err)
	}
}

func TestWriteHistoryJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-history.jsonl")

	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			Source:        "agent_reconstruction",
			AgentID:       "test-agent",
			CapturedAt:    time.Now(),
		},
		Entries: []HistoryEntry{
			{Seq: 1, Type: "user", Content: "Hello"},
			{Seq: 2, Type: "assistant", Content: "Hi there"},
		},
	}

	err := WriteHistoryJSONL(path, history)
	if err != nil {
		t.Fatalf("WriteHistoryJSONL failed: %v", err)
	}

	// verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected file to exist")
	}

	// read and verify content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := splitJSONL(data)
	if len(lines) != 3 { // meta + 2 entries
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// verify first line is meta
	var metaWrapper struct {
		Meta *HistoryMeta `json:"_meta"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &metaWrapper); err != nil {
		t.Fatalf("failed to parse meta line: %v", err)
	}
	if metaWrapper.Meta == nil {
		t.Error("expected _meta in first line")
	}
	if metaWrapper.Meta.AgentID != "test-agent" {
		t.Errorf("expected agent_id=test-agent, got %s", metaWrapper.Meta.AgentID)
	}
}

func TestWriteHistoryJSONL_NilHistory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jsonl")

	err := WriteHistoryJSONL(path, nil)
	if err != ErrHistoryNilHistory {
		t.Errorf("expected ErrHistoryNilHistory, got %v", err)
	}
}

func TestMergeHistoryWithRecording_NilHistory(t *testing.T) {
	err := MergeHistoryWithRecording("/tmp/test", nil)
	if err != ErrHistoryNilHistory {
		t.Errorf("expected ErrHistoryNilHistory, got %v", err)
	}
}

func TestMergeHistoryWithRecording_EmptyEntries(t *testing.T) {
	history := &CapturedHistory{
		Meta:    &HistoryMeta{SchemaVersion: "1", Source: "test", AgentID: "test"},
		Entries: []HistoryEntry{},
	}
	err := MergeHistoryWithRecording("/tmp/test", history)
	if err != ErrHistoryEmptyEntries {
		t.Errorf("expected ErrHistoryEmptyEntries, got %v", err)
	}
}

func TestMergeHistoryWithRecording_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			Source:        "agent_reconstruction",
			AgentID:       "test-agent",
		},
		Entries: []HistoryEntry{
			{Seq: 1, Type: "user", Content: "Planning message"},
			{Seq: 2, Type: "assistant", Content: "Planning response"},
		},
	}

	err := MergeHistoryWithRecording(tmpDir, history)
	if err != nil {
		t.Fatalf("MergeHistoryWithRecording failed: %v", err)
	}

	// verify raw.jsonl was created
	rawPath := filepath.Join(tmpDir, "raw.jsonl")
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Fatal("expected raw.jsonl to be created")
	}

	// read and verify content
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := splitJSONL(data)
	// should have 2 entries + footer (no header when creating new)
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines, got %d", len(lines))
	}

	// verify entries have source=planning_history
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entryType, ok := entry["type"].(string); ok && entryType != "footer" {
			if source, ok := entry["source"].(string); ok {
				if source != "planning_history" {
					t.Errorf("expected source=planning_history, got %s", source)
				}
			}
		}
	}
}

func TestMergeHistoryWithRecording_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	rawPath := filepath.Join(tmpDir, "raw.jsonl")

	// create existing raw.jsonl with some entries
	existingContent := `{"type":"header","metadata":{"version":"1.0"}}
{"ts":"2024-01-01T10:00:00Z","type":"user","content":"Existing user message","seq":1}
{"ts":"2024-01-01T10:01:00Z","type":"assistant","content":"Existing assistant message","seq":2}
{"type":"footer","entry_count":2}
`
	if err := os.WriteFile(rawPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	history := &CapturedHistory{
		Meta: &HistoryMeta{
			SchemaVersion: "1",
			Source:        "agent_reconstruction",
			AgentID:       "test-agent",
		},
		Entries: []HistoryEntry{
			{Seq: 1, Type: "user", Content: "Prior planning", Timestamp: time.Now()},
			{Seq: 2, Type: "assistant", Content: "Prior response", Timestamp: time.Now()},
		},
	}

	err := MergeHistoryWithRecording(tmpDir, history)
	if err != nil {
		t.Fatalf("MergeHistoryWithRecording failed: %v", err)
	}

	// read merged file
	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := splitJSONL(data)

	// count entries (excluding header and footer)
	entryCount := 0
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entryType, ok := entry["type"].(string); ok {
			if entryType != "header" && entryType != "footer" {
				entryCount++
			}
		}
	}

	// should have 4 entries (2 history + 2 existing)
	if entryCount != 4 {
		t.Errorf("expected 4 entries, got %d", entryCount)
	}

	// verify seq numbers are adjusted
	seqNumbers := []int{}
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entryType, ok := entry["type"].(string); ok {
			if entryType != "header" && entryType != "footer" {
				if seq, ok := entry["seq"].(float64); ok {
					seqNumbers = append(seqNumbers, int(seq))
				}
			}
		}
	}

	// seq should be [1, 2, 3, 4] after merging
	expectedSeq := []int{1, 2, 3, 4}
	if len(seqNumbers) != len(expectedSeq) {
		t.Errorf("expected %d seq numbers, got %d", len(expectedSeq), len(seqNumbers))
	}
	for i, expected := range expectedSeq {
		if i < len(seqNumbers) && seqNumbers[i] != expected {
			t.Errorf("expected seq[%d]=%d, got %d", i, expected, seqNumbers[i])
		}
	}
}

func TestSessionHasHistory(t *testing.T) {
	tmpDir := t.TempDir()

	// no history file
	if SessionHasHistory(tmpDir) {
		t.Error("expected false when no history file exists")
	}

	// create history file
	historyPath := filepath.Join(tmpDir, "prior-history.jsonl")
	if err := os.WriteFile(historyPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}

	if !SessionHasHistory(tmpDir) {
		t.Error("expected true when history file exists")
	}
}

func TestGetHistoryStoragePath_NoActiveRecording(t *testing.T) {
	path := GetHistoryStoragePath("test-agent", false)
	if path == "" {
		t.Error("expected non-empty path")
	}
	if filepath.Base(path) != "prior-history.jsonl" {
		t.Errorf("expected filename prior-history.jsonl, got %s", filepath.Base(path))
	}
}

func TestGetHistoryStoragePath_EmptyAgentID(t *testing.T) {
	path := GetHistoryStoragePath("", false)
	if path == "" {
		t.Error("expected non-empty path even with empty agent ID")
	}
}

// splitJSONL splits JSONL data into lines, filtering empty lines.
func splitJSONL(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, string(data[start:i]))
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, string(data[start:]))
	}
	return lines
}
