package adapters

import (
	"errors"
	"testing"

	"github.com/sageox/ox/internal/uxfriction"
	"github.com/spf13/cobra"
)

// buildTestCommandTree creates a mock cobra command tree for testing.
// Structure:
//
//	root
//	  |- agent (with --verbose, -v flags)
//	  |    |- prime (with --force, -f flags)
//	  |    |- status
//	  |- config
//	  |    |- set (with --global, -g and --required flags)
//	  |    |- get
//	  |- hidden-cmd (hidden)
//	  |- deprecated-cmd (deprecated)
func buildTestCommandTree() *cobra.Command {
	root := &cobra.Command{
		Use: "ox",
	}

	// agent command with subcommands
	agentCmd := &cobra.Command{
		Use: "agent",
	}
	agentCmd.Flags().BoolP("verbose", "v", false, "verbose output")

	primeCmd := &cobra.Command{
		Use: "prime",
	}
	primeCmd.Flags().BoolP("force", "f", false, "force operation")

	statusCmd := &cobra.Command{
		Use: "status",
	}

	agentCmd.AddCommand(primeCmd, statusCmd)

	// config command with subcommands
	configCmd := &cobra.Command{
		Use: "config",
	}

	setCmd := &cobra.Command{
		Use: "set",
	}
	setCmd.Flags().BoolP("global", "g", false, "global scope")
	setCmd.Flags().String("required", "", "required flag")
	_ = setCmd.MarkFlagRequired("required")

	getCmd := &cobra.Command{
		Use: "get",
	}

	configCmd.AddCommand(setCmd, getCmd)

	// hidden command
	hiddenCmd := &cobra.Command{
		Use:    "hidden-cmd",
		Hidden: true,
	}

	// deprecated command
	deprecatedCmd := &cobra.Command{
		Use:        "deprecated-cmd",
		Deprecated: "use 'new-cmd' instead",
	}

	root.AddCommand(agentCmd, configCmd, hiddenCmd, deprecatedCmd)

	return root
}

func TestNewCobraAdapter(t *testing.T) {
	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	if adapter == nil {
		t.Fatal("expected adapter to be non-nil")
	}

	if adapter.root != root {
		t.Error("expected adapter.root to be the provided root command")
	}
}

func TestCobraAdapter_CommandNames(t *testing.T) {
	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	names := adapter.CommandNames()

	// expected commands (excluding hidden, deprecated, and root)
	expected := map[string]bool{
		"agent":        true,
		"agent prime":  true,
		"agent status": true,
		"config":       true,
		"config set":   true,
		"config get":   true,
	}

	// verify expected commands are present
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	for cmd := range expected {
		if !found[cmd] {
			t.Errorf("expected command %q to be in names", cmd)
		}
	}

	// verify hidden and deprecated commands are excluded
	excluded := []string{"hidden-cmd", "deprecated-cmd"}
	for _, cmd := range excluded {
		if found[cmd] {
			t.Errorf("expected command %q to be excluded", cmd)
		}
	}

	// verify count matches expected
	if len(names) != len(expected) {
		t.Errorf("expected %d commands, got %d: %v", len(expected), len(names), names)
	}
}

func TestCobraAdapter_FlagNames(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "empty command returns root flags",
			command:  "",
			expected: []string{}, // root has no flags in our test tree
		},
		{
			name:    "agent command flags",
			command: "agent",
			expected: []string{
				"--verbose",
				"-v",
			},
		},
		{
			name:    "nested command flags",
			command: "agent prime",
			expected: []string{
				"--force",
				"-f",
			},
		},
		{
			name:    "command with multiple flags",
			command: "config set",
			expected: []string{
				"--global",
				"-g",
				"--required",
			},
		},
		{
			name:     "nonexistent command returns empty",
			command:  "nonexistent",
			expected: []string{},
		},
		{
			name:     "partial nonexistent path returns empty",
			command:  "agent nonexistent",
			expected: []string{},
		},
	}

	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := adapter.FlagNames(tt.command)

			if len(flags) != len(tt.expected) {
				t.Errorf("expected %d flags, got %d: %v", len(tt.expected), len(flags), flags)
				return
			}

			flagSet := make(map[string]bool)
			for _, f := range flags {
				flagSet[f] = true
			}

			for _, exp := range tt.expected {
				if !flagSet[exp] {
					t.Errorf("expected flag %q to be present", exp)
				}
			}
		})
	}
}

