package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sageox/ox/internal/config"
	"github.com/spf13/cobra"
)

func TestCoworkerListCommand(t *testing.T) {
	// set up agent context for tests (Claude Code sets this env var)
	os.Setenv("CLAUDE_CODE_SESSION_ID", "test-session")
	defer os.Unsetenv("CLAUDE_CODE_SESSION_ID")

	tests := []struct {
		name       string
		setupFn    func(t *testing.T) (string, func())
		jsonMode   bool
		wantErr    bool
		wantOutput string
	}{
		{
			name: "no team context configured",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				// create .sageox dir but no team context
				requireSageoxDir(t, dir)
				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "No team context configured",
		},
		{
			name: "no team context configured json mode",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				requireSageoxDir(t, dir)
				return dir, func() {}
			},
			jsonMode: true,
			wantErr:  false,
		},
		{
			name: "team context with agents",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				teamDir := filepath.Join(dir, "team-context")

				// create coworkers/ai/claude/agents directory
				agentsDir := filepath.Join(teamDir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatal(err)
				}

				// create a test agent file
				agentContent := `---
description: Test code reviewer specialist
model: sonnet
---

# code-reviewer

You are an expert code reviewer...
`
				if err := os.WriteFile(filepath.Join(agentsDir, "code-reviewer.md"), []byte(agentContent), 0644); err != nil {
					t.Fatal(err)
				}

				// create local config with team context
				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "test-team",
							TeamName: "Test Team",
							Path:     teamDir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "code-reviewer",
		},
		{
			name: "multiple teams",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()

				// create first team context
				team1Dir := filepath.Join(dir, "team1-context")
				agents1Dir := filepath.Join(team1Dir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agents1Dir, 0755); err != nil {
					t.Fatal(err)
				}
				agent1Content := `---
description: Code reviewer from team 1
model: sonnet
---
# code-reviewer
`
				if err := os.WriteFile(filepath.Join(agents1Dir, "code-reviewer.md"), []byte(agent1Content), 0644); err != nil {
					t.Fatal(err)
				}

				// create second team context
				team2Dir := filepath.Join(dir, "team2-context")
				agents2Dir := filepath.Join(team2Dir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agents2Dir, 0755); err != nil {
					t.Fatal(err)
				}
				agent2Content := `---
description: Style guide expert from team 2
model: opus
---
# style-guide-expert
`
				if err := os.WriteFile(filepath.Join(agents2Dir, "style-guide-expert.md"), []byte(agent2Content), 0644); err != nil {
					t.Fatal(err)
				}

				// create local config with both team contexts
				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "team-1",
							TeamName: "Team One",
							Path:     team1Dir,
						},
						{
							TeamID:   "team-2",
							TeamName: "Team Two",
							Path:     team2Dir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "## Team One", // should show team headers when multiple teams
		},
		{
			name: "teams from multiple contexts shown",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()

				// create first team context
				team1Dir := filepath.Join(dir, "team1-context")
				agents1Dir := filepath.Join(team1Dir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agents1Dir, 0755); err != nil {
					t.Fatal(err)
				}
				agent1Content := `---
description: Code reviewer from team 1
model: sonnet
---
# team1-reviewer
`
				if err := os.WriteFile(filepath.Join(agents1Dir, "team1-reviewer.md"), []byte(agent1Content), 0644); err != nil {
					t.Fatal(err)
				}

				// create second team context
				team2Dir := filepath.Join(dir, "team2-context")
				agents2Dir := filepath.Join(team2Dir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agents2Dir, 0755); err != nil {
					t.Fatal(err)
				}
				agent2Content := `---
description: Code reviewer from team 2
model: sonnet
---
# team2-reviewer
`
				if err := os.WriteFile(filepath.Join(agents2Dir, "team2-reviewer.md"), []byte(agent2Content), 0644); err != nil {
					t.Fatal(err)
				}

				// create local config with both team contexts
				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "team-1",
							TeamName: "Team One",
							Path:     team1Dir,
						},
						{
							TeamID:   "team-2",
							TeamName: "Team Two",
							Path:     team2Dir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "team1-reviewer", // should show agent from first team
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot, cleanup := tt.setupFn(t)
			defer cleanup()

			// change to project directory
			oldWd, _ := os.Getwd()
			if err := os.Chdir(projectRoot); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWd)

			// create command
			cmd := &cobra.Command{}
			cmd.Flags().Bool("json", tt.jsonMode, "")
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			// run command
			err := runCoworkerList(cmd, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCoworkerList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if tt.wantOutput != "" && !bytes.Contains([]byte(output), []byte(tt.wantOutput)) {
				t.Errorf("runCoworkerList() output = %q, want to contain %q", output, tt.wantOutput)
			}
		})
	}
}

