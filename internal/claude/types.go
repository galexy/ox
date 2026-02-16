// Package claude provides utilities for discovering and parsing team Claude customizations.
// It handles discovery of agents and commands from team context directories.
package claude

// Agent represents a Claude Code subagent definition from a team context.
type Agent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"` // "inherit", "opus", "sonnet", "haiku"
	Path        string `json:"path"`            // absolute file path
	FromIndex   bool   `json:"from_index"`      // true if description came from index.md
}

// Command represents a slash command defined in a team context.
type Command struct {
	Name        string `json:"name"`        // command name (without /)
	Trigger     string `json:"trigger"`     // how to invoke (e.g., "/deploy")
	Description string `json:"description"` // what it does
	Path        string `json:"path"`        // absolute file path
	FromIndex   bool   `json:"from_index"`  // true if description came from index.md
}

// TeamCustomizations holds all Claude customizations discovered from a team context.
type TeamCustomizations struct {
	TeamPath string `json:"team_path"` // base path of team context

	// Instruction files (CLAUDE.md, AGENTS.md)
	ClaudeMDPath string `json:"claude_md_path,omitempty"` // coworkers/ai/claude/CLAUDE.md
	AgentsMDPath string `json:"agents_md_path,omitempty"` // coworkers/ai/claude/AGENTS.md
	HasClaudeMD  bool   `json:"has_claude_md"`
	HasAgentsMD  bool   `json:"has_agents_md"`

	// Agents index (coworkers/ai/claude/agents/index.md)
	HasAgentsIndex  bool   `json:"has_agents_index,omitempty"`
	AgentsIndexPath string `json:"agents_index_path,omitempty"` // path to index.md if exists

	// Discovered items
	Agents   []Agent   `json:"agents,omitempty"`
	Commands []Command `json:"commands,omitempty"`
}

// HasInstructionFiles returns true if either CLAUDE.md or AGENTS.md exists.
func (tc *TeamCustomizations) HasInstructionFiles() bool {
	return tc.HasClaudeMD || tc.HasAgentsMD
}

// HasAnyCustomizations returns true if any customizations were discovered.
func (tc *TeamCustomizations) HasAnyCustomizations() bool {
	return tc.HasClaudeMD || tc.HasAgentsMD || len(tc.Agents) > 0 || len(tc.Commands) > 0
}
