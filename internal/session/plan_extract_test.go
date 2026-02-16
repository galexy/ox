package session

import (
	"strings"
	"testing"
	"time"
)

func TestExtractPlan(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		entries        []SessionEntry
		wantNil        bool
		wantMarker     PlanMarker
		wantIsPlanFlag bool
		wantContent    string
	}{
		{
			name:    "empty entries slice",
			entries: []SessionEntry{},
			wantNil: true,
		},
		{
			name: "entry with is_plan:true marker",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Create a plan"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Here is the plan with is_plan:true marker\n## Plan\n1. Step one"},
			},
			wantNil:        false,
			wantMarker:     PlanMarkerIsPlan,
			wantIsPlanFlag: true,
			wantContent:    "is_plan:true",
		},
		{
			name: "entry with ## Plan header",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "What should we do?"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Plan\n\n1. First step\n2. Second step"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerPlan,
			wantContent: "## Plan",
		},
		{
			name: "entry with ## Final Plan header",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Finalize the approach"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Final Plan\n\nHere is what we will do"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerFinalPlan,
			wantContent: "## Final Plan",
		},
		{
			name: "entry with ## Implementation Plan header",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "How will you implement this?"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Implementation Plan\n\n1. Create files\n2. Write tests"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerImplementationPlan,
			wantContent: "## Implementation Plan",
		},
		{
			name: "no plan markers - returns last assistant message",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Hello"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Hi there! How can I help?"},
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Tell me something"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Here is some information for you"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerNone,
			wantContent: "Here is some information",
		},
		{
			name: "multiple entries with markers - returns last one with highest priority",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Let us plan"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Plan\n\nInitial plan"},
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Revise it"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Plan\n\nRevised plan"},
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Finalize"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Final Plan\n\nThe final approach"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerFinalPlan,
			wantContent: "The final approach",
		},
		{
			name: "is_plan takes priority over Final Plan",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## Final Plan\n\nOne approach"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "is_plan:true\n\nThis is the actual plan"},
			},
			wantNil:        false,
			wantMarker:     PlanMarkerIsPlan,
			wantIsPlanFlag: true,
			wantContent:    "actual plan",
		},
		{
			name: "only user messages - returns nil",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Hello"},
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Anyone there?"},
			},
			wantNil: true,
		},
		{
			name: "system and tool entries ignored",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeSystem, Content: "## Plan\n\nSystem message with plan"},
				{Timestamp: now, Type: SessionEntryTypeTool, Content: "## Plan\n\nTool output with plan"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Just a regular response"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerNone,
			wantContent: "Just a regular response",
		},
		{
			name: "case insensitive plan detection",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "## PLAN\n\nUppercase header"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerPlan,
			wantContent: "## PLAN",
		},
		{
			name: "plan header not at line start still detected",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Here is my analysis.\n\n## Plan\n\n1. Do this\n2. Do that"},
			},
			wantNil:     false,
			wantMarker:  PlanMarkerPlan,
			wantContent: "## Plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPlan(tt.entries)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractPlan() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("ExtractPlan() = nil, want non-nil")
			}

			if got.Marker != tt.wantMarker {
				t.Errorf("ExtractPlan().Marker = %v, want %v", got.Marker, tt.wantMarker)
			}

			if got.IsPlanFlag != tt.wantIsPlanFlag {
				t.Errorf("ExtractPlan().IsPlanFlag = %v, want %v", got.IsPlanFlag, tt.wantIsPlanFlag)
			}

			if tt.wantContent != "" {
				if !strings.Contains(got.Entry.Content, tt.wantContent) {
					t.Errorf("ExtractPlan().Entry.Content does not contain %q, got %q", tt.wantContent, got.Entry.Content)
				}
			}
		})
	}
}

