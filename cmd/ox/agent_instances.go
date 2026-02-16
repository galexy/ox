package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sageox/ox/internal/cli"
	"github.com/sageox/ox/internal/daemon"
	"github.com/spf13/cobra"
)

var agentInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List active agent instances across all workspaces",
	Long: `List all active agent instances tracked by ox daemons.

Shows agents that have recently sent heartbeats to any running daemon.
Instances are considered active if they've sent a heartbeat within 30 seconds.

Example:
  $ ox agent instances
  AGENT     WORKSPACE              STATUS   LAST HEARTBEAT
  Oxa7b3    ~/code/project-a       active   2s ago
  Oxc2d4    ~/code/project-b       idle     45s ago

  $ ox agent instances --json
  {"instances":[{"agent_id":"Oxa7b3","workspace_path":"/Users/dev/code/project-a",...}]}`,
	RunE: runAgentInstances,
}

// agentSessionsCmd is deprecated, use agentInstancesCmd instead
var agentSessionsCmd = &cobra.Command{
	Use:    "sessions",
	Short:  "List active agent instances (deprecated: use 'instances')",
	Hidden: true, // hide from help, redirect to instances
	RunE:   runAgentInstances,
}

func init() {
	agentInstancesCmd.Flags().Bool("json", false, "Output as JSON")
	agentSessionsCmd.Flags().Bool("json", false, "Output as JSON")
	agentCmd.AddCommand(agentInstancesCmd)
	agentCmd.AddCommand(agentSessionsCmd) // keep for backward compat
}

func runAgentInstances(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	instances, err := daemon.GetAllInstances()
	if err != nil {
		if jsonOutput {
			return outputInstancesJSON(nil, err)
		}
		return fmt.Errorf("failed to get instances: %w", err)
	}

	if jsonOutput {
		return outputInstancesJSON(instances, nil)
	}

	return outputInstancesTable(instances)
}

func outputInstancesJSON(instances []daemon.InstanceInfo, err error) error {
	type output struct {
		Instances []daemon.InstanceInfo `json:"instances"`
		Error     string                `json:"error,omitempty"`
	}

	out := output{Instances: instances}
	if err != nil {
		out.Error = err.Error()
	}
	if out.Instances == nil {
		out.Instances = []daemon.InstanceInfo{}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputInstancesTable(instances []daemon.InstanceInfo) error {
	if len(instances) == 0 {
		fmt.Println("No active agent instances.")
		fmt.Println()
		fmt.Println("Agent instances are tracked when agents run 'ox agent prime' and send heartbeats.")
		fmt.Println("Run 'ox agent prime' in a repository to start a new instance.")
		return nil
	}

	// header
	fmt.Printf("%-10s %-40s %-8s %s\n",
		cli.StyleBold.Render("AGENT"),
		cli.StyleBold.Render("WORKSPACE"),
		cli.StyleBold.Render("STATUS"),
		cli.StyleBold.Render("LAST HEARTBEAT"))
	fmt.Println(strings.Repeat("-", 80))

	// rows
	for _, inst := range instances {
		workspace := shortenPath(inst.WorkspacePath)
		lastHB := formatTimeAgoShort(inst.LastHeartbeat)

		statusStyle := cli.StyleSuccess
		if inst.Status == daemon.StatusIdle {
			statusStyle = cli.StyleDim
		}

		fmt.Printf("%-10s %-40s %s %s\n",
			inst.AgentID,
			workspace,
			statusStyle.Render(fmt.Sprintf("%-8s", inst.Status)),
			cli.StyleDim.Render(lastHB))
	}

	fmt.Println()
	fmt.Printf("%d active instance(s)\n", len(instances))

	return nil
}

// shortenPath shortens a path for display by replacing home dir with ~
// and truncating long paths
func shortenPath(path string) string {
	if path == "" {
		return "-"
	}

	// replace home dir with ~
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}

	// truncate if too long (keep last 38 chars)
	maxLen := 38
	if len(path) > maxLen {
		// try to keep meaningful part (project name)
		parts := strings.Split(path, string(filepath.Separator))
		if len(parts) > 2 {
			// keep last two parts
			path = ".../" + filepath.Join(parts[len(parts)-2], parts[len(parts)-1])
		}
		if len(path) > maxLen {
			path = "..." + path[len(path)-maxLen+3:]
		}
	}

	return path
}

// formatTimeAgoShort formats a time as a short relative time
func formatTimeAgoShort(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	diff := time.Since(t)

	switch {
	case diff < time.Second:
		return "now"
	case diff < time.Minute:
		return fmt.Sprintf("%ds ago", int(diff.Seconds()))
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}
