package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sageox/ox/internal/claude"
	"github.com/sageox/ox/internal/config"
	"github.com/sageox/ox/internal/endpoint"
	"github.com/sageox/ox/internal/session"
	"github.com/sageox/ox/internal/telemetry"
	"github.com/sageox/ox/pkg/agentx"
	"github.com/spf13/cobra"
)

// coworkerListOutput is the JSON output structure for ox coworker list
type coworkerListOutput struct {
	Coworkers []coworkerInfo `json:"coworkers"`
	Source    string         `json:"source"`
	Endpoint  string         `json:"endpoint"`
	Teams     []string       `json:"teams,omitempty"` // team IDs included in this list
}

// coworkerInfo represents a single coworker in the list output
type coworkerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"`
	Path        string `json:"path"`
	TeamID      string `json:"team_id"`
	TeamName    string `json:"team_name,omitempty"`
}

// coworkerAgentOutput is the JSON output structure for ox coworker agent <name>
type coworkerAgentOutput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
	Content     string `json:"content"`
	Path        string `json:"path"`
	TeamID      string `json:"team_id"`
	TeamName    string `json:"team_name,omitempty"`
	Loaded      bool   `json:"loaded"`
}

var coworkerCmd = &cobra.Command{
	Use:   "coworker",
	Short: "Manage team coworkers (AI subagents)",
	Long: `Manage team coworkers - expert AI subagents defined in your team context.

Coworkers are specialized agents with domain expertise that can be loaded
into your context when needed. They are defined in your team's coworkers/
directory and can be listed and loaded on demand.

Commands:
  list       List available coworkers
  agent      Load a coworker's prompt into context`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var coworkerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available team coworkers",
	Long: `List all available coworkers from team contexts at the current endpoint.

Coworkers are expert AI subagents defined in coworkers/ai/claude/agents/.
Use 'ox coworker agent <name>' to load a coworker's expertise into your context.

If you have multiple teams at the current endpoint, coworkers from all teams
are listed with their team attribution.`,
	RunE: runCoworkerList,
}

var coworkerAgentCmd = &cobra.Command{
	Use:   "agent <name>",
	Short: "Load a coworker's prompt into context",
	Long: `Load a coworker's full prompt content into your agent context.

This outputs the coworker's expertise as markdown, which Claude can then
use for specialized tasks. The load event is also logged to the session
for metrics on coworker usage.

Searches all team contexts at the current endpoint for the named coworker.

Example:
  ox coworker agent code-reviewer
  ox coworker agent code-reviewer --model opus`,
	Args: cobra.ExactArgs(1),
	RunE: runCoworkerAgent,
}

func init() {
	// coworker list flags
	coworkerListCmd.Flags().Bool("json", false, "Output as JSON")

	// coworker agent flags
	coworkerAgentCmd.Flags().String("model", "", "Override the coworker's default model (sonnet, opus, haiku)")
	coworkerAgentCmd.Flags().Bool("json", false, "Output as JSON")

	coworkerCmd.AddCommand(coworkerListCmd)
	coworkerCmd.AddCommand(coworkerAgentCmd)
	rootCmd.AddCommand(coworkerCmd)
}

