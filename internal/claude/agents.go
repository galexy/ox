package claude

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ClaudeDir is the standard path for Claude customizations within a team context.
const ClaudeDir = "coworkers/ai/claude"

// DiscoverAll finds all Claude customizations in a team context path.
// This is the main entry point for discovering team Claude customizations.
func DiscoverAll(teamPath string) (*TeamCustomizations, error) {
	if teamPath == "" {
		return nil, nil
	}

	tc := &TeamCustomizations{
		TeamPath: teamPath,
	}

	claudeBase := filepath.Join(teamPath, ClaudeDir)

	// check for instruction files
	claudeMD := filepath.Join(claudeBase, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); err == nil {
		tc.ClaudeMDPath = claudeMD
		tc.HasClaudeMD = true
	}

	agentsMD := filepath.Join(claudeBase, "AGENTS.md")
	if _, err := os.Stat(agentsMD); err == nil {
		tc.AgentsMDPath = agentsMD
		tc.HasAgentsMD = true
	}

	// discover agents
	agents, _ := DiscoverAgents(teamPath)
	tc.Agents = agents

	// check for agents index.md (provides catalog of available specialists)
	agentsIndexPath := filepath.Join(claudeBase, "agents", "index.md")
	if _, err := os.Stat(agentsIndexPath); err == nil {
		tc.HasAgentsIndex = true
		tc.AgentsIndexPath = agentsIndexPath
	}

	// discover commands (uses DiscoverTeamCommands from commands.go)
	commands, _ := DiscoverTeamCommands(teamPath)
	tc.Commands = commands

	return tc, nil
}

// DiscoverAgents finds all agents in a team context.
// Checks index.md first (token-optimized), falls back to individual files.
func DiscoverAgents(teamPath string) ([]Agent, error) {
	agentsDir := filepath.Join(teamPath, ClaudeDir, "agents")

	// check if agents directory exists
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return nil, nil
	}

	// try to load index.md for optimized descriptions
	indexDescriptions := ParseIndex(filepath.Join(agentsDir, "index.md"))

	// scan for .md files
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, err
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if entry.Name() == "index.md" {
			continue // skip index itself
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		agentPath := filepath.Join(agentsDir, entry.Name())

		agent := Agent{
			Name: name,
			Path: agentPath,
		}

		// prefer index.md description (token-optimized)
		if desc, ok := indexDescriptions[name]; ok {
			agent.Description = desc
			agent.FromIndex = true
		} else {
			// fallback to parsing file frontmatter
			agent.Description, agent.Model = parseAgentFrontmatter(agentPath)
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// parseAgentFrontmatter extracts description and model from agent file frontmatter.
// Returns empty strings if frontmatter is not found or invalid.
func parseAgentFrontmatter(path string) (description, model string) {
	file, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// frontmatter starts with ---
		if lineCount == 1 && line == "---" {
			inFrontmatter = true
			continue
		}

		// frontmatter ends with ---
		if inFrontmatter && line == "---" {
			break
		}

		if inFrontmatter {
			if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				description = strings.Trim(description, `"'`)
			}
			if strings.HasPrefix(line, "model:") {
				model = strings.TrimSpace(strings.TrimPrefix(line, "model:"))
				model = strings.Trim(model, `"'`)
			}
		}

		// stop after 20 lines to avoid scanning entire file
		if lineCount > 20 {
			break
		}
	}

	return description, model
}

// AgentContent holds the full content and metadata of an agent file.
type AgentContent struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
	Path        string `json:"path"`
	Content     string `json:"content"` // full markdown content
}

// LoadAgent loads the full content of an agent file by name.
// Searches in the standard coworkers/ai/claude/agents/ directory.
// Returns the full file content along with parsed frontmatter metadata.
func LoadAgent(teamPath, name string) (*AgentContent, error) {
	if teamPath == "" || name == "" {
		return nil, nil
	}

	agentPath := filepath.Join(teamPath, ClaudeDir, "agents", name+".md")

	data, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, err
	}

	description, model := parseAgentFrontmatter(agentPath)

	return &AgentContent{
		Name:        name,
		Description: description,
		Model:       model,
		Path:        agentPath,
		Content:     string(data),
	}, nil
}

// FindAgent looks up an agent by name across all configured team contexts.
// Returns nil if the agent is not found.
func FindAgent(teamPaths []string, name string) (*AgentContent, error) {
	for _, teamPath := range teamPaths {
		agent, err := LoadAgent(teamPath, name)
		if err == nil && agent != nil {
			return agent, nil
		}
	}
	return nil, nil
}

// parseFirstDescription extracts the first meaningful description from a file.
// Looks for frontmatter description or first paragraph after title.
func parseFirstDescription(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	lineCount := 0
	passedTitle := false

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// handle frontmatter
		if lineCount == 1 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
				continue
			}
			if strings.HasPrefix(line, "description:") {
				desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				return strings.Trim(desc, `"'`)
			}
			continue
		}

		// skip empty lines and titles
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			passedTitle = true
			continue
		}

		// return first non-empty, non-title line after title
		if passedTitle && trimmed != "" {
			// truncate long descriptions
			if len(trimmed) > 120 {
				return trimmed[:117] + "..."
			}
			return trimmed
		}

		// stop after 30 lines
		if lineCount > 30 {
			break
		}
	}

	return ""
}
