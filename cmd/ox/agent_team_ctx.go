package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sageox/ox/internal/config"
	"github.com/spf13/cobra"
)

var agentTeamCtxCmd = &cobra.Command{
	Use:   "team-ctx",
	Short: "Output team context for AI agent planning",
	Long: `Output distilled team discussions and decisions for AI agent planning.

This command reads the team's agent-context/distilled-discussions.md file
and outputs it with context for the AI agent to use during planning.

Output includes a content hash (team-ctx:<hash>) - if this marker is already
in your context, you don't need to re-run this command.`,
	RunE: runAgentTeamCtx,
}

func runAgentTeamCtx(cmd *cobra.Command, args []string) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("not in a SageOx project: %w", err)
	}

	localCfg, err := config.LoadLocalConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	if len(localCfg.TeamContexts) == 0 {
		return fmt.Errorf("no team context configured for this project")
	}

	tc := localCfg.TeamContexts[0]
	agentContextPath := filepath.Join(tc.Path, "agent-context", "distilled-discussions.md")
	content, err := os.ReadFile(agentContextPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no team context available: %s does not exist", agentContextPath)
		}
		return fmt.Errorf("failed to read agent context: %w", err)
	}

	// compute content hash for deduplication
	hash := sha256.Sum256(content)
	hashStr := fmt.Sprintf("%x", hash[:4]) // first 8 hex chars

	// output with hash marker for context deduplication
	fmt.Fprintf(cmd.OutOrStdout(), "<!-- team-ctx:%s -->\n", hashStr)
	fmt.Fprintln(cmd.OutOrStdout(), "Use this team context during planning:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprint(cmd.OutOrStdout(), string(content))

	return nil
}
