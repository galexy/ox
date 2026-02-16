package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sageox/ox/internal/cli"
	"github.com/sageox/ox/internal/session"
	"github.com/spf13/cobra"
)

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check recording status",
	Long: `Check the current session recording status.

Shows whether a recording is in progress, and if so, displays:
- Recording title (if set)
- Duration
- Coding agent being recorded
- Session file location

Examples:
  ox session status
  ox session status --json`,
	RunE: runSessionStatus,
}

// sessionStatusOutput is the JSON output format for session status.
type sessionStatusOutput struct {
	Recording    bool   `json:"recording"`
	Title        string `json:"title,omitempty"`
	DurationSecs int    `json:"duration_seconds,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Agent        string `json:"agent,omitempty"`
	AgentID      string `json:"agent_id,omitempty"`
	SessionFile  string `json:"session_file,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
}

func init() {
	sessionCmd.AddCommand(sessionStatusCmd)
}

func runSessionStatus(cmd *cobra.Command, args []string) error {
	// check for --json flag from root command
	jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")

	// find project root using helper
	projectRoot, err := requireProjectRoot()
	if err != nil {
		if jsonOutput {
			return outputJSON(sessionStatusOutput{Recording: false})
		}
		return err
	}

	// load recording state
	state, err := session.LoadRecordingState(projectRoot)
	if err != nil {
		if jsonOutput {
			return outputJSON(sessionStatusOutput{Recording: false})
		}
		return fmt.Errorf("failed to check recording state: %w", err)
	}

	// not recording
	if state == nil {
		if jsonOutput {
			return outputJSON(sessionStatusOutput{Recording: false})
		}

		fmt.Println(cli.StyleDim.Render("Not recording"))
		fmt.Println()
		fmt.Println("Run 'ox agent <id> session start' to begin recording")
		return nil
	}

	// recording in progress
	duration := state.Duration()
	durationStr := formatDurationHuman(duration)

	if jsonOutput {
		output := sessionStatusOutput{
			Recording:    true,
			Title:        state.Title,
			DurationSecs: int(duration.Seconds()),
			Duration:     durationStr,
			Agent:        state.AdapterName,
			AgentID:      state.AgentID,
			SessionFile:  state.SessionFile,
			StartedAt:    state.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		return outputJSON(output)
	}

	// text output
	fmt.Println(cli.StyleSuccess.Render("Recording in progress"))
	fmt.Println()

	if state.Title != "" {
		fmt.Printf("  Title:    %s\n", state.Title)
	}
	fmt.Printf("  Duration: %s\n", durationStr)
	fmt.Printf("  Agent:    %s\n", state.AdapterName)
	fmt.Printf("  Started:  %s\n", state.StartedAt.Format("15:04:05"))

	if state.AgentID != "" {
		fmt.Printf("  Agent ID: %s\n", state.AgentID)
	}

	fmt.Println()
	fmt.Println(cli.StyleDim.Render("Run 'ox agent <id> session stop' to save the recording"))

	return nil
}

// outputJSON writes JSON to stdout.
func outputJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
