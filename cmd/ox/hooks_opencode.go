package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sageox/ox/internal/ui"
)

// opencode plugin template
const openCodePluginTemplate = `import type { Plugin } from "@opencode-ai/plugin"

// SageOx plugin for OpenCode
// runs 'ox agent prime' on session start to load team context
export const OxPrimePlugin: Plugin = async ({ $, directory }) => {
  return {
    event: async ({ event }) => {
      if (event.type === "session.created") {
        try {
          await $` + "`ox agent prime`" + `.cwd(directory).quiet()
        } catch {
          // ox not installed or failed - continue silently
        }
      }
    }
  }
}
`

// getOpenCodePluginPath returns the path to the OpenCode plugin file
func getOpenCodePluginPath(user bool) (string, error) {
	if user {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, openCodeUserPath, openCodePluginFileName), nil
	}
	// project-level
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, openCodeProjectPath, openCodePluginFileName), nil
}

// hasOpenCodeHooks checks if the OpenCode plugin exists
func hasOpenCodeHooks(user bool) bool {
	path, err := getOpenCodePluginPath(user)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// installOpenCodeHooks installs the ox prime plugin for OpenCode
func installOpenCodeHooks(user bool) error {
	path, err := getOpenCodePluginPath(user)
	if err != nil {
		return err
	}

	// check if already installed with same content
	if existingContent, err := os.ReadFile(path); err == nil {
		if string(existingContent) == openCodePluginTemplate {
			fmt.Println(ui.PassStyle.Render("✓") + " OpenCode plugin already installed at " + path)
			return nil
		}
	}

	// create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// write plugin file
	if err := os.WriteFile(path, []byte(openCodePluginTemplate), pluginPerm); err != nil {
		return fmt.Errorf("failed to write plugin file: %w", err)
	}

	return nil
}

// uninstallOpenCodeHooks removes the ox prime plugin for OpenCode
func uninstallOpenCodeHooks(user bool) error {
	path, err := getOpenCodePluginPath(user)
	if err != nil {
		return err
	}

	// check if exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("OpenCode plugin not found at " + path)
		return nil
	}

	// remove plugin file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove plugin file: %w", err)
	}

	return nil
}

// listOpenCodeHooks returns the installation status of OpenCode plugins
func listOpenCodeHooks() map[string]bool {
	return map[string]bool{
		"Project plugin": hasOpenCodeHooks(false),
		"User plugin":    hasOpenCodeHooks(true),
	}
}

// InstallProjectOpenCodeHooks installs the ox prime plugin to a specific project directory
func InstallProjectOpenCodeHooks(gitRoot string) error {
	path := filepath.Join(gitRoot, openCodeProjectPath, openCodePluginFileName)

	// check if already installed with same content
	if existingContent, err := os.ReadFile(path); err == nil {
		if string(existingContent) == openCodePluginTemplate {
			return nil // already installed
		}
	}

	// create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// write plugin file
	if err := os.WriteFile(path, []byte(openCodePluginTemplate), pluginPerm); err != nil {
		return fmt.Errorf("failed to write plugin file: %w", err)
	}

	return nil
}

// HasProjectOpenCodeHooks checks if ox prime plugin exists in a specific project directory
func HasProjectOpenCodeHooks(gitRoot string) bool {
	path := filepath.Join(gitRoot, openCodeProjectPath, openCodePluginFileName)
	_, err := os.Stat(path)
	return err == nil
}
