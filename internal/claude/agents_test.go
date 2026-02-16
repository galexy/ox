package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAgents_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	agents, err := DiscoverAgents(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestDiscoverAgents_WithAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// create agents directory
	agentsDir := filepath.Join(tmpDir, ClaudeDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// create an agent file with frontmatter
	agentContent := `---
description: "Test agent for code review"
model: "opus"
---

# test-agent

A test agent for reviewing code.
`
	if err := os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte(agentContent), 0644); err != nil {
		t.Fatal(err)
	}

	agents, err := DiscoverAgents(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}

	agent := agents[0]
	if agent.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", agent.Name)
	}
	if agent.Description != "Test agent for code review" {
		t.Errorf("expected description 'Test agent for code review', got %q", agent.Description)
	}
	if agent.Model != "opus" {
		t.Errorf("expected model 'opus', got %q", agent.Model)
	}
}

func TestDiscoverAgents_WithIndex(t *testing.T) {
	tmpDir := t.TempDir()

	agentsDir := filepath.Join(tmpDir, ClaudeDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// create index.md with token-optimized descriptions
	indexContent := `# Agents

- **my-agent**: Optimized description from index
`
	if err := os.WriteFile(filepath.Join(agentsDir, "index.md"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	// create agent file (description should be overridden by index)
	agentContent := `---
description: "Long description in file"
---

# my-agent

Some content.
`
	if err := os.WriteFile(filepath.Join(agentsDir, "my-agent.md"), []byte(agentContent), 0644); err != nil {
		t.Fatal(err)
	}

	agents, err := DiscoverAgents(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}

	agent := agents[0]
	if agent.Description != "Optimized description from index" {
		t.Errorf("expected index description, got %q", agent.Description)
	}
	if !agent.FromIndex {
		t.Error("expected FromIndex to be true")
	}
}

func TestDiscoverAll(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ClaudeDir)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// create CLAUDE.md
	if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# Team Claude Config"), 0644); err != nil {
		t.Fatal(err)
	}

	// create AGENTS.md
	if err := os.WriteFile(filepath.Join(claudeDir, "AGENTS.md"), []byte("# Team Agents"), 0644); err != nil {
		t.Fatal(err)
	}

	tc, err := DiscoverAll(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tc.HasClaudeMD {
		t.Error("expected HasClaudeMD to be true")
	}
	if !tc.HasAgentsMD {
		t.Error("expected HasAgentsMD to be true")
	}
	if tc.ClaudeMDPath != filepath.Join(claudeDir, "CLAUDE.md") {
		t.Errorf("unexpected ClaudeMDPath: %s", tc.ClaudeMDPath)
	}
	if !tc.HasInstructionFiles() {
		t.Error("expected HasInstructionFiles to return true")
	}
}

func TestParseIndex(t *testing.T) {
	tmpDir := t.TempDir()

	indexContent := `# Index

- **agent-one**: First agent description
- **agent-two**: Second agent description
* **agent-three** - Third with dash separator
`
	indexPath := filepath.Join(tmpDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := ParseIndex(indexPath)

	if result["agent-one"] != "First agent description" {
		t.Errorf("agent-one: got %q", result["agent-one"])
	}
	if result["agent-two"] != "Second agent description" {
		t.Errorf("agent-two: got %q", result["agent-two"])
	}
	if result["agent-three"] != "Third with dash separator" {
		t.Errorf("agent-three: got %q", result["agent-three"])
	}
}
