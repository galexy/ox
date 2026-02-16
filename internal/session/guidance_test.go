package session

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartGuidance(t *testing.T) {
	g := StartGuidance()

	assert.NotEmpty(t, g.Include, "StartGuidance should have include items")
	assert.NotEmpty(t, g.Exclude, "StartGuidance should have exclude items")
	assert.NotEmpty(t, g.Tips, "StartGuidance should have tips")
	assert.Equal(t, DefaultReminderInterval, g.ReminderInterval)
}

func TestStopGuidance(t *testing.T) {
	g := StopGuidance()

	assert.NotEmpty(t, g.Include, "StopGuidance should have include items")
	assert.NotEmpty(t, g.Exclude, "StopGuidance should have exclude items")
}

func TestRemindGuidance(t *testing.T) {
	g := RemindGuidance()

	assert.NotEmpty(t, g.Include, "RemindGuidance should have include items")
	assert.NotEmpty(t, g.Exclude, "RemindGuidance should have exclude items")
	// remind guidance should be more condensed than start
	startG := StartGuidance()
	assert.Less(t, len(g.Include), len(startG.Include), "RemindGuidance should be more condensed than StartGuidance")
}

func TestFormatGuidanceText(t *testing.T) {
	tests := []struct {
		name    string
		phase   GuidancePhase
		agentID string
		want    []string // strings that should be present
	}{
		{
			name:    "start phase",
			phase:   GuidancePhaseStart,
			agentID: "Ox1234",
			want:    []string{"Session Recording Started", "INCLUDE", "EXCLUDE", "Ox1234", "session remind"},
		},
		{
			name:    "stop phase",
			phase:   GuidancePhaseStop,
			agentID: "Ox5678",
			want:    []string{"Session Recording Stopping", "INCLUDE", "EXCLUDE"},
		},
		{
			name:    "remind phase",
			phase:   GuidancePhaseRemind,
			agentID: "OxABCD",
			want:    []string{"Session Recording Reminder", "INCLUDE", "EXCLUDE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var guidance SessionGuidance
			switch tt.phase {
			case GuidancePhaseStart:
				guidance = StartGuidance()
			case GuidancePhaseStop:
				guidance = StopGuidance()
			case GuidancePhaseRemind:
				guidance = RemindGuidance()
			}

			result := FormatGuidanceText(tt.phase, tt.agentID, guidance)

			for _, want := range tt.want {
				assert.Contains(t, result, want)
			}
		})
	}
}

func TestFormatGuidanceJSON(t *testing.T) {
	guidance := StartGuidance()
	agentID := "Ox1234"

	output, err := FormatGuidanceJSON(GuidancePhaseStart, agentID, guidance, "test message")
	require.NoError(t, err)

	// verify it's valid JSON
	var result GuidanceOutput
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "session_guidance", result.Type)
	assert.Equal(t, GuidancePhaseStart, result.Phase)
	assert.Equal(t, agentID, result.AgentID)
	assert.Equal(t, "test message", result.Message)
	assert.NotEmpty(t, result.Guidance.Include)
}

func TestGetSummarizeGuidance(t *testing.T) {
	agentID := "Ox1234"
	contextPath := "/home/user/code/myrepo_sageox_ledger"

	g := GetSummarizeGuidance(agentID, contextPath)

	assert.NotEmpty(t, g.Instructions, "GetSummarizeGuidance should have instructions")
	assert.NotEmpty(t, g.Format, "GetSummarizeGuidance should have format template")
	assert.Contains(t, g.SavePath, contextPath)
	assert.Equal(t, contextPath, g.LedgerPath)
	// check delegation hint
	require.NotNil(t, g.DelegationHint, "GetSummarizeGuidance should have delegation hint")
	assert.True(t, g.DelegationHint.Recommended, "delegation should be recommended for summarize")
	assert.NotEmpty(t, g.DelegationHint.SubagentType, "delegation should specify subagent type")
	assert.NotEmpty(t, g.DelegationHint.ModelTier, "delegation should specify model tier")
}

func TestFormatSummarizeGuidanceText(t *testing.T) {
	agentID := "Ox1234"
	guidance := GetSummarizeGuidance(agentID, "/test/path")

	result := FormatSummarizeGuidanceText(agentID, guidance)

	expectedStrings := []string{
		"Summary Guidance",
		"DELEGATION RECOMMENDED",
		"Subagent:",
		"Model:",
		"Background:",
		"Instructions:",
		"Recommended format:",
		"Save to:",
		"Ledger:",
	}

	for _, want := range expectedStrings {
		assert.Contains(t, result, want)
	}
}

func TestGetHTMLGuidance(t *testing.T) {
	agentID := "Ox1234"
	rawPath := "/path/to/session.jsonl"
	outputPath := "/path/to/session.html"

	g := GetHTMLGuidance(agentID, rawPath, outputPath)

	assert.NotEmpty(t, g.Instructions, "GetHTMLGuidance should have instructions")
	assert.NotEmpty(t, g.Features, "GetHTMLGuidance should have features")
	assert.Equal(t, rawPath, g.SourcePath)
	assert.Equal(t, outputPath, g.OutputPath)
	// check delegation hint
	require.NotNil(t, g.DelegationHint, "GetHTMLGuidance should have delegation hint")
	assert.True(t, g.DelegationHint.Recommended, "delegation should be recommended for HTML generation")
	assert.Equal(t, "fast", g.DelegationHint.ModelTier, "HTML generation should use fast model")
	assert.True(t, g.DelegationHint.RunInBackground, "HTML generation should run in background")
}

func TestFormatHTMLGuidanceText(t *testing.T) {
	agentID := "Ox1234"
	guidance := GetHTMLGuidance(agentID, "/source.jsonl", "/output.html")

	result := FormatHTMLGuidanceText(agentID, guidance)

	expectedStrings := []string{
		"HTML Session Viewer",
		"DELEGATION RECOMMENDED",
		"Subagent:",
		"Model:",
		"Background:",
		"Instructions:",
		"Source:",
		"Output:",
		"Features included:",
	}

	for _, want := range expectedStrings {
		assert.Contains(t, result, want)
	}
}

func TestFormatHTMLGuidanceJSON(t *testing.T) {
	agentID := "Ox1234"
	guidance := GetHTMLGuidance(agentID, "/source.jsonl", "/output.html")

	output, err := FormatHTMLGuidanceJSON(agentID, guidance, true, "/output.html", "HTML viewer exists")
	require.NoError(t, err)

	var result HTMLGuidanceOutput
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "session_html_guidance", result.Type)
	assert.True(t, result.Generated)
	assert.Equal(t, "/output.html", result.HTMLPath)
}

func TestFormatSummaryText(t *testing.T) {
	agentID := "Ox1234"
	summary := &SummarizeResponse{
		Summary:     "This session focused on implementing new features",
		KeyActions:  []string{"Added new API", "Fixed bug"},
		Outcome:     "success",
		TopicsFound: []string{"API", "testing"},
	}

	result := FormatSummaryText(agentID, summary, 100)

	expectedStrings := []string{
		"Session Summary",
		agentID,
		"Entries: 100",
		"Summary:",
		"This session focused on implementing",
		"Key Actions:",
		"Added new API",
		"Outcome: success",
		"Topics:",
	}

	for _, want := range expectedStrings {
		assert.Contains(t, result, want, "FormatSummaryText() missing %q in output", want)
	}
}

func TestDefaultReminderInterval(t *testing.T) {
	assert.Equal(t, 50, DefaultReminderInterval)
}
