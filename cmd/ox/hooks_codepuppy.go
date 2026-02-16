package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sageox/ox/internal/ui"
)

// code_puppy plugin template - runs ox agent prime on session start
const codePuppyPluginTemplate = `"""SageOx plugin for code_puppy - runs ox agent prime on session start"""
import subprocess
from code_puppy.callbacks import register_callback

async def on_startup():
    """Run ox agent prime when code_puppy session starts"""
    try:
        subprocess.run(["ox", "agent", "prime"], capture_output=True, check=False)
    except FileNotFoundError:
        pass  # ox not installed

register_callback("startup", on_startup)
`

// code_puppy paths
const (
	codePuppyPluginDir      = "ox_prime"
	codePuppyPluginFileName = "register_callbacks.py"
	codePuppyUserPath       = ".code_puppy/plugins"
	codePuppyProjectPath    = ".code_puppy"
)

// getCodePuppyPluginPath returns the path to the code_puppy plugin file
func getCodePuppyPluginPath(user bool) (string, error) {
	if user {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, codePuppyUserPath, codePuppyPluginDir, codePuppyPluginFileName), nil
	}
	// project-level - code_puppy doesn't support project-level plugins in the same way
	// but we support it for consistency with other agents
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, codePuppyProjectPath, "plugins", codePuppyPluginDir, codePuppyPluginFileName), nil
}

// hasCodePuppyHooks checks if the code_puppy plugin exists
func hasCodePuppyHooks(user bool) bool {
	path, err := getCodePuppyPluginPath(user)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// installCodePuppyHooks installs the ox prime plugin for code_puppy
func installCodePuppyHooks(user bool) error {
	path, err := getCodePuppyPluginPath(user)
	if err != nil {
		return err
	}

	// check if already installed with same content
	if existingContent, err := os.ReadFile(path); err == nil {
		if string(existingContent) == codePuppyPluginTemplate {
			fmt.Println(ui.PassStyle.Render("✓") + " code_puppy plugin already installed at " + path)
			return nil
		}
	}

	// create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// write plugin file
	if err := os.WriteFile(path, []byte(codePuppyPluginTemplate), pluginPerm); err != nil {
		return fmt.Errorf("failed to write plugin file: %w", err)
	}

	return nil
}

// uninstallCodePuppyHooks removes the ox prime plugin for code_puppy
func uninstallCodePuppyHooks(user bool) error {
	path, err := getCodePuppyPluginPath(user)
	if err != nil {
		return err
	}

	// check if exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("code_puppy plugin not found at " + path)
		return nil
	}

	// remove plugin file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove plugin file: %w", err)
	}

	// try to remove the plugin directory if empty
	pluginDir := filepath.Dir(path)
	_ = os.Remove(pluginDir) // ignore errors if not empty

	return nil
}

// listCodePuppyHooks returns the installation status of code_puppy plugins
func listCodePuppyHooks() map[string]bool {
	return map[string]bool{
		"User plugin": hasCodePuppyHooks(true),
	}
}