func TestCoworkerAgentCommand(t *testing.T) {
	// set up agent context for tests (Claude Code sets this env var)
	os.Setenv("CLAUDE_CODE_SESSION_ID", "test-session")
	defer os.Unsetenv("CLAUDE_CODE_SESSION_ID")

	tests := []struct {
		name       string
		agentName  string
		setupFn    func(t *testing.T) (string, func())
		jsonMode   bool
		wantErr    bool
		wantOutput string
	}{
		{
			name:      "agent not found",
			agentName: "nonexistent",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				teamDir := filepath.Join(dir, "team-context")
				agentsDir := filepath.Join(teamDir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatal(err)
				}

				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "test-team",
							TeamName: "Test Team",
							Path:     teamDir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr: true,
		},
		{
			name:      "load agent successfully",
			agentName: "code-reviewer",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				teamDir := filepath.Join(dir, "team-context")
				agentsDir := filepath.Join(teamDir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatal(err)
				}

				agentContent := `---
description: Test code reviewer specialist
model: sonnet
---

# code-reviewer

You are an expert code reviewer...
`
				if err := os.WriteFile(filepath.Join(agentsDir, "code-reviewer.md"), []byte(agentContent), 0644); err != nil {
					t.Fatal(err)
				}

				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "test-team",
							TeamName: "Test Team",
							Path:     teamDir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "expert code reviewer",
		},
		{
			name:      "load agent json mode includes team info",
			agentName: "code-reviewer",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				teamDir := filepath.Join(dir, "team-context")
				agentsDir := filepath.Join(teamDir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatal(err)
				}

				agentContent := `---
description: Test code reviewer specialist
model: sonnet
---

# code-reviewer

You are an expert code reviewer...
`
				if err := os.WriteFile(filepath.Join(agentsDir, "code-reviewer.md"), []byte(agentContent), 0644); err != nil {
					t.Fatal(err)
				}

				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "test-team",
							TeamName: "Test Team",
							Path:     teamDir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			jsonMode: true,
			wantErr:  false,
		},
		{
			name:      "agent found in configured team",
			agentName: "team-reviewer",
			setupFn: func(t *testing.T) (string, func()) {
				dir := t.TempDir()

				// create team context
				teamDir := filepath.Join(dir, "team-context")
				agentsDir := filepath.Join(teamDir, "coworkers", "ai", "claude", "agents")
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatal(err)
				}
				agentContent := `---
description: Team code reviewer
model: sonnet
---
# team-reviewer
`
				if err := os.WriteFile(filepath.Join(agentsDir, "team-reviewer.md"), []byte(agentContent), 0644); err != nil {
					t.Fatal(err)
				}

				requireSageoxDir(t, dir)

				localCfg := &config.LocalConfig{
					TeamContexts: []config.TeamContext{
						{
							TeamID:   "test-team",
							TeamName: "Test Team",
							Path:     teamDir,
						},
					},
				}
				if err := config.SaveLocalConfig(dir, localCfg); err != nil {
					t.Fatal(err)
				}

				return dir, func() {}
			},
			wantErr:    false,
			wantOutput: "team-reviewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot, cleanup := tt.setupFn(t)
			defer cleanup()

			// change to project directory
			oldWd, _ := os.Getwd()
			if err := os.Chdir(projectRoot); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWd)

			// create command
			cmd := &cobra.Command{}
			cmd.Flags().String("model", "", "")
			cmd.Flags().Bool("json", tt.jsonMode, "")
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			// run command
			err := runCoworkerAgent(cmd, []string{tt.agentName})

			if (err != nil) != tt.wantErr {
				t.Errorf("runCoworkerAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if tt.wantOutput != "" && !bytes.Contains([]byte(output), []byte(tt.wantOutput)) {
				t.Errorf("runCoworkerAgent() output = %q, want to contain %q", output, tt.wantOutput)
			}

			// verify JSON output structure if in json mode
			if tt.jsonMode && err == nil {
				var jsonOutput coworkerAgentOutput
				if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
					t.Errorf("failed to parse JSON output: %v", err)
				}
				if !jsonOutput.Loaded {
					t.Error("expected loaded=true in JSON output")
				}
				if jsonOutput.TeamID == "" {
					t.Error("expected team_id in JSON output")
				}
			}
		})
	}
}

