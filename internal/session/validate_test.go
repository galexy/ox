package session

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONL_ValidHistory(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"agent_reconstruction","agent_id":"Oxa7b3"}}
{"seq":1,"type":"user","content":"Hello","ts":"2026-01-16T10:00:00Z"}
{"seq":2,"type":"assistant","content":"Hi there!","ts":"2026-01-16T10:01:00Z"}`

	history, err := ValidateJSONL(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history.Meta)
	assert.Equal(t, "1", history.Meta.SchemaVersion)
	assert.Equal(t, "Oxa7b3", history.Meta.AgentID)
	assert.Len(t, history.Entries, 2)
}

func TestValidateJSONL_MissingMeta(t *testing.T) {
	// when first line is an entry without _meta, validation fails
	jsonl := `{"seq":1,"type":"user","content":"Hello"}`

	_, err := ValidateJSONL(strings.NewReader(jsonl))

	require.Error(t, err)
	// error message indicates missing required fields in _meta
	assert.Contains(t, err.Error(), "_meta")
}

func TestValidateJSONL_InvalidEntryType(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"invalid","content":"Hello"}`

	_, err := ValidateJSONL(strings.NewReader(jsonl))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidateJSONL_NonMonotonicSeq(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":5,"type":"user","content":"First"}
{"seq":3,"type":"assistant","content":"Second"}`

	_, err := ValidateJSONL(strings.NewReader(jsonl))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "monotonically increasing")
}

func TestValidateJSONL_EmptyReader(t *testing.T) {
	_, err := ValidateJSONL(strings.NewReader(""))

	require.Error(t, err)
	// empty input causes missing _meta line error
	assert.Contains(t, err.Error(), "missing _meta")
}

func TestValidateJSONL_NilReader(t *testing.T) {
	_, err := ValidateJSONL(nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrHistoryEmptyInput)
}

func TestValidateAndRedact_RedactsSecrets(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"user","content":"My AWS key is AKIAIOSFODNN7EXAMPLE"}`

	history, redactedCount, err := ValidateAndRedact(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history)
	assert.Equal(t, 1, redactedCount)
	assert.Contains(t, history.Entries[0].Content, "[REDACTED_AWS_KEY]")
	assert.NotContains(t, history.Entries[0].Content, "AKIAIOSFODNN7EXAMPLE")
}

func TestValidateAndRedact_NoSecrets(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"user","content":"Hello, how are you?"}`

	history, redactedCount, err := ValidateAndRedact(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history)
	assert.Equal(t, 0, redactedCount)
	assert.Equal(t, "Hello, how are you?", history.Entries[0].Content)
}

func TestValidateAndRedact_ValidationError(t *testing.T) {
	jsonl := `{"seq":1,"type":"user","content":"No meta"}`

	_, _, err := ValidateAndRedact(strings.NewReader(jsonl))

	require.Error(t, err)
}

func TestValidateCapturePriorInput_Valid(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"agent_reconstruction","agent_id":"Oxa7b3"}}
{"seq":1,"type":"user","content":"Plan something","ts":"2026-01-16T10:00:00Z"}
{"seq":2,"type":"assistant","content":"Here is the plan","ts":"2026-01-16T10:30:00Z"}`

	history, err := ValidateCapturePriorInput(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history)
	assert.Equal(t, 2, len(history.Entries))
	assert.Equal(t, 2, history.Meta.MessageCount)
	require.NotNil(t, history.Meta.TimeRange)
}

func TestValidateCapturePriorInput_EmptyEntries(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"agent_reconstruction","agent_id":"Oxa7b3"}}`

	_, err := ValidateCapturePriorInput(strings.NewReader(jsonl))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one entry")
}

func TestValidateCapturePriorInput_ComputesTimeRange(t *testing.T) {
	// create timestamps
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	later := time.Now().Format(time.RFC3339)

	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"user","content":"Earlier","ts":"` + earlier + `"}
{"seq":2,"type":"assistant","content":"Later","ts":"` + later + `"}`

	history, err := ValidateCapturePriorInput(strings.NewReader(jsonl))

	require.NoError(t, err)
	require.NotNil(t, history.Meta.TimeRange)
	assert.False(t, history.Meta.TimeRange.Earliest.IsZero())
	assert.False(t, history.Meta.TimeRange.Latest.IsZero())
	assert.True(t, history.Meta.TimeRange.Earliest.Before(history.Meta.TimeRange.Latest))
}

func TestValidateCapturePriorInput_SetsMessageCount(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"user","content":"One"}
{"seq":2,"type":"assistant","content":"Two"}
{"seq":3,"type":"user","content":"Three"}`

	history, err := ValidateCapturePriorInput(strings.NewReader(jsonl))

	require.NoError(t, err)
	assert.Equal(t, 3, history.Meta.MessageCount)
}

func TestValidateJSONL_ToolEntries(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"tool","content":"output","tool_name":"Bash","tool_input":"ls -la"}`

	history, err := ValidateJSONL(strings.NewReader(jsonl))

	require.NoError(t, err)
	assert.Len(t, history.Entries, 1)
	assert.Equal(t, "tool", history.Entries[0].Type)
	assert.Equal(t, "Bash", history.Entries[0].ToolName)
	assert.Equal(t, "ls -la", history.Entries[0].ToolInput)
}

func TestValidateJSONL_AllEntryTypes(t *testing.T) {
	jsonl := `{"_meta":{"schema_version":"1","source":"test","agent_id":"test"}}
{"seq":1,"type":"user","content":"User message"}
{"seq":2,"type":"assistant","content":"Assistant response"}
{"seq":3,"type":"system","content":"System context"}
{"seq":4,"type":"tool","content":"Tool output","tool_name":"Read"}`

	history, err := ValidateJSONL(strings.NewReader(jsonl))

	require.NoError(t, err)
	assert.Len(t, history.Entries, 4)
	assert.Equal(t, "user", history.Entries[0].Type)
	assert.Equal(t, "assistant", history.Entries[1].Type)
	assert.Equal(t, "system", history.Entries[2].Type)
	assert.Equal(t, "tool", history.Entries[3].Type)
}
