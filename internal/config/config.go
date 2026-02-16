package config

import (
	"os"
)

type Config struct {
	Verbose       bool
	Quiet         bool
	JSON          bool
	Text          bool // human-readable text output (overrides JSON default)
	Review        bool // security audit mode: both human summary and machine output
	NoInteractive bool // disable spinners and TUI elements (auto-enabled in CI)
}

// Load creates a Config from environment variables only.
// Runtime flags (--verbose, --json, etc.) are applied by the cobra layer in cli/context.go.
func Load() *Config {
	return &Config{
		Verbose:       os.Getenv("OX_VERBOSE") == "1",
		Quiet:         os.Getenv("OX_QUIET") == "1",
		JSON:          os.Getenv("OX_JSON") == "1",
		Text:          os.Getenv("OX_TEXT") == "1",
		Review:        os.Getenv("OX_REVIEW") == "1",
		NoInteractive: os.Getenv("OX_NO_INTERACTIVE") == "1" || isCI(),
	}
}

// isCI returns true if running in a CI environment.
// Checks standard CI environment variables used by major CI providers.
func isCI() bool {
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE", "CODEBUILD_BUILD_ID"}
	for _, v := range ciVars {
		if val := os.Getenv(v); val != "" && val != "false" && val != "0" {
			return true
		}
	}
	return false
}