func TestExtractMermaidDiagrams(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "single mermaid block",
			content: "Here is a diagram:\n\n```mermaid\ngraph LR\n  A --> B\n```\n\nEnd of content.",
			want:    []string{"graph LR\n  A --> B"},
		},
		{
			name:    "multiple mermaid blocks",
			content: "First:\n```mermaid\ngraph TD\n  Start --> End\n```\nSecond:\n```mermaid\nsequenceDiagram\n  Alice->>Bob: Hello\n```",
			want: []string{
				"graph TD\n  Start --> End",
				"sequenceDiagram\n  Alice->>Bob: Hello",
			},
		},
		{
			name:    "no mermaid blocks",
			content: "Just regular text without any diagrams.\n\n```go\nfunc main() {}\n```",
			want:    nil,
		},
		{
			name:    "malformed block - unclosed",
			content: "Here is an unclosed block:\n\n```mermaid\ngraph LR\n  A --> B\nNo closing backticks here",
			want:    nil,
		},
		{
			name:    "mixed content with mermaid",
			content: "# Overview\n\n```mermaid\nflowchart TD\n  User --> API\n```\n\n```go\nfunc main() {}\n```\n\n```mermaid\nstateDiagram-v2\n  [*] --> Ready\n```",
			want: []string{
				"flowchart TD\n  User --> API",
				"stateDiagram-v2\n  [*] --> Ready",
			},
		},
		{
			name:    "empty mermaid block",
			content: "```mermaid\n```",
			want:    nil,
		},
		{
			name:    "mermaid block with only whitespace",
			content: "```mermaid\n\n\n```",
			want:    nil,
		},
		{
			name:    "mermaid with extra whitespace in opening",
			content: "```mermaid  \ngraph LR\n  X --> Y\n```",
			want:    []string{"graph LR\n  X --> Y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMermaidDiagrams(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("ExtractMermaidDiagrams() returned %d diagrams, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i, diagram := range got {
				if diagram != tt.want[i] {
					t.Errorf("ExtractMermaidDiagrams()[%d] = %q, want %q", i, diagram, tt.want[i])
				}
			}
		})
	}
}

func TestDetectPlanMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    PlanMarker
	}{
		{
			name:    "empty content",
			content: "",
			want:    PlanMarkerNone,
		},
		{
			name:    "## Plan header",
			content: "## Plan\n\n1. First step",
			want:    PlanMarkerPlan,
		},
		{
			name:    "## Implementation Plan header",
			content: "Some intro\n\n## Implementation Plan\n\nDetails here",
			want:    PlanMarkerImplementationPlan,
		},
		{
			name:    "## Final Plan header",
			content: "## Final Plan\n\nThis is the final approach",
			want:    PlanMarkerFinalPlan,
		},
		{
			name:    "no markers",
			content: "Just regular content without any plan headers",
			want:    PlanMarkerNone,
		},
		{
			name:    "case insensitivity - lowercase",
			content: "## plan\n\nLowercase header",
			want:    PlanMarkerPlan,
		},
		{
			name:    "case insensitivity - uppercase",
			content: "## PLAN\n\nUppercase header",
			want:    PlanMarkerPlan,
		},
		{
			name:    "case insensitivity - mixed case",
			content: "## PlAn\n\nMixed case header",
			want:    PlanMarkerPlan,
		},
		{
			name:    "## Final Plan case insensitive",
			content: "## FINAL PLAN\n\nUppercase",
			want:    PlanMarkerFinalPlan,
		},
		{
			name:    "## Implementation Plan case insensitive",
			content: "## implementation plan\n\nLowercase",
			want:    PlanMarkerImplementationPlan,
		},
		{
			name:    "plan word in content but not header",
			content: "This is a plan for the project but no header",
			want:    PlanMarkerNone,
		},
		{
			name:    "# Plan - single hash not matched",
			content: "# Plan\n\nSingle hash header",
			want:    PlanMarkerNone,
		},
		{
			name:    "### Plan - triple hash not matched",
			content: "### Plan\n\nTriple hash header",
			want:    PlanMarkerNone,
		},
		{
			name:    "## Plan with extra text after",
			content: "## Plan for Implementation\n\nWith suffix",
			want:    PlanMarkerPlan,
		},
		{
			name:    "## Planning - not matched (different word)",
			content: "## Planning\n\nDifferent word",
			want:    PlanMarkerNone,
		},
		{
			name:    "header with leading whitespace",
			content: "  ## Plan\n\nWith leading spaces",
			want:    PlanMarkerPlan,
		},
		{
			name:    "multiple headers - first one wins",
			content: "## Plan\n\nFirst\n\n## Final Plan\n\nSecond",
			want:    PlanMarkerPlan,
		},
		{
			name:    "## Final Plan appears first",
			content: "## Final Plan\n\nThis is final\n\n## Plan\n\nThis is generic",
			want:    PlanMarkerFinalPlan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPlanMarkers(tt.content)
			if got != tt.want {
				t.Errorf("DetectPlanMarkers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractAllMermaidFromEntries(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		entries []SessionEntry
		want    []string
	}{
		{
			name:    "empty entries",
			entries: []SessionEntry{},
			want:    nil,
		},
		{
			name: "single entry with diagram",
			entries: []SessionEntry{
				{
					Timestamp: now,
					Type:      SessionEntryTypeAssistant,
					Content:   "```mermaid\ngraph LR\n  A --> B\n```",
				},
			},
			want: []string{"graph LR\n  A --> B"},
		},
		{
			name: "multiple entries with diagrams",
			entries: []SessionEntry{
				{
					Timestamp: now,
					Type:      SessionEntryTypeAssistant,
					Content:   "First:\n```mermaid\ngraph TD\n  X --> Y\n```",
				},
				{
					Timestamp: now,
					Type:      SessionEntryTypeAssistant,
					Content:   "Second:\n```mermaid\nsequenceDiagram\n  A->>B: Hi\n```",
				},
			},
			want: []string{
				"graph TD\n  X --> Y",
				"sequenceDiagram\n  A->>B: Hi",
			},
		},
		{
			name: "duplicate diagrams are deduplicated",
			entries: []SessionEntry{
				{
					Timestamp: now,
					Type:      SessionEntryTypeAssistant,
					Content:   "```mermaid\ngraph LR\n  A --> B\n```",
				},
				{
					Timestamp: now,
					Type:      SessionEntryTypeAssistant,
					Content:   "Same diagram:\n```mermaid\ngraph LR\n  A --> B\n```",
				},
			},
			want: []string{"graph LR\n  A --> B"},
		},
		{
			name: "entries without mermaid",
			entries: []SessionEntry{
				{Timestamp: now, Type: SessionEntryTypeUser, Content: "Hello"},
				{Timestamp: now, Type: SessionEntryTypeAssistant, Content: "Hi there"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAllMermaidFromEntries(tt.entries)

			if len(got) != len(tt.want) {
				t.Errorf("ExtractAllMermaidFromEntries() returned %d diagrams, want %d", len(got), len(tt.want))
				return
			}

			for i, diagram := range got {
				if diagram != tt.want[i] {
					t.Errorf("ExtractAllMermaidFromEntries()[%d] = %q, want %q", i, diagram, tt.want[i])
				}
			}
		})
	}
}

func TestHasIsPlanMarker(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "is_plan:true no space",
			content: "Content with is_plan:true marker",
			want:    true,
		},
		{
			name:    "is_plan: true with space",
			content: "Content with is_plan: true marker",
			want:    true,
		},
		{
			name:    "JSON format no space",
			content: `{"is_plan":true, "content": "..."}`,
			want:    true,
		},
		{
			name:    "JSON format with space",
			content: `{"is_plan": true, "content": "..."}`,
			want:    true,
		},
		{
			name:    "no is_plan marker",
			content: "Regular content without any marker",
			want:    false,
		},
		{
			name:    "is_plan:false not matched",
			content: "Content with is_plan:false marker",
			want:    false,
		},
		{
			name:    "case insensitive",
			content: "Content with IS_PLAN:TRUE marker",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasIsPlanMarker(tt.content)
			if got != tt.want {
				t.Errorf("hasIsPlanMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}