func TestCobraAdapter_ParseError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedKind uxfriction.FailureKind
		expectedBad  string
		expectedCmd  string
		expectNil    bool
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			expectNil: true,
		},
		{
			name:         "unknown command error",
			err:          errors.New(`unknown command "foobar" for "ox"`),
			expectedKind: uxfriction.FailureUnknownCommand,
			expectedBad:  "foobar",
			expectedCmd:  "ox",
		},
		{
			name:         "unknown command error with nested parent",
			err:          errors.New(`unknown command "xyz" for "ox agent"`),
			expectedKind: uxfriction.FailureUnknownCommand,
			expectedBad:  "xyz",
			expectedCmd:  "ox agent",
		},
		{
			name:         "unknown flag error",
			err:          errors.New(`unknown flag: --badflg`),
			expectedKind: uxfriction.FailureUnknownFlag,
			expectedBad:  "--badflg",
		},
		{
			name:         "unknown flag error with extra text",
			err:          errors.New(`unknown flag: --badflg some extra text`),
			expectedKind: uxfriction.FailureUnknownFlag,
			expectedBad:  "--badflg",
		},
		{
			name:         "unknown shorthand flag error",
			err:          errors.New(`unknown shorthand flag: 'x' in -x`),
			expectedKind: uxfriction.FailureUnknownFlag,
			expectedBad:  "-x",
		},
		{
			name:         "required flag error single",
			err:          errors.New(`required flag(s) "name" not set`),
			expectedKind: uxfriction.FailureMissingRequired,
			expectedBad:  "name",
		},
		{
			name:         "required flag error multiple",
			err:          errors.New(`required flag(s) "name", "value" not set`),
			expectedKind: uxfriction.FailureMissingRequired,
			expectedBad:  "name", // only first is extracted
		},
		{
			name:         "invalid argument error",
			err:          errors.New(`invalid argument "abc" for "--count" flag: strconv.ParseInt: parsing "abc": invalid syntax`),
			expectedKind: uxfriction.FailureInvalidArg,
			expectedBad:  "abc",
		},
		{
			name:      "unrecognized error returns nil",
			err:       errors.New(`some random error that doesn't match patterns`),
			expectNil: true,
		},
		{
			name:      "generic error without pattern returns nil",
			err:       errors.New(`connection refused`),
			expectNil: true,
		},
	}

	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := adapter.ParseError(tt.err)

			if tt.expectNil {
				if parsed != nil {
					t.Errorf("expected nil, got %+v", parsed)
				}
				return
			}

			if parsed == nil {
				t.Fatal("expected non-nil result")
			}

			if parsed.Kind != tt.expectedKind {
				t.Errorf("expected Kind=%q, got %q", tt.expectedKind, parsed.Kind)
			}

			if parsed.BadToken != tt.expectedBad {
				t.Errorf("expected BadToken=%q, got %q", tt.expectedBad, parsed.BadToken)
			}

			if tt.expectedCmd != "" && parsed.Command != tt.expectedCmd {
				t.Errorf("expected Command=%q, got %q", tt.expectedCmd, parsed.Command)
			}

			if tt.err != nil && parsed.RawMessage != tt.err.Error() {
				t.Errorf("expected RawMessage=%q, got %q", tt.err.Error(), parsed.RawMessage)
			}
		})
	}
}

func TestCobraAdapter_ParseError_EdgeCases(t *testing.T) {
	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	tests := []struct {
		name        string
		err         error
		expectedBad string
	}{
		{
			name:        "unknown command without parent for clause",
			err:         errors.New(`unknown command "test"`),
			expectedBad: "test",
		},
		{
			name:        "unknown flag with equals",
			err:         errors.New(`unknown flag: --bad=value`),
			expectedBad: "--bad=value",
		},
		{
			name:        "shorthand in middle of combined flags",
			err:         errors.New(`unknown shorthand flag: 'z' in -vz`),
			expectedBad: "-z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := adapter.ParseError(tt.err)
			if parsed == nil {
				t.Fatal("expected non-nil result")
			}
			if parsed.BadToken != tt.expectedBad {
				t.Errorf("expected BadToken=%q, got %q", tt.expectedBad, parsed.BadToken)
			}
		})
	}
}

func TestCobraAdapter_ImplementsInterface(t *testing.T) {
	// compile-time check is in cobra.go, but verify at runtime too
	root := buildTestCommandTree()
	var _ uxfriction.CLIAdapter = NewCobraAdapter(root)
}

