package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionFilterMode_IsValid(t *testing.T) {
	tests := []struct {
		mode  SessionFilterMode
		valid bool
	}{
		{SessionFilterModeNone, true},
		{SessionFilterModeAll, true},
		{"", true}, // empty is valid (inherits)
		{"invalid", false},
		{"NONE", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.mode.IsValid())
		})
	}
}

func TestSessionFilterMode_ShouldRecord(t *testing.T) {
	tests := []struct {
		mode   SessionFilterMode
		record bool
	}{
		{SessionFilterModeNone, false},
		{SessionFilterModeAll, true},
		{"", false}, // empty = none
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.record, tt.mode.ShouldRecord())
		})
	}
}

func TestIsNoiseCommand(t *testing.T) {
	noiseCmds := []string{
		"ls",
		"ls -la",
		"pwd",
		"cat file.txt",
		"head -n 10",
		"echo hello",
	}

	meaningfulCmds := []string{
		"go build",
		"npm install",
		"git commit",
	}

	for _, cmd := range noiseCmds {
		t.Run(cmd, func(t *testing.T) {
			assert.True(t, IsNoiseCommand(cmd), "expected %q to be noise", cmd)
		})
	}

	for _, cmd := range meaningfulCmds {
		t.Run(cmd, func(t *testing.T) {
			assert.False(t, IsNoiseCommand(cmd), "expected %q to NOT be noise", cmd)
		})
	}
}

func TestFilterEvents_ModeNone(t *testing.T) {
	events := []ExtractedEvent{
		{Type: ExtractedEventUserAsked, Summary: "test"},
	}

	result := FilterEvents(events, SessionFilterModeNone)
	assert.Nil(t, result)

	result = FilterEvents(events, "")
	assert.Nil(t, result)
}

func TestFilterEvents_ModeAll(t *testing.T) {
	events := []ExtractedEvent{
		{Type: ExtractedEventUserAsked, Summary: "help me with docker"},
		{Type: ExtractedEventCommandRun, Summary: "ls -la"},         // noise
		{Type: ExtractedEventCommandRun, Summary: "docker build ."}, // meaningful
		{Type: ExtractedEventFileEdited, Summary: "edited main.go"},
		{Type: ExtractedEventErrorOccurred, Summary: "build failed"}, // errors never filtered
	}

	result := FilterEvents(events, SessionFilterModeAll)

	// should filter out "ls -la" but keep everything else
	assert.Len(t, result, 4)

	// verify ls command was filtered
	for _, e := range result {
		if e.Type == ExtractedEventCommandRun {
			assert.NotEqual(t, "ls -la", e.Summary)
		}
	}
}

func TestFilterEvents_PreservesErrors(t *testing.T) {
	events := []ExtractedEvent{
		{Type: ExtractedEventCommandRun, Summary: "ls"},           // noise, should be filtered
		{Type: ExtractedEventErrorOccurred, Summary: "ls failed"}, // error, should NOT be filtered
	}

	result := FilterEvents(events, SessionFilterModeAll)
	assert.Len(t, result, 1)
	assert.Equal(t, ExtractedEventErrorOccurred, result[0].Type)
}