func runCoworkerList(cmd *cobra.Command, args []string) error {
	// no agent gate - coworker list is useful for both humans and agents
	jsonMode, _ := cmd.Flags().GetBool("json")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// get current endpoint
	currentEndpoint := endpoint.GetForProject(projectRoot)

	// load team contexts
	localCfg, err := config.LoadLocalConfig(projectRoot)
	if err != nil || len(localCfg.TeamContexts) == 0 {
		if jsonMode {
			output := coworkerListOutput{
				Coworkers: []coworkerInfo{},
				Source:    "none",
				Endpoint:  currentEndpoint,
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No team context configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'ox init' to set up team context.")
		return nil
	}

	// filter team contexts by current endpoint
	matchingTeams := filterTeamsByEndpoint(localCfg.TeamContexts, currentEndpoint)

	if len(matchingTeams) == 0 {
		if jsonMode {
			output := coworkerListOutput{
				Coworkers: []coworkerInfo{},
				Source:    "no_matching_teams",
				Endpoint:  currentEndpoint,
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "No team contexts found for endpoint: %s\n", currentEndpoint)
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'ox init' to set up team context.")
		return nil
	}

	// aggregate coworkers from all matching teams
	var allCoworkers []coworkerInfo
	var teamIDs []string

	for _, tc := range matchingTeams {
		// verify team context path exists
		if _, err := os.Stat(tc.Path); os.IsNotExist(err) {
			continue
		}

		// discover agents for this team
		agents, err := claude.DiscoverAgents(tc.Path)
		if err != nil {
			continue
		}

		teamIDs = append(teamIDs, tc.TeamID)

		for _, agent := range agents {
			allCoworkers = append(allCoworkers, coworkerInfo{
				Name:        agent.Name,
				Description: agent.Description,
				Model:       agent.Model,
				Path:        agent.Path,
				TeamID:      tc.TeamID,
				TeamName:    tc.TeamName,
			})
		}
	}

	if jsonMode {
		output := coworkerListOutput{
			Coworkers: allCoworkers,
			Source:    "team_context",
			Endpoint:  currentEndpoint,
			Teams:     teamIDs,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// text output
	if len(allCoworkers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Claude subagents found in team contexts.")
		fmt.Fprintln(cmd.OutOrStdout(), "Add agent files to coworkers/ai/claude/agents/ in your team context.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Claude Subagents")
	fmt.Fprintln(cmd.OutOrStdout())

	// group coworkers by team to avoid repeating team name on every row
	coworkersByTeam := make(map[string][]coworkerInfo)
	teamOrder := make([]string, 0) // preserve order
	for _, cw := range allCoworkers {
		key := cw.TeamID
		if _, exists := coworkersByTeam[key]; !exists {
			teamOrder = append(teamOrder, key)
		}
		coworkersByTeam[key] = append(coworkersByTeam[key], cw)
	}

	// output grouped by team
	for _, teamID := range teamOrder {
		coworkers := coworkersByTeam[teamID]
		if len(coworkers) == 0 {
			continue
		}

		// team header (only if multiple teams)
		if len(teamOrder) > 1 {
			teamDisplay := teamID
			if coworkers[0].TeamName != "" {
				teamDisplay = coworkers[0].TeamName
			}
			fmt.Fprintf(cmd.OutOrStdout(), "## %s\n", teamDisplay)
			fmt.Fprintln(cmd.OutOrStdout())
		}

		fmt.Fprintln(cmd.OutOrStdout(), "| Name | When to Use |")
		fmt.Fprintln(cmd.OutOrStdout(), "|------|-------------|")
		for _, cw := range coworkers {
			desc := cw.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "| %s | %s |\n", cw.Name, desc)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Load: ox coworker agent <name>")

	return nil
}

func runCoworkerAgent(cmd *cobra.Command, args []string) error {
	// gate: require agent context
	if errMsg := agentx.RequireAgent("ox coworker agent"); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}

	name := args[0]
	modelOverride, _ := cmd.Flags().GetString("model")
	jsonMode, _ := cmd.Flags().GetBool("json")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("could not find project root: %w", err)
	}

	// get current endpoint
	currentEndpoint := endpoint.GetForProject(projectRoot)

	// load team contexts
	localCfg, err := config.LoadLocalConfig(projectRoot)
	if err != nil || len(localCfg.TeamContexts) == 0 {
		return fmt.Errorf("no team context configured; run 'ox init' to set up")
	}

	// filter team contexts by current endpoint
	matchingTeams := filterTeamsByEndpoint(localCfg.TeamContexts, currentEndpoint)

	if len(matchingTeams) == 0 {
		return fmt.Errorf("no team contexts found for endpoint %s; run 'ox init' to set up", currentEndpoint)
	}

	// search for agent in matching team contexts only
	var agentContent *claude.AgentContent
	var foundTeam *config.TeamContext
	for i := range matchingTeams {
		tc := &matchingTeams[i]
		content, err := claude.LoadAgent(tc.Path, name)
		if err == nil && content != nil {
			agentContent = content
			foundTeam = tc
			break
		}
	}

	if agentContent == nil {
		return fmt.Errorf("coworker %q not found in team contexts for endpoint %s", name, currentEndpoint)
	}

	// apply model override if provided
	model := agentContent.Model
	if modelOverride != "" {
		model = modelOverride
	}

	// log to session if recording
	logCoworkerLoad(projectRoot, name, model)

	// track coworker load via telemetry
	trackCoworkerLoad(name, model, foundTeam.TeamID)

	if jsonMode {
		output := coworkerAgentOutput{
			Name:        agentContent.Name,
			Description: agentContent.Description,
			Model:       model,
			Content:     agentContent.Content,
			Path:        agentContent.Path,
			TeamID:      foundTeam.TeamID,
			TeamName:    foundTeam.TeamName,
			Loaded:      true,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// text output - emit full content for agent consumption
	fmt.Fprintln(cmd.OutOrStdout(), agentContent.Content)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "---")
	modelInfo := model
	if modelInfo == "" {
		modelInfo = "inherit"
	}
	teamInfo := foundTeam.TeamID
	if foundTeam.TeamName != "" {
		teamInfo = foundTeam.TeamName
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Coworker loaded: %s (model: %s, team: %s)\n", name, modelInfo, teamInfo)

	return nil
}

// filterTeamsByEndpoint returns team contexts that match the given endpoint.
// Team contexts are now always derived from the project's endpoint configuration,
// so all team contexts for this project use the same endpoint.
// This function validates that the current endpoint is a valid production or
// non-production endpoint for filtering purposes.
func filterTeamsByEndpoint(teams []config.TeamContext, currentEndpoint string) []config.TeamContext {
	// all team contexts in a project now use the project's endpoint
	// the filtering is implicit: if the project has team contexts configured,
	// they are already scoped to this project's endpoint
	//
	// we still validate that the current endpoint is sensible (production or explicit)
	// but all configured teams match since they're derived from project config
	if currentEndpoint == "" || endpoint.IsProduction(currentEndpoint) {
		// production endpoint or default - all teams match
		return teams
	}

	// non-production endpoint - all teams still match since they're project-scoped
	return teams
}

// logCoworkerLoad writes a coworker load entry to the session if recording.
func logCoworkerLoad(projectRoot, name, model string) {
	if !session.IsRecording(projectRoot) {
		return
	}

	state, err := session.LoadRecordingState(projectRoot)
	if err != nil || state == nil {
		return
	}

	// create coworker load entry
	entry := session.NewCoworkerLoadEntry(name, model)

	// append to raw.jsonl
	if state.SessionFile != "" {
		appendEntryToSession(state.SessionFile, entry)
	} else if state.SessionPath != "" {
		rawFile := state.SessionPath + "/raw.jsonl"
		appendEntryToSession(rawFile, entry)
	}
}

// appendEntryToSession appends a session entry to a session file.
func appendEntryToSession(filePath string, entry session.SessionEntry) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = f.Write(data)
	_, _ = f.WriteString("\n")
}

// trackCoworkerLoad sends a telemetry event for coworker loads.
func trackCoworkerLoad(name, model, teamID string) {
	if cliCtx == nil || cliCtx.TelemetryClient == nil {
		return
	}

	metadata := make(map[string]string)
	if teamID != "" {
		metadata["team_id"] = teamID
	}

	cliCtx.TelemetryClient.Track(telemetry.Event{
		Type:          telemetry.EventCoworkerLoad,
		CoworkerName:  name,
		CoworkerModel: model,
		Metadata:      metadata,
		Success:       true,
	})
}
