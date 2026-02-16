package main

import (
	"fmt"
	"strings"

	"github.com/sageox/ox/internal/cli"
	"github.com/sageox/ox/internal/session"
	"github.com/spf13/cobra"
)

var sessionRemoveCmd = &cobra.Command{
	Use:   "remove <filename>",
	Short: "Remove a session",
	Long: `Remove a session from the local cache.

The filename can be a full filename or a partial match.
Use 'ox session list' to see available sessions.

This removes the session from local storage. If the session
has already been committed to the ledger, you may need to
remove it there separately.

Examples:
  ox session remove 2026-01-05T10-30-ryan-Oxa7b3.jsonl
  ox session remove Oxa7b3    # partial match by agent ID
  ox session remove --all     # remove all sessions (with confirmation)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSessionRemove,
}

// Future: add session editing capability
// This could be implemented as:
// 1. Cloud feature via SageOx dashboard for collaborative editing
// 2. Local 'ox dashboard' server at localhost:1729 with browser UI
// 3. Export to markdown for manual editing, then re-import
// For now, users can manually edit JSONL files if needed.

func init() {
	sessionCmd.AddCommand(sessionRemoveCmd)
	sessionRemoveCmd.Flags().Bool("all", false, "Remove all sessions (requires confirmation)")
	sessionRemoveCmd.Flags().Bool("force", false, "Skip confirmation prompts")
}

func runSessionRemove(cmd *cobra.Command, args []string) error {
	removeAll, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

	store, _, err := newSessionStore()
	if err != nil {
		return err
	}

	if removeAll {
		return removeAllSessions(store, force)
	}

	if len(args) == 0 {
		return fmt.Errorf("please specify a session filename or use --all\nRun 'ox session list' to see available sessions")
	}

	return removeSessionByPattern(store, args[0], force)
}

// removeAllSessions removes all sessions with confirmation
func removeAllSessions(store *session.Store, force bool) error {
	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions to remove.")
		return nil
	}

	// confirm unless force flag is set
	if !force {
		if !cli.ConfirmYesNo(fmt.Sprintf("This will remove %d session(s). Continue?", len(sessions)), false) {
			fmt.Println("Canceled.")
			return nil
		}
	}

	var removed int
	for _, t := range sessions {
		if err := store.Delete(t.Filename); err != nil {
			fmt.Printf("  Warning: failed to remove %s: %v\n", t.Filename, err)
		} else {
			removed++
		}
	}

	cli.PrintSuccess(fmt.Sprintf("Removed %d session(s)", removed))
	return nil
}

// removeSessionByPattern removes sessions matching the pattern
func removeSessionByPattern(store *session.Store, pattern string, force bool) error {
	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// find matching sessions
	var matches []session.SessionInfo
	for _, t := range sessions {
		if strings.Contains(t.Filename, pattern) {
			matches = append(matches, t)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no sessions found matching %q\nRun 'ox session list' to see available sessions", pattern)
	}

	// if multiple matches, show them and ask user to be more specific
	if len(matches) > 1 && !force {
		fmt.Printf("Multiple sessions match %q:\n", pattern)
		for _, m := range matches {
			fmt.Printf("  %s (%s)\n", m.Filename, m.Type)
		}
		fmt.Println("\nPlease provide a more specific pattern or use --force to remove all matches.")
		return nil
	}

	// confirm unless force flag is set
	if !force && len(matches) == 1 {
		if !cli.ConfirmYesNo(fmt.Sprintf("Remove %s?", matches[0].Filename), false) {
			fmt.Println("Canceled.")
			return nil
		}
	}

	var removed int
	for _, m := range matches {
		if err := store.Delete(m.Filename); err != nil {
			fmt.Printf("  Warning: failed to remove %s: %v\n", m.Filename, err)
		} else {
			removed++
		}
	}

	if removed == 1 {
		cli.PrintSuccess(fmt.Sprintf("Removed %s", matches[0].Filename))
	} else {
		cli.PrintSuccess(fmt.Sprintf("Removed %d session(s)", removed))
	}

	return nil
}
