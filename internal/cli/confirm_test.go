package cli

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShowUninstallPreview(t *testing.T) {
	tests := []struct {
		name    string
		items   []RemovalItem
		dryRun  bool
		wantOut []string // expected output substrings
	}{
		{
			name:    "empty items shows info message",
			items:   []RemovalItem{},
			dryRun:  false,
			wantOut: []string{"No SageOx files found"},
		},
		{
			name: "single item displays correctly",
			items: []RemovalItem{
				{Type: "directory", Path: ".sageox", Description: "SageOx state directory"},
			},
			dryRun:  false,
			wantOut: []string{"Directory (1)", ".sageox", "SageOx state directory"},
		},
		{
			name: "dry run mode shows warning",
			items: []RemovalItem{
				{Type: "file", Path: ".sageox/config.yml", Description: ""},
			},
			dryRun:  true,
			wantOut: []string{"DRY RUN MODE", "File (1)", ".sageox/config.yml"},
		},
		{
			name: "multiple types grouped and ordered",
			items: []RemovalItem{
				{Type: "file", Path: "AGENTS.md", Description: "agent guidance"},
				{Type: "directory", Path: ".sageox", Description: "state dir"},
				{Type: "hook", Path: ".git/hooks/pre-commit", Description: "git hook"},
				{Type: "file", Path: ".sageox/config.yml", Description: "config"},
			},
			dryRun: false,
			wantOut: []string{
				"Directory (1)", ".sageox",
				"File (2)", "AGENTS.md", ".sageox/config.yml",
				"Hook (1)", ".git/hooks/pre-commit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ShowUninstallPreview(tt.items, tt.dryRun)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// verify expected strings appear in output
			for _, want := range tt.wantOut {
				assert.Contains(t, output, want, "ShowUninstallPreview() output missing %q", want)
			}
		})
	}
}

func TestConfirmUninstall(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		force     bool
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "force flag bypasses prompt",
			repoName:  "test-repo",
			force:     true,
			input:     "",
			wantError: false,
		},
		{
			name:      "correct input confirms",
			repoName:  "test-repo",
			force:     false,
			input:     "uninstall\n",
			wantError: false,
		},
		{
			name:      "incorrect input cancels",
			repoName:  "test-repo",
			force:     false,
			input:     "nope\n",
			wantError: true,
			errorMsg:  "confirmation failed",
		},
		{
			name:      "empty input cancels",
			repoName:  "test-repo",
			force:     false,
			input:     "\n",
			wantError: true,
			errorMsg:  "confirmation failed",
		},
		{
			name:      "case sensitive match required",
			repoName:  "test-repo",
			force:     false,
			input:     "UNINSTALL\n",
			wantError: true,
			errorMsg:  "confirmation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// skip stdin tests when force is true
			if !tt.force {
				// mock stdin
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				w.Write([]byte(tt.input))
				w.Close()
				defer func() { os.Stdin = oldStdin }()
			}

			// capture stdout to avoid test noise
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			err := ConfirmUninstall(tt.repoName, tt.force)

			if tt.wantError {
				assert.Error(t, err, "ConfirmUninstall() expected error")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "ConfirmUninstall() error message")
				}
			} else {
				assert.NoError(t, err, "ConfirmUninstall() unexpected error")
			}
		})
	}
}

func TestConfirmDangerousOperation(t *testing.T) {
	tests := []struct {
		name          string
		operationName string
		exactMatch    string
		force         bool
		input         string
		wantError     bool
	}{
		{
			name:          "force bypasses confirmation",
			operationName: "delete everything",
			exactMatch:    "DELETE",
			force:         true,
			input:         "",
			wantError:     false,
		},
		{
			name:          "exact match confirms",
			operationName: "remove repo",
			exactMatch:    "my-repo-name",
			force:         false,
			input:         "my-repo-name\n",
			wantError:     false,
		},
		{
			name:          "wrong match cancels",
			operationName: "remove repo",
			exactMatch:    "my-repo-name",
			force:         false,
			input:         "wrong\n",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.force {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				w.Write([]byte(tt.input))
				w.Close()
				defer func() { os.Stdin = oldStdin }()
			}

			// capture output to avoid test noise
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			os.Stdout, _ = os.Open(os.DevNull)
			os.Stderr, _ = os.Open(os.DevNull)
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			err := ConfirmDangerousOperation(tt.operationName, tt.exactMatch, tt.force)

			if tt.wantError {
				assert.Error(t, err, "ConfirmDangerousOperation() expected error")
			} else {
				assert.NoError(t, err, "ConfirmDangerousOperation() unexpected error")
			}
		})
	}
}

func TestConfirmYesNo(t *testing.T) {
	tests := []struct {
		name       string
		prompt     string
		defaultYes bool
		input      string
		want       bool
	}{
		{
			name:       "yes input returns true",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "yes\n",
			want:       true,
		},
		{
			name:       "y input returns true",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "y\n",
			want:       true,
		},
		{
			name:       "no input returns false",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "no\n",
			want:       false,
		},
		{
			name:       "n input returns false",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "n\n",
			want:       false,
		},
		{
			name:       "empty uses default false",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "\n",
			want:       false,
		},
		{
			name:       "empty uses default true",
			prompt:     "Continue?",
			defaultYes: true,
			input:      "\n",
			want:       true,
		},
		{
			name:       "case insensitive",
			prompt:     "Continue?",
			defaultYes: false,
			input:      "YES\n",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			w.Write([]byte(tt.input))
			w.Close()
			defer func() { os.Stdin = oldStdin }()

			// capture stdout
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			got := ConfirmYesNo(tt.prompt, tt.defaultYes)

			assert.Equal(t, tt.want, got, "ConfirmYesNo()")
		})
	}
}

func TestRemovalItemGrouping(t *testing.T) {
	items := []RemovalItem{
		{Type: "file", Path: "file1.txt", Description: "test file 1"},
		{Type: "directory", Path: "dir1", Description: "test dir"},
		{Type: "file", Path: "file2.txt", Description: "test file 2"},
		{Type: "hook", Path: "hook1", Description: "test hook"},
		{Type: "custom", Path: "custom1", Description: "custom type"},
	}

	// capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowUninstallPreview(items, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// verify types are grouped
	assert.Contains(t, output, "Directory (1)", "Expected Directory group with count")
	assert.Contains(t, output, "File (2)", "Expected File group with count of 2")
	assert.Contains(t, output, "Hook (1)", "Expected Hook group")
	assert.Contains(t, output, "Custom (1)", "Expected Custom group for unknown type")
}

func TestDangerousOperationWarning(t *testing.T) {
	// capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DangerousOperationWarning("UNINSTALL", "my-repo")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedStrings := []string{
		"DESTRUCTIVE OPERATION",
		"UNINSTALL",
		"my-repo",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "DangerousOperationWarning() output missing %q", expected)
	}
}
