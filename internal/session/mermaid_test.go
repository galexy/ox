package session

import (
	"strings"
	"testing"
)

func TestHasMermaidBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "no mermaid",
			content: "Hello world\n```go\nfunc main() {}\n```",
			want:    false,
		},
		{
			name:    "has mermaid",
			content: "Here's a diagram:\n```mermaid\ngraph LR\n  A --> B\n```",
			want:    true,
		},
		{
			name:    "mermaid with extra whitespace",
			content: "```mermaid\n\ngraph TD\n  X --> Y\n```",
			want:    true,
		},
		{
			name:    "empty content",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasMermaidBlocks(tt.content)
			if got != tt.want {
				t.Errorf("HasMermaidBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessMermaidBlocks_NoMermaid(t *testing.T) {
	content := "Hello world\n```go\nfunc main() {}\n```"
	result := ProcessMermaidBlocks(content)

	if result != content {
		t.Errorf("ProcessMermaidBlocks() modified content without mermaid blocks")
	}
}

func TestProcessMermaidBlocks_WithMermaid(t *testing.T) {
	content := "Here's a diagram:\n```mermaid\ngraph LR\n  A --> B\n```\nEnd."

	result := ProcessMermaidBlocks(content)

	// should have replaced mermaid block
	if strings.Contains(result, "```mermaid") {
		t.Error("ProcessMermaidBlocks() did not replace mermaid block")
	}

	// should still have surrounding text
	if !strings.Contains(result, "Here's a diagram:") {
		t.Error("ProcessMermaidBlocks() lost prefix text")
	}
	if !strings.Contains(result, "End.") {
		t.Error("ProcessMermaidBlocks() lost suffix text")
	}

	// should contain ASCII diagram elements
	if !strings.Contains(result, "A") || !strings.Contains(result, "B") {
		t.Error("ProcessMermaidBlocks() output doesn't contain expected nodes")
	}
}

func TestRenderMermaidToASCII(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple graph",
			input:   "graph LR\n  A --> B",
			wantErr: false,
		},
		{
			name:    "sequence diagram",
			input:   "sequenceDiagram\n  A->>B: Hello",
			wantErr: false,
		},
		{
			name:    "invalid syntax",
			input:   "not a valid mermaid diagram",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderMermaidToASCII(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("RenderMermaidToASCII() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RenderMermaidToASCII() unexpected error: %v", err)
				}
				if result == "" {
					t.Error("RenderMermaidToASCII() returned empty result")
				}
			}
		})
	}
}
