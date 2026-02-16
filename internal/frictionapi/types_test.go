package frictionapi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewFrictionEvent(t *testing.T) {
	event := NewFrictionEvent("unknown-command")

	if event.Kind != "unknown-command" {
		t.Errorf("Kind = %q, want %q", event.Kind, "unknown-command")
	}

	if event.Timestamp == "" {
		t.Error("Timestamp should be set")
	}

	// verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, event.Timestamp)
	if err != nil {
		t.Errorf("Timestamp is not valid RFC3339: %v", err)
	}
}

func TestFrictionEvent_Truncate(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		errorMsg     string
		wantInputLen int
		wantErrorLen int
	}{
		{
			name:         "short fields unchanged",
			input:        "short input",
			errorMsg:     "short error",
			wantInputLen: 11,
			wantErrorLen: 11,
		},
		{
			name:         "input truncated at max",
			input:        string(make([]byte, 600)),
			errorMsg:     "short",
			wantInputLen: MaxInputLength,
			wantErrorLen: 5,
		},
		{
			name:         "error truncated at max",
			input:        "short",
			errorMsg:     string(make([]byte, 300)),
			wantInputLen: 5,
			wantErrorLen: MaxErrorLength,
		},
		{
			name:         "both truncated",
			input:        string(make([]byte, 1000)),
			errorMsg:     string(make([]byte, 500)),
			wantInputLen: MaxInputLength,
			wantErrorLen: MaxErrorLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := FrictionEvent{
				Input:    tt.input,
				ErrorMsg: tt.errorMsg,
			}

			event.Truncate()

			if len(event.Input) != tt.wantInputLen {
				t.Errorf("Input length = %d, want %d", len(event.Input), tt.wantInputLen)
			}
			if len(event.ErrorMsg) != tt.wantErrorLen {
				t.Errorf("ErrorMsg length = %d, want %d", len(event.ErrorMsg), tt.wantErrorLen)
			}
		})
	}
}

