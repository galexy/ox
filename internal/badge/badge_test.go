package badge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindReadme(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		wantExists bool
	}{
		{
			name:       "finds README.md",
			files:      []string{"README.md"},
			wantExists: true,
		},
		{
			name:       "finds readme.md (lowercase)",
			files:      []string{"readme.md"},
			wantExists: true,
		},
		{
			name:       "finds Readme.md",
			files:      []string{"Readme.md"},
			wantExists: true,
		},
		{
			name:       "no readme found",
			files:      []string{"other.txt"},
			wantExists: false,
		},
		{
			name:       "empty directory",
			files:      []string{},
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// create test files
			for _, f := range tt.files {
				path := filepath.Join(tmpDir, f)
				require.NoError(t, os.WriteFile(path, []byte("# Test"), 0644), "failed to create test file")
			}

			got := FindReadme(tmpDir)

			if tt.wantExists {
				assert.NotEmpty(t, got, "FindReadme() = empty, want a readme file")
				if got != "" {
					// verify the returned file actually exists
					_, err := os.Stat(got)
					assert.False(t, os.IsNotExist(err), "FindReadme() returned non-existent file: %s", got)
				}
			} else {
				assert.Empty(t, got, "FindReadme() = %s, want empty", got)
			}
		})
	}
}

func TestFindReadmePreference(t *testing.T) {
	// test that README.md is preferred over other variants
	tmpDir := t.TempDir()

	// create both files
	for _, f := range []string{"README.md", "readme.md"} {
		path := filepath.Join(tmpDir, f)
		require.NoError(t, os.WriteFile(path, []byte("# Test"), 0644), "failed to create test file")
	}

	got := FindReadme(tmpDir)
	require.NotEmpty(t, got, "FindReadme() returned empty")

	// README.md should be preferred (it comes first in the candidate list)
	assert.Equal(t, "README.md", filepath.Base(got), "FindReadme() should prefer uppercase README.md")
}

func TestHasBadge(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "has full badge markdown",
			content: "# My Project\n\n" + BadgeMarkdown + "\n\nSome content",
			want:    true,
		},
		{
			name:    "has SageOx reference",
			content: "# My Project\n\nPowered by SageOx\n",
			want:    true,
		},
		{
			name:    "has sageox lowercase",
			content: "# My Project\n\nUsing sageox for infra\n",
			want:    true,
		},
		{
			name:    "has sageox/ox link",
			content: "# My Project\n\n[link](https://github.com/sageox/ox)\n",
			want:    true,
		},
		{
			name:    "no badge present",
			content: "# My Project\n\nSome content\n",
			want:    false,
		},
		{
			name:    "empty file",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			readmePath := filepath.Join(tmpDir, "README.md")

			require.NoError(t, os.WriteFile(readmePath, []byte(tt.content), 0644), "failed to create test file")

			assert.Equal(t, tt.want, HasBadge(readmePath), "HasBadge()")
		})
	}
}

func TestInjectBadge(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantContain string
	}{
		{
			name:        "injects after heading",
			content:     "# My Project\n\nSome content",
			wantContain: BadgeMarkdown,
		},
		{
			name:        "injects after existing badges",
			content:     "# My Project\n\n[![Build](https://example.com/badge.svg)](https://example.com)\n\nContent",
			wantContain: BadgeMarkdown,
		},
		{
			name:        "handles empty file",
			content:     "",
			wantContain: BadgeMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			readmePath := filepath.Join(tmpDir, "README.md")

			require.NoError(t, os.WriteFile(readmePath, []byte(tt.content), 0644), "failed to create test file")

			require.NoError(t, InjectBadge(readmePath), "InjectBadge() error")

			content, err := os.ReadFile(readmePath)
			require.NoError(t, err, "failed to read result")

			assert.Contains(t, string(content), tt.wantContain, "InjectBadge() result does not contain badge")
		})
	}
}

func TestFindBadgeInsertionPoint(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
	}{
		{
			name:  "after heading",
			lines: []string{"# Title", "", "Content"},
			want:  1,
		},
		{
			name:  "after existing badge",
			lines: []string{"# Title", "[![Badge](url)](link)", "", "Content"},
			want:  2,
		},
		{
			name:  "after multiple badges",
			lines: []string{"# Title", "[![Badge1](url1)](link1)", "[![Badge2](url2)](link2)", "", "Content"},
			want:  3,
		},
		{
			name:  "empty file",
			lines: []string{},
			want:  0,
		},
		{
			name:  "no heading",
			lines: []string{"Just content", "More content"},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, findBadgeInsertionPoint(tt.lines), "findBadgeInsertionPoint()")
		})
	}
}

func TestBadgeMarkdown(t *testing.T) {
	// verify badge contains expected elements
	assert.Contains(t, BadgeMarkdown, "powered%20by-SageOx", "BadgeMarkdown should contain 'powered by SageOx'")
	assert.Contains(t, BadgeMarkdown, "7A8F78", "BadgeMarkdown should contain SageOx brand color")
	assert.Contains(t, BadgeMarkdown, "github.com/sageox/ox", "BadgeMarkdown should link to ox repo")
}
