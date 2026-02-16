package main

import (
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage AI coworker sessions",
	Long: `View and manage sessions of human + AI coworker conversations.

Sessions are recordings of interactive discussions between developers
and AI coworkers (Claude Code, Cursor, etc.). Capture struggles,
solutions, and learnings to build collective team knowledge.

Viewing:
  ox session list      List all sessions
  ox session view      View a session (html, text, or json)

Export:
  ox session export    Export session to HTML or markdown file

Management:
  ox session commit    Commit pending sessions
  ox session remove    Remove a session
  ox session download  Download session content from the ledger
  ox session upload    Upload session content to the ledger

Recording (for AI coworkers):
  ox agent <id> session start   Start recording a session
  ox agent <id> session stop    Stop recording and save

Policy:
  ox session redaction policy   View secret redaction patterns
  ox session redaction verify   Verify redaction policy signature`,
}

func init() {
	sessionCmd.GroupID = "dev"

	// register subcommands
	sessionCmd.AddCommand(sessionCommitCmd)
	sessionCmd.AddCommand(sessionHydrateCmd)
	sessionCmd.AddCommand(sessionUploadCmd)
	sessionCmd.AddCommand(sessionPushSummaryCmd)

	// TODO(post-MVP): commit, download, and upload should be automated.
	// Users should only need start/stop — the rest is implementation detail.
	// Currently surfaced because automation isn't reliable enough yet.

	rootCmd.AddCommand(sessionCmd)
}