func TestFrictionEvent_JSONSerialization(t *testing.T) {
	event := FrictionEvent{
		Timestamp:  "2024-01-15T10:30:00Z",
		Kind:       "unknown-command",
		Actor:      "human",
		PathBucket: "repo",
		Input:      "ox foo",
		ErrorMsg:   "unknown command",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// verify JSON field names
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["ts"] != "2024-01-15T10:30:00Z" {
		t.Errorf("ts = %v, want 2024-01-15T10:30:00Z", m["ts"])
	}
	if m["kind"] != "unknown-command" {
		t.Errorf("kind = %v, want unknown-command", m["kind"])
	}
	if m["actor"] != "human" {
		t.Errorf("actor = %v, want human", m["actor"])
	}
	if m["input"] != "ox foo" {
		t.Errorf("input = %v, want 'ox foo'", m["input"])
	}
	if m["error_msg"] != "unknown command" {
		t.Errorf("error_msg = %v, want 'unknown command'", m["error_msg"])
	}
	if m["path_bucket"] != "repo" {
		t.Errorf("path_bucket = %v, want 'repo'", m["path_bucket"])
	}
}

func TestFrictionEvent_AgentTypeOmittedWhenEmpty(t *testing.T) {
	event := FrictionEvent{
		Timestamp:  "2024-01-15T10:30:00Z",
		Kind:       "unknown-command",
		Actor:      "human",
		AgentType:  "", // empty
		PathBucket: "home",
		Input:      "test",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// agent_type field should be omitted
	if _, ok := m["agent_type"]; ok {
		t.Error("agent_type field should be omitted when empty")
	}
}

func TestFrictionEvent_AgentTypeIncludedWhenSet(t *testing.T) {
	event := FrictionEvent{
		Timestamp:  "2024-01-15T10:30:00Z",
		Kind:       "unknown-command",
		Actor:      "agent",
		AgentType:  "claude-code",
		PathBucket: "repo",
		Input:      "test",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["agent_type"] != "claude-code" {
		t.Errorf("agent_type = %v, want claude-code", m["agent_type"])
	}
}

func TestConstants(t *testing.T) {
	if MaxInputLength != 500 {
		t.Errorf("MaxInputLength = %d, want 500", MaxInputLength)
	}
	if MaxErrorLength != 200 {
		t.Errorf("MaxErrorLength = %d, want 200", MaxErrorLength)
	}
}

func TestCatalogData_JSONSerialization(t *testing.T) {
	t.Parallel()

	catalog := CatalogData{
		Version: "v2026-01-17-001",
		Commands: []CommandMapping{
			{
				Pattern:     "daemons list --every",
				Target:      "daemons show --all",
				Count:       127,
				Confidence:  0.95,
				Description: "Command renamed in v0.3.0",
			},
		},
		Tokens: []TokenMapping{
			{
				Pattern:    "prine",
				Target:     "prime",
				Kind:       "unknown-command",
				Count:      89,
				Confidence: 0.92,
			},
		},
	}

	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded CatalogData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Version != "v2026-01-17-001" {
		t.Errorf("Version = %q, want v2026-01-17-001", decoded.Version)
	}
	if len(decoded.Commands) != 1 {
		t.Fatalf("Commands length = %d, want 1", len(decoded.Commands))
	}
	if decoded.Commands[0].Pattern != "daemons list --every" {
		t.Errorf("Command pattern = %q, want 'daemons list --every'", decoded.Commands[0].Pattern)
	}
	if decoded.Commands[0].Confidence != 0.95 {
		t.Errorf("Command confidence = %f, want 0.95", decoded.Commands[0].Confidence)
	}
	if len(decoded.Tokens) != 1 {
		t.Fatalf("Tokens length = %d, want 1", len(decoded.Tokens))
	}
	if decoded.Tokens[0].Pattern != "prine" {
		t.Errorf("Token pattern = %q, want 'prine'", decoded.Tokens[0].Pattern)
	}
}

func TestCatalogData_EmptyFieldsOmitted(t *testing.T) {
	t.Parallel()

	catalog := CatalogData{
		Version: "v1",
		// no Commands or Tokens
	}

	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// commands and tokens should be omitted when empty
	if _, ok := m["commands"]; ok {
		t.Error("commands field should be omitted when empty")
	}
	if _, ok := m["tokens"]; ok {
		t.Error("tokens field should be omitted when empty")
	}
}

func TestFrictionResponse_JSONSerialization(t *testing.T) {
	t.Parallel()

	resp := FrictionResponse{
		Accepted: 5,
		Catalog: &CatalogData{
			Version: "v1",
			Tokens: []TokenMapping{
				{Pattern: "typo", Target: "correct"},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded FrictionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Accepted != 5 {
		t.Errorf("Accepted = %d, want 5", decoded.Accepted)
	}
	if decoded.Catalog == nil {
		t.Fatal("Catalog should not be nil")
	}
	if decoded.Catalog.Version != "v1" {
		t.Errorf("Catalog.Version = %q, want v1", decoded.Catalog.Version)
	}
}

func TestFrictionResponse_CatalogOmittedWhenNil(t *testing.T) {
	t.Parallel()

	resp := FrictionResponse{
		Accepted: 3,
		Catalog:  nil,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// catalog should be omitted when nil
	if _, ok := m["catalog"]; ok {
		t.Error("catalog field should be omitted when nil")
	}
}

func TestCommandMapping_JSONSerialization(t *testing.T) {
	t.Parallel()

	cmd := CommandMapping{
		Pattern:     "old command",
		Target:      "new command",
		Count:       100,
		Confidence:  0.85,
		Description: "test description",
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded CommandMapping
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Pattern != "old command" {
		t.Errorf("Pattern = %q, want 'old command'", decoded.Pattern)
	}
	if decoded.Target != "new command" {
		t.Errorf("Target = %q, want 'new command'", decoded.Target)
	}
	if decoded.Count != 100 {
		t.Errorf("Count = %d, want 100", decoded.Count)
	}
	if decoded.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want 0.85", decoded.Confidence)
	}
	if decoded.Description != "test description" {
		t.Errorf("Description = %q, want 'test description'", decoded.Description)
	}
}

func TestTokenMapping_JSONSerialization(t *testing.T) {
	t.Parallel()

	token := TokenMapping{
		Pattern:    "typo",
		Target:     "correct",
		Kind:       "unknown-flag",
		Count:      50,
		Confidence: 0.75,
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded TokenMapping
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Pattern != "typo" {
		t.Errorf("Pattern = %q, want 'typo'", decoded.Pattern)
	}
	if decoded.Target != "correct" {
		t.Errorf("Target = %q, want 'correct'", decoded.Target)
	}
	if decoded.Kind != "unknown-flag" {
		t.Errorf("Kind = %q, want 'unknown-flag'", decoded.Kind)
	}
	if decoded.Count != 50 {
		t.Errorf("Count = %d, want 50", decoded.Count)
	}
	if decoded.Confidence != 0.75 {
		t.Errorf("Confidence = %f, want 0.75", decoded.Confidence)
	}
}
