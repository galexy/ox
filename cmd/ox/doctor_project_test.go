//go:build !short

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sageox/ox/internal/config"
)

// testGitRepoWithSageox creates a git repo with .sageox directory
func testGitRepoWithSageox(t *testing.T) string {
	t.Helper()

	gitRoot := testGitRepo(t)
	requireSageoxDir(t, gitRoot)

	return gitRoot
}

func TestCheckSageoxDirectory_Exists(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkSageoxDirectory()

	if !result.passed {
		t.Errorf("expected passed=true when .sageox exists, got false")
	}
	if result.name != ".sageox/ directory" {
		t.Errorf("unexpected check name: %s", result.name)
	}
}

func TestCheckSageoxDirectory_Missing(t *testing.T) {
	gitRoot := testGitRepo(t)

	// change to git directory (no .sageox yet)
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkSageoxDirectory()

	if result.passed {
		t.Errorf("expected passed=false when .sageox missing, got true")
	}
	if result.message != "not found" {
		t.Errorf("expected message='not found', got '%s'", result.message)
	}
	if result.detail != "Run `ox init` to create it" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckLegacySageoxMd_None(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkLegacySageoxMd()

	if !result.passed {
		t.Errorf("expected passed=true when no legacy files, got false")
	}
	if result.warning {
		t.Errorf("expected warning=false when no legacy files, got true")
	}
	if result.message != "none" {
		t.Errorf("expected message='none', got '%s'", result.message)
	}
}

func TestCheckLegacySageoxMd_RootExists(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// create SAGEOX.md in root
	sageoxPath := filepath.Join(gitRoot, "SAGEOX.md")
	if err := os.WriteFile(sageoxPath, []byte("# Legacy SAGEOX.md\n"), 0644); err != nil {
		t.Fatalf("failed to create SAGEOX.md: %v", err)
	}

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkLegacySageoxMd()

	if !result.passed {
		t.Errorf("expected passed=true (with warning), got false")
	}
	if !result.warning {
		t.Errorf("expected warning=true when legacy file exists, got false")
	}
	if result.message != "found: SAGEOX.md" {
		t.Errorf("expected message about SAGEOX.md, got '%s'", result.message)
	}
	if result.detail != "SAGEOX.md is deprecated; use `ox agent prime` for guidance" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckLegacySageoxMd_DotSageoxExists(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// create SAGEOX.md in .sageox
	sageoxPath := filepath.Join(gitRoot, ".sageox", "SAGEOX.md")
	if err := os.WriteFile(sageoxPath, []byte("# Legacy .sageox/SAGEOX.md\n"), 0644); err != nil {
		t.Fatalf("failed to create .sageox/SAGEOX.md: %v", err)
	}

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkLegacySageoxMd()

	if !result.passed {
		t.Errorf("expected passed=true (with warning), got false")
	}
	if !result.warning {
		t.Errorf("expected warning=true when legacy file exists, got false")
	}
	if result.message != "found: .sageox/SAGEOX.md" {
		t.Errorf("expected message about .sageox/SAGEOX.md, got '%s'", result.message)
	}
	if result.detail != "SAGEOX.md is deprecated; use `ox agent prime` for guidance" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckLegacySageoxMd_BothExist(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// create both legacy files
	rootPath := filepath.Join(gitRoot, "SAGEOX.md")
	if err := os.WriteFile(rootPath, []byte("# Root SAGEOX.md\n"), 0644); err != nil {
		t.Fatalf("failed to create SAGEOX.md: %v", err)
	}

	sageoxPath := filepath.Join(gitRoot, ".sageox", "SAGEOX.md")
	if err := os.WriteFile(sageoxPath, []byte("# .sageox/SAGEOX.md\n"), 0644); err != nil {
		t.Fatalf("failed to create .sageox/SAGEOX.md: %v", err)
	}

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	result := checkLegacySageoxMd()

	if !result.passed {
		t.Errorf("expected passed=true (with warning), got false")
	}
	if !result.warning {
		t.Errorf("expected warning=true when legacy files exist, got false")
	}
	if result.message != "found: SAGEOX.md, .sageox/SAGEOX.md" {
		t.Errorf("expected message listing both files, got '%s'", result.message)
	}
	if result.detail != "SAGEOX.md is deprecated; use `ox agent prime` for guidance" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckConfigFile_Missing(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// config.json doesn't exist
	result := checkConfigFile(false)

	if result.passed {
		t.Errorf("expected passed=false, got true")
	}
	if result.message != "not found" {
		t.Errorf("expected message='not found', got '%s'", result.message)
	}
	// when .sageox exists but config missing, suggest doctor --fix (incomplete init)
	if result.detail != "Run `ox doctor --fix` to create it" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckConfigFile_Exists(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create a valid config
	cfg := config.GetDefaultProjectConfig()
	if err := config.SaveProjectConfig(gitRoot, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	result := checkConfigFile(false)

	if !result.passed {
		t.Errorf("expected passed=true, got false")
	}
	if result.skipped {
		t.Errorf("expected skipped=false, got true")
	}
	if result.warning {
		t.Errorf("expected warning=false, got true")
	}
	expectedMsg := "v" + config.CurrentConfigVersion
	if result.message != expectedMsg {
		t.Errorf("expected message='%s', got '%s'", expectedMsg, result.message)
	}
}

func TestCheckConfigFile_Invalid(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create invalid JSON
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	invalidJSON := `{"invalid": json`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	result := checkConfigFile(false)

	if result.passed {
		t.Errorf("expected passed=false for invalid JSON")
	}
	if result.message != "parse error" {
		t.Errorf("expected message='parse error', got '%s'", result.message)
	}
	if result.detail == "" {
		t.Error("expected detail to contain error information")
	}
}

func TestCheckConfigFile_FixCreatesNewConfig(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// config.json doesn't exist
	result := checkConfigFile(true)

	if !result.passed {
		t.Errorf("expected passed=true after fix, got false")
	}
	if result.message != "created" {
		t.Errorf("expected message='created', got '%s'", result.message)
	}

	// verify config was actually created
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.json was not created by fix")
	}

	// verify it's valid
	cfg, err := config.LoadProjectConfig(gitRoot)
	if err != nil {
		t.Fatalf("failed to load created config: %v", err)
	}

	if cfg.ConfigVersion != config.CurrentConfigVersion {
		t.Errorf("expected version %s, got %s", config.CurrentConfigVersion, cfg.ConfigVersion)
	}
}

func TestCheckConfigFile_FixUpgradesOldVersion(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create an old version config
	oldConfig := map[string]interface{}{
		"config_version":         "1",
		"update_frequency_hours": 24,
	}
	data, _ := json.MarshalIndent(oldConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write old config: %v", err)
	}

	result := checkConfigFile(true)

	if !result.passed {
		t.Errorf("expected passed=true after upgrade, got false")
	}
	if result.message != "upgraded to v"+config.CurrentConfigVersion {
		t.Errorf("expected upgrade message, got '%s'", result.message)
	}

	// verify version was upgraded
	cfg, err := config.LoadProjectConfig(gitRoot)
	if err != nil {
		t.Fatalf("failed to load upgraded config: %v", err)
	}

	if cfg.ConfigVersion != config.CurrentConfigVersion {
		t.Errorf("expected version %s after upgrade, got %s", config.CurrentConfigVersion, cfg.ConfigVersion)
	}
}

func TestCheckConfigFile_NeedsUpgrade_NoFix(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create old version config
	oldConfig := map[string]interface{}{
		"config_version":         "1",
		"update_frequency_hours": 24,
	}
	data, _ := json.MarshalIndent(oldConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write old config: %v", err)
	}

	result := checkConfigFile(false)

	if !result.passed {
		t.Errorf("expected passed=true (with warning), got false")
	}
	if !result.warning {
		t.Error("expected warning=true for outdated config")
	}
	if result.message != "outdated (v1)" {
		t.Errorf("unexpected message: %s", result.message)
	}
	if !strings.Contains(result.detail, "ox doctor --fix") {
		t.Error("expected detail to mention 'ox doctor --fix'")
	}
	if !strings.Contains(result.detail, "v"+config.CurrentConfigVersion) {
		t.Errorf("expected detail to mention current version v%s", config.CurrentConfigVersion)
	}
}

func TestCheckConfigFields_ValidVersion(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create a valid current config
	cfg := config.GetDefaultProjectConfig()
	if err := config.SaveProjectConfig(gitRoot, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	result := checkConfigFields(false)

	if !result.passed {
		t.Errorf("expected passed=true for valid config, got false")
	}
	if result.message != "" {
		t.Errorf("expected empty message for valid config, got '%s'", result.message)
	}
}

func TestCheckConfigFields_OldVersion(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create an old version config
	oldConfig := map[string]interface{}{
		"config_version":         "1",
		"update_frequency_hours": 24,
	}
	data, _ := json.MarshalIndent(oldConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write old config: %v", err)
	}

	// checkConfigFields validates fields, not version - that's checkConfigFile's job
	// but if config is valid per field validation, it should pass
	result := checkConfigFields(false)

	if !result.passed {
		t.Errorf("expected passed=true for valid fields, got false: %s", result.message)
	}
}

func TestCheckConfigFields_InvalidFields(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create config with invalid fields
	invalidConfig := map[string]interface{}{
		"config_version":         config.CurrentConfigVersion,
		"update_frequency_hours": 0, // invalid - must be > 0
	}
	data, _ := json.MarshalIndent(invalidConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	result := checkConfigFields(false)

	if result.passed {
		t.Errorf("expected passed=false for invalid fields, got true")
	}
	if result.message == "" {
		t.Error("expected error message for invalid fields")
	}
	if !strings.Contains(result.message, "update_frequency_hours") {
		t.Errorf("expected message to mention update_frequency_hours, got: %s", result.message)
	}
	if result.detail != "Run `ox doctor --fix`" {
		t.Errorf("unexpected detail: %s", result.detail)
	}
}

func TestCheckConfigFields_FixRepairs(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create config with invalid fields
	invalidConfig := map[string]interface{}{
		"config_version":         config.CurrentConfigVersion,
		"update_frequency_hours": -1, // invalid
	}
	data, _ := json.MarshalIndent(invalidConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	result := checkConfigFields(true)

	if !result.passed {
		t.Errorf("expected passed=true after fix, got false")
	}
	if result.message != "fixed" {
		t.Errorf("expected message='fixed', got '%s'", result.message)
	}

	// verify fields were fixed
	cfg, err := config.LoadProjectConfig(gitRoot)
	if err != nil {
		t.Fatalf("failed to load fixed config: %v", err)
	}

	errors := config.ValidateProjectConfig(cfg)
	if len(errors) > 0 {
		t.Errorf("expected no validation errors after fix, got: %v", errors)
	}
}

func TestCheckConfigFields_ParseError(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create invalid JSON
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	invalidJSON := `{invalid json content`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	result := checkConfigFields(false)

	if result.passed {
		t.Errorf("expected passed=false for parse error, got true")
	}
	if result.message != "parse failed" {
		t.Errorf("expected message='parse failed', got '%s'", result.message)
	}
	if result.detail == "" {
		t.Error("expected detail to contain error information")
	}
}

func TestCheckConfigFields_ReadError(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// don't create config.json - test read failure
	result := checkConfigFields(false)

	if result.passed {
		t.Errorf("expected passed=false for read error, got true")
	}
	if result.message != "read failed" {
		t.Errorf("expected message='read failed', got '%s'", result.message)
	}
	if result.detail == "" {
		t.Error("expected detail to contain error information")
	}
}

// TestCheckConfigFields_ValidationErrors tests that checkConfigFields reports validation errors
func TestCheckConfigFields_ValidationErrors(t *testing.T) {
	gitRoot := testGitRepoWithSageox(t)

	// change to git directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(gitRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// create config with validation error
	invalidConfig := map[string]interface{}{
		"config_version":         config.CurrentConfigVersion,
		"update_frequency_hours": 0, // invalid
	}
	data, _ := json.MarshalIndent(invalidConfig, "", "  ")
	configPath := filepath.Join(gitRoot, ".sageox", "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	result := checkConfigFields(false)

	if result.passed {
		t.Error("expected passed=false for validation error")
	}
}
