package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sageox/ox/internal/ui"
)

// GeminiHook represents a hook configuration for Gemini CLI
type GeminiHook struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// GeminiSettings represents the structure of ~/.gemini/settings.json
type GeminiSettings struct {
	Hooks map[string]GeminiHook `json:"hooks,omitempty"`
}

// getGeminiSettingsPath returns the path to Gemini CLI settings.json
func getGeminiSettingsPath(user bool) (string, error) {
	if user {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, geminiUserPath, geminiSettingsFileName), nil
	}
	// project-level
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, geminiProjectPath, geminiSettingsFileName), nil
}

// readGeminiSettings reads and parses Gemini CLI settings.json
func readGeminiSettings(user bool) (*GeminiSettings, error) {
	path, err := getGeminiSettingsPath(user)
	if err != nil {
		return nil, err
	}

	// return empty settings if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &GeminiSettings{
			Hooks: make(map[string]GeminiHook),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	// handle empty file
	if len(data) == 0 {
		return &GeminiSettings{
			Hooks: make(map[string]GeminiHook),
		}, nil
	}

	var settings GeminiSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	if settings.Hooks == nil {
		settings.Hooks = make(map[string]GeminiHook)
	}

	return &settings, nil
}

// writeGeminiSettings writes Gemini CLI settings.json
func writeGeminiSettings(user bool, settings *GeminiSettings) error {
	path, err := getGeminiSettingsPath(user)
	if err != nil {
		return err
	}

	// create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, settingsPerm); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// hasGeminiHooks checks if ox prime hooks exist in Gemini CLI settings
func hasGeminiHooks(user bool) bool {
	settings, err := readGeminiSettings(user)
	if err != nil {
		return false
	}

	hook, exists := settings.Hooks[geminiSessionStart]
	return exists && isOxPrimeCommand(hook.Command)
}

// installGeminiHooks installs ox prime hooks for Gemini CLI
func installGeminiHooks(user bool) error {
	settings, err := readGeminiSettings(user)
	if err != nil {
		return err
	}

	// check if already installed
	if hook, exists := settings.Hooks[geminiSessionStart]; exists && isOxPrimeCommand(hook.Command) {
		path, _ := getGeminiSettingsPath(user)
		fmt.Println(ui.PassStyle.Render("✓") + " Gemini CLI hooks already installed at " + path)
		return nil
	}

	// add SessionStart hook
	// Use Gemini-specific command with AGENT_ENV=gemini prefix
	settings.Hooks[geminiSessionStart] = GeminiHook{
		Command: oxPrimeCommandGemini,
		Timeout: defaultHookTimeout,
	}

	return writeGeminiSettings(user, settings)
}

// uninstallGeminiHooks removes ox prime hooks from Gemini CLI
func uninstallGeminiHooks(user bool) error {
	settings, err := readGeminiSettings(user)
	if err != nil {
		return err
	}

	// check if hook exists
	hook, exists := settings.Hooks[geminiSessionStart]
	if !exists || !isOxPrimeCommand(hook.Command) {
		path, _ := getGeminiSettingsPath(user)
		fmt.Println("Gemini CLI ox prime hook not found at " + path)
		return nil
	}

	// remove the hook
	delete(settings.Hooks, geminiSessionStart)

	return writeGeminiSettings(user, settings)
}

// listGeminiHooks returns the installation status of Gemini CLI hooks
func listGeminiHooks() map[string]bool {
	return map[string]bool{
		"Project": hasGeminiHooks(false),
		"User":    hasGeminiHooks(true),
	}
}
