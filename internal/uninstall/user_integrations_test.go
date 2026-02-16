package uninstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sageox/ox/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOxPrimeCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "exact match",
			command:  "ox agent prime",
			expected: true,
		},
		{
			name:     "with user flag",
			command:  "ox agent prime --user",
			expected: true,
		},
		{
			name:     "with fallback",
			command:  constants.OxPrimeCommand,
			expected: true,
		},
		{
			name:     "legacy format",
			command:  "ox prime",
			expected: true,
		},
		{
			name:     "claude code specific command",
			command:  constants.OxPrimeCommandClaudeCode,
			expected: true,
		},
		{
			name:     "gemini specific command",
			command:  constants.OxPrimeCommandGemini,
			expected: true,
		},
		{
			name:     "command with AGENT_ENV prefix",
			command:  "AGENT_ENV=claude-code ox agent prime",
			expected: true,
		},
		{
			name:     "other command",
			command:  "npm install",
			expected: false,
		},
		{
			name:     "empty",
			command:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOxPrimeCommand(tt.command)
			assert.Equal(t, tt.expected, result, "isOxPrimeCommand(%q)", tt.command)
		})
	}
}

func TestContainsOxPrime(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "contains ox agent prime",
			content:  "Run ox agent prime at session start",
			expected: true,
		},
		{
			name:     "contains ox prime",
			content:  "Use ox prime for guidance",
			expected: true,
		},
		{
			name:     "no ox prime",
			content:  "Some other content",
			expected: false,
		},
		{
			name:     "empty",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsOxPrime(tt.content)
			assert.Equal(t, tt.expected, result, "containsOxPrime(%q)", tt.content)
		})
	}
}

func TestFindClaudeHooks(t *testing.T) {
	tmpDir := t.TempDir()
	finder := &UserIntegrationsFinder{homeDir: tmpDir}

	// create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	// test case 1: no settings file
	items, err := finder.findClaudeHooks()
	assert.NoError(t, err)
	assert.Empty(t, items)

	// test case 2: settings file with ox prime hooks
	settingsPath := filepath.Join(claudeDir, "settings.json")
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"command": "ox agent prime",
							"type":    "command",
						},
					},
					"matcher": "",
				},
			},
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(settingsPath, data, 0600)
	require.NoError(t, err)

	items, err = finder.findClaudeHooks()
	assert.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "claude", items[0].Agent)
	assert.Equal(t, "hook", items[0].Type)

	// test case 3: settings file without ox prime hooks
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []map[string]interface{}{
				{
					"hooks": []map[string]interface{}{
						{
							"command": "echo hello",
							"type":    "command",
						},
					},
					"matcher": "",
				},
			},
		},
	}
	data, err = json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(settingsPath, data, 0600)
	require.NoError(t, err)

	items, err = finder.findClaudeHooks()
	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestFindClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()
	finder := &UserIntegrationsFinder{homeDir: tmpDir}

	// test case 1: no CLAUDE.md
	items, err := finder.findClaudeMD()
	assert.NoError(t, err)
	assert.Empty(t, items)

	// test case 2: CLAUDE.md with ox prime
	claudeDir := filepath.Join(tmpDir, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)
	claudeMDPath := filepath.Join(claudeDir, "CLAUDE.md")
	content := "Run ox agent prime at session start"
	err = os.WriteFile(claudeMDPath, []byte(content), 0644)
	require.NoError(t, err)

	items, err = finder.findClaudeMD()
	assert.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "claude", items[0].Agent)
	assert.Equal(t, "config", items[0].Type)

	// test case 3: CLAUDE.md without ox prime
	content = "Some other content"
	err = os.WriteFile(claudeMDPath, []byte(content), 0644)
	require.NoError(t, err)

	items, err = finder.findClaudeMD()
	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestFindOpenCodePlugins(t *testing.T) {
	tmpDir := t.TempDir()
	finder := &UserIntegrationsFinder{homeDir: tmpDir}

	// test case 1: no plugin
	items, err := finder.findOpenCodePlugins()
	assert.NoError(t, err)
	assert.Empty(t, items)

	// test case 2: plugin exists
	pluginDir := filepath.Join(tmpDir, ".config", "opencode", "plugin")
	err = os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	pluginPath := filepath.Join(pluginDir, "ox-prime.ts")
	err = os.WriteFile(pluginPath, []byte("plugin content"), 0644)
	require.NoError(t, err)

	items, err = finder.findOpenCodePlugins()
	assert.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "opencode", items[0].Agent)
	assert.Equal(t, "plugin", items[0].Type)
}

func TestFindGeminiHooks(t *testing.T) {
	tmpDir := t.TempDir()
	finder := &UserIntegrationsFinder{homeDir: tmpDir}

	// test case 1: no settings file
	items, err := finder.findGeminiHooks()
	assert.NoError(t, err)
	assert.Empty(t, items)

	// test case 2: settings file with ox prime hook
	geminiDir := filepath.Join(tmpDir, ".gemini")
	err = os.MkdirAll(geminiDir, 0755)
	require.NoError(t, err)
	settingsPath := filepath.Join(geminiDir, "settings.json")
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": map[string]interface{}{
				"command": "ox agent prime",
				"timeout": 30000,
			},
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(settingsPath, data, 0600)
	require.NoError(t, err)

	items, err = finder.findGeminiHooks()
	assert.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "gemini", items[0].Agent)
	assert.Equal(t, "hook", items[0].Type)
}

func TestRemoveUserIntegrations_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	items := []UserIntegrationItem{
		{
			Path:        testFile,
			Type:        "plugin",
			Agent:       "test",
			Description: "test file",
		},
	}

	// dry run should not remove file
	err = RemoveUserIntegrations(items, true)
	assert.NoError(t, err)

	// file should still exist
	_, err = os.Stat(testFile)
	assert.False(t, os.IsNotExist(err), "file was removed during dry run")
}

func TestRemoveUserIntegrations_ActualRemoval(t *testing.T) {
	tmpDir := t.TempDir()

	// create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	items := []UserIntegrationItem{
		{
			Path:        testFile,
			Type:        "plugin",
			Agent:       "test",
			Description: "test file",
		},
	}

	// actual removal
	err = RemoveUserIntegrations(items, false)
	assert.NoError(t, err)

	// file should be removed
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "file was not removed")
}

func TestRemoveUserIntegrations_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// create test directory with file
	testDir := filepath.Join(tmpDir, "testdir")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	testFile := filepath.Join(testDir, "file.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	items := []UserIntegrationItem{
		{
			Path:        testDir,
			Type:        "plugin",
			Agent:       "test",
			Description: "test directory",
		},
	}

	// actual removal
	err = RemoveUserIntegrations(items, false)
	assert.NoError(t, err)

	// directory should be removed
	_, err = os.Stat(testDir)
	assert.True(t, os.IsNotExist(err), "directory was not removed")
}