func TestExtractQuoted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single quoted string",
			input:    `error with "token" here`,
			expected: "token",
		},
		{
			name:     "multiple quoted strings returns first",
			input:    `"first" and "second"`,
			expected: "first",
		},
		{
			name:     "no quoted string",
			input:    `no quotes here`,
			expected: "",
		},
		{
			name:     "empty quoted string",
			input:    `empty "" string`,
			expected: "",
		},
		{
			name:     "nested path in quotes",
			input:    `unknown command "ox agent" for "root"`,
			expected: "ox agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractQuoted(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractSingleQuoted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single quoted character",
			input:    `flag: 'x' in -x`,
			expected: "x",
		},
		{
			name:     "multiple single quoted strings returns first",
			input:    `'first' and 'second'`,
			expected: "first",
		},
		{
			name:     "no single quoted string",
			input:    `no quotes here`,
			expected: "",
		},
		{
			name:     "mixed quotes returns single quoted",
			input:    `"double" and 'single'`,
			expected: "single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSingleQuoted(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFindCommand(t *testing.T) {
	root := buildTestCommandTree()
	adapter := NewCobraAdapter(root)

	tests := []struct {
		name        string
		command     string
		expectFound bool
		expectName  string
	}{
		{
			name:        "empty returns root",
			command:     "",
			expectFound: true,
			expectName:  "ox",
		},
		{
			name:        "top level command",
			command:     "agent",
			expectFound: true,
			expectName:  "agent",
		},
		{
			name:        "nested command",
			command:     "agent prime",
			expectFound: true,
			expectName:  "prime",
		},
		{
			name:        "deeply nested command",
			command:     "config set",
			expectFound: true,
			expectName:  "set",
		},
		{
			name:        "nonexistent command",
			command:     "doesnotexist",
			expectFound: false,
		},
		{
			name:        "partial path nonexistent",
			command:     "agent doesnotexist",
			expectFound: false,
		},
		{
			name:        "nonexistent parent",
			command:     "doesnotexist prime",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := adapter.findCommand(tt.command)

			if tt.expectFound {
				if cmd == nil {
					t.Fatal("expected command to be found")
				}
				if cmd.Name() != tt.expectName {
					t.Errorf("expected Name=%q, got %q", tt.expectName, cmd.Name())
				}
			} else {
				if cmd != nil {
					t.Errorf("expected nil, got command %q", cmd.Name())
				}
			}
		})
	}
}

func TestWalkCommands(t *testing.T) {
	root := buildTestCommandTree()

	var visited []string
	walkCommands(root, "", func(cmd *cobra.Command, prefix string) {
		name := cmd.Name()
		if prefix != "" {
			name = prefix + " " + name
		}
		visited = append(visited, name)
	})

	// should include root and all children
	expected := []string{
		"ox",             // root
		"agent",          // direct child
		"prime",          // nested (prefix handled by walkCommands)
		"status",         // nested
		"config",         // direct child
		"set",            // nested
		"get",            // nested
		"hidden-cmd",     // included in walk (filtering happens in CommandNames)
		"deprecated-cmd", // included in walk
	}

	if len(visited) != len(expected) {
		t.Errorf("expected %d commands visited, got %d: %v", len(expected), len(visited), visited)
	}
}

func TestCobraAdapter_FlagNames_EmptyRoot(t *testing.T) {
	// root with no commands or flags
	root := &cobra.Command{Use: "empty"}
	adapter := NewCobraAdapter(root)

	flags := adapter.FlagNames("")
	if len(flags) != 0 {
		t.Errorf("expected 0 flags for empty root, got %d: %v", len(flags), flags)
	}

	cmds := adapter.CommandNames()
	if len(cmds) != 0 {
		t.Errorf("expected 0 commands for empty root, got %d: %v", len(cmds), cmds)
	}
}

func TestCobraAdapter_FlagNames_PersistentFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().BoolP("debug", "d", false, "debug mode")
	// add a local flag to root as well
	root.Flags().Bool("local", false, "local flag")

	child := &cobra.Command{Use: "child"}
	child.Flags().String("child-flag", "", "child specific flag")
	root.AddCommand(child)

	adapter := NewCobraAdapter(root)

	// root Flags() only returns local flags, not persistent flags
	// persistent flags are accessed via PersistentFlags()
	// the implementation uses cmd.Flags().VisitAll which iterates local flags
	rootFlags := adapter.FlagNames("")
	// expect only --local (persistent flags are not included via Flags())
	hasLocal := false
	for _, f := range rootFlags {
		if f == "--local" {
			hasLocal = true
		}
	}
	if !hasLocal {
		t.Errorf("expected --local flag in root flags, got: %v", rootFlags)
	}

	// child should have its own flags
	childFlags := adapter.FlagNames("child")
	hasChildFlag := false
	for _, f := range childFlags {
		if f == "--child-flag" {
			hasChildFlag = true
		}
	}
	if !hasChildFlag {
		t.Errorf("expected --child-flag in child flags, got: %v", childFlags)
	}
}
