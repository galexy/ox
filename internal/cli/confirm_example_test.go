package cli_test

import (
	"fmt"

	"github.com/sageox/ox/internal/cli"
)

// Example demonstrates how to use the uninstall confirmation flow
func ExampleConfirmUninstall() {
	// in actual use, these would come from scanning the repository
	items := []cli.RemovalItem{
		{
			Type:        "directory",
			Path:        ".sageox",
			Description: "SageOx state and configuration",
		},
		{
			Type:        "file",
			Path:        "AGENTS.md",
			Description: "agent guidance document",
		},
		{
			Type:        "file",
			Path:        ".claude/CLAUDE.md",
			Description: "Claude-specific guidance",
		},
		{
			Type:        "hook",
			Path:        ".git/hooks/pre-commit",
			Description: "git pre-commit hook",
		},
	}

	// show what will be removed
	cli.ShowUninstallPreview(items, false)

	// display warning
	cli.DangerousOperationWarning("UNINSTALL SAGEOX", "my-awesome-repo")

	// get confirmation (would use force flag from command line)
	force := false // in real code: cmd.Flags().GetBool("force")
	if err := cli.ConfirmUninstall("my-awesome-repo", force); err != nil {
		fmt.Println("Uninstall canceled")
		return
	}

	fmt.Println("Proceeding with uninstall...")
	// ... actual uninstall logic here ...
}

// Example demonstrates dry-run mode
func ExampleShowUninstallPreview_dryRun() {
	items := []cli.RemovalItem{
		{Type: "directory", Path: ".sageox", Description: "state directory"},
		{Type: "file", Path: "AGENTS.md", Description: "guidance"},
	}

	// dry-run mode shows what would be removed without doing it
	cli.ShowUninstallPreview(items, true)
}

// Example demonstrates generic dangerous operation confirmation
func ExampleConfirmDangerousOperation() {
	// for operations where you want user to type a specific string
	operation := "delete all production data"
	confirmText := "DELETE-PROD"

	if err := cli.ConfirmDangerousOperation(operation, confirmText, false); err != nil {
		fmt.Println("Operation canceled")
		return
	}

	fmt.Println("User confirmed, proceeding...")
}

// Example demonstrates yes/no prompts
func ExampleConfirmYesNo() {
	// simple yes/no with default to No (safer for destructive ops)
	if cli.ConfirmYesNo("Remove user-level config too?", false) {
		fmt.Println("Will remove user config")
	}

	// with default to Yes (for non-destructive confirmations)
	if cli.ConfirmYesNo("Install recommended plugins?", true) {
		fmt.Println("Will install plugins")
	}
}
