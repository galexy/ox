package main

import (
	"strings"
	"testing"
)

func TestIsAgentSupported(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		want      bool
	}{
		{
			name:      "claude-code is supported",
			agentType: "claude-code",
			want:      true,
		},
		{
			name:      "cursor is not supported",
			agentType: "cursor",
			want:      false,
		},
		{
			name:      "windsurf is not supported",
			agentType: "windsurf",
			want:      false,
		},
		{
			name:      "droid is not supported",
			agentType: "droid",
			want:      false,
		},
		{
			name:      "empty agent type is not supported",
			agentType: "",
			want:      false,
		},
		{
			name:      "unknown agent is not supported",
			agentType: "unknown-agent",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAgentSupported(tt.agentType)
			if got != tt.want {
				t.Errorf("isAgentSupported(%q) = %v, want %v", tt.agentType, got, tt.want)
			}
		})
	}
}

func TestGetAgentSupportNotice(t *testing.T) {
	tests := []struct {
		name         string
		agentType    string
		wantEmpty    bool
		wantContains string
	}{
		{
			name:      "claude-code returns empty notice",
			agentType: "claude-code",
			wantEmpty: true,
		},
		{
			name:         "cursor returns notice with display name",
			agentType:    "cursor",
			wantEmpty:    false,
			wantContains: "Cursor",
		},
		{
			name:         "empty agent returns notice about this agent",
			agentType:    "",
			wantEmpty:    false,
			wantContains: "this agent",
		},
		{
			name:         "windsurf returns notice with display name",
			agentType:    "windsurf",
			wantEmpty:    false,
			wantContains: "Windsurf",
		},
		{
			name:         "unsupported agent notice explains review needed",
			agentType:    "cursor",
			wantEmpty:    false,
			wantContains: "review plans deeply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAgentSupportNotice(tt.agentType)

			if tt.wantEmpty && got != "" {
				t.Errorf("getAgentSupportNotice(%q) = %q, want empty", tt.agentType, got)
			}

			if !tt.wantEmpty && got == "" {
				t.Errorf("getAgentSupportNotice(%q) = empty, want non-empty", tt.agentType)
			}

			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("getAgentSupportNotice(%q) = %q, want to contain %q", tt.agentType, got, tt.wantContains)
			}
		})
	}
}
