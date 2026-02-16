package adapters

import (
	"regexp"
	"strings"

	"github.com/sageox/ox/internal/uxfriction"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CobraAdapter implements uxfriction.CLIAdapter for spf13/cobra.
type CobraAdapter struct {
	root *cobra.Command
}

// NewCobraAdapter creates a new Cobra adapter for the given root command.
func NewCobraAdapter(root *cobra.Command) *CobraAdapter {
	return &CobraAdapter{root: root}
}

// CommandNames returns all available command names.
// Returns names as "subcmd" or "parent subcmd" for nested commands.
// Hidden and deprecated commands are excluded.
func (a *CobraAdapter) CommandNames() []string {
	var names []string
	walkCommands(a.root, "", func(cmd *cobra.Command, prefix string) {
		// skip root command itself
		if cmd == a.root {
			return
		}
		// skip hidden or deprecated commands
		if cmd.Hidden || cmd.Deprecated != "" {
			return
		}
		name := cmd.Name()
		if prefix != "" {
			name = prefix + " " + name
		}
		names = append(names, name)
	})
	return names
}

// FlagNames returns all available flag names for a command.
// If command is empty, returns global (persistent) flags from root.
// Returns flags as "--name" and "-shorthand" if shorthand is available.
func (a *CobraAdapter) FlagNames(command string) []string {
	var flags []string

	cmd := a.findCommand(command)
	if cmd == nil {
		return flags
	}

	// collect flags from this command
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, "--"+f.Name)
		if f.Shorthand != "" {
			flags = append(flags, "-"+f.Shorthand)
		}
	})

	return flags
}

// ParseError extracts structured info from a Cobra CLI error.
// Returns nil if the error is not a parseable Cobra error.
func (a *CobraAdapter) ParseError(err error) *uxfriction.ParsedError {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// "unknown command \"X\" for \"parent\""
	if strings.Contains(msg, "unknown command") {
		parsed := &uxfriction.ParsedError{
			Kind:       uxfriction.FailureUnknownCommand,
			RawMessage: msg,
		}
		if token := extractQuoted(msg); token != "" {
			parsed.BadToken = token
		}
		// extract parent command from "for \"parent\""
		if idx := strings.Index(msg, "\" for \""); idx != -1 {
			rest := msg[idx+7:]
			if end := strings.Index(rest, "\""); end != -1 {
				parsed.Command = rest[:end]
			}
		}
		return parsed
	}

	// "unknown flag: --X"
	if strings.Contains(msg, "unknown flag:") {
		parsed := &uxfriction.ParsedError{
			Kind:       uxfriction.FailureUnknownFlag,
			RawMessage: msg,
		}
		// extract the flag name after "unknown flag: "
		if idx := strings.Index(msg, "unknown flag: "); idx != -1 {
			token := strings.TrimSpace(msg[idx+14:])
			// take only the flag part (stop at space)
			if space := strings.Index(token, " "); space != -1 {
				token = token[:space]
			}
			parsed.BadToken = token
		}
		return parsed
	}

	// "unknown shorthand flag: 'X' in -X"
	if strings.Contains(msg, "unknown shorthand flag:") {
		parsed := &uxfriction.ParsedError{
			Kind:       uxfriction.FailureUnknownFlag,
			RawMessage: msg,
		}
		// extract from single quotes
		if token := extractSingleQuoted(msg); token != "" {
			parsed.BadToken = "-" + token
		}
		return parsed
	}

	// "required flag(s) \"X\", \"Y\" not set"
	if strings.Contains(msg, "required flag") {
		return &uxfriction.ParsedError{
			Kind:       uxfriction.FailureMissingRequired,
			BadToken:   extractQuoted(msg),
			RawMessage: msg,
		}
	}

	// "invalid argument \"X\" for \"--flag\""
	if strings.Contains(msg, "invalid argument") {
		parsed := &uxfriction.ParsedError{
			Kind:       uxfriction.FailureInvalidArg,
			RawMessage: msg,
		}
		if token := extractQuoted(msg); token != "" {
			parsed.BadToken = token
		}
		return parsed
	}

	return nil
}

// walkCommands recursively walks the command tree calling fn for each command.
func walkCommands(cmd *cobra.Command, prefix string, fn func(*cobra.Command, string)) {
	fn(cmd, prefix)
	for _, child := range cmd.Commands() {
		childPrefix := prefix
		if cmd.Name() != "" && prefix == "" {
			childPrefix = cmd.Name()
		} else if prefix != "" {
			childPrefix = prefix + " " + cmd.Name()
		}
		// for root command, prefix stays empty for direct children
		if cmd.Parent() == nil {
			childPrefix = ""
		}
		walkCommands(child, childPrefix, fn)
	}
}

// findCommand locates a command by name (space-separated for nested commands).
// If command is empty, returns the root command.
func (a *CobraAdapter) findCommand(command string) *cobra.Command {
	if command == "" {
		return a.root
	}

	parts := strings.Split(command, " ")
	cmd := a.root
	for _, part := range parts {
		found := false
		for _, child := range cmd.Commands() {
			if child.Name() == part {
				cmd = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return cmd
}

// extractQuoted extracts the first double-quoted string from a message.
func extractQuoted(msg string) string {
	re := regexp.MustCompile(`"([^"]+)"`)
	match := re.FindStringSubmatch(msg)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// extractSingleQuoted extracts the first single-quoted string from a message.
func extractSingleQuoted(msg string) string {
	re := regexp.MustCompile(`'([^']+)'`)
	match := re.FindStringSubmatch(msg)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// Compile-time check that CobraAdapter implements CLIAdapter.
var _ uxfriction.CLIAdapter = (*CobraAdapter)(nil)