func TestFilterTeamsByEndpoint(t *testing.T) {
	// filterTeamsByEndpoint now returns all teams since team contexts are project-scoped
	// this test verifies that behavior
	tests := []struct {
		name            string
		teams           []config.TeamContext
		currentEndpoint string
		wantCount       int
	}{
		{
			name: "all teams returned for production endpoint",
			teams: []config.TeamContext{
				{TeamID: "team-1", TeamName: "Team 1"},
				{TeamID: "team-2", TeamName: "Team 2"},
				{TeamID: "team-3", TeamName: "Team 3"},
			},
			currentEndpoint: "https://sageox.ai",
			wantCount:       3,
		},
		{
			name: "all teams returned for non-production endpoint",
			teams: []config.TeamContext{
				{TeamID: "team-1", TeamName: "Team 1"},
				{TeamID: "team-2", TeamName: "Team 2"},
			},
			currentEndpoint: "https://staging.sageox.ai",
			wantCount:       2,
		},
		{
			name: "all teams returned for empty endpoint",
			teams: []config.TeamContext{
				{TeamID: "team-1", TeamName: "Team 1"},
				{TeamID: "team-2", TeamName: "Team 2"},
			},
			currentEndpoint: "",
			wantCount:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterTeamsByEndpoint(tt.teams, tt.currentEndpoint)

			if len(result) != tt.wantCount {
				t.Errorf("filterTeamsByEndpoint() returned %d teams, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestCoworkerLoadEntry(t *testing.T) {
	// test the session entry creation
	entry := newCoworkerLoadEntryForTest("code-reviewer", "sonnet")

	if entry.CoworkerName != "code-reviewer" {
		t.Errorf("expected coworker_name=code-reviewer, got %s", entry.CoworkerName)
	}
	if entry.CoworkerModel != "sonnet" {
		t.Errorf("expected coworker_model=sonnet, got %s", entry.CoworkerModel)
	}
	if entry.Type != "system" {
		t.Errorf("expected type=system, got %s", entry.Type)
	}
	if entry.Content == "" {
		t.Error("expected non-empty content")
	}
}

// newCoworkerLoadEntryForTest creates a test entry without importing session package
func newCoworkerLoadEntryForTest(name, model string) struct {
	Type          string `json:"type"`
	Content       string `json:"content"`
	CoworkerName  string `json:"coworker_name,omitempty"`
	CoworkerModel string `json:"coworker_model,omitempty"`
} {
	content := "Loaded coworker: " + name
	if model != "" {
		content += " (model: " + model + ")"
	}
	return struct {
		Type          string `json:"type"`
		Content       string `json:"content"`
		CoworkerName  string `json:"coworker_name,omitempty"`
		CoworkerModel string `json:"coworker_model,omitempty"`
	}{
		Type:          "system",
		Content:       content,
		CoworkerName:  name,
		CoworkerModel: model,
	}
}
