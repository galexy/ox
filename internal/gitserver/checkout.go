package gitserver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sageox/ox/internal/logger"
)

var (
	// ErrPathExists is returned when the checkout path already exists and is not empty
	ErrPathExists = errors.New("path already exists and is not empty")

	// ErrNoCredentials is returned when credentials are not available
	ErrNoCredentials = errors.New("git credentials not available")

	// ErrRepoNotFound is returned when the requested repo is not in credentials
	ErrRepoNotFound = errors.New("repository not found in credentials")

	// ErrCloneFailed is returned when git clone fails
	ErrCloneFailed = errors.New("git clone failed")

	// ErrEmptyURL is returned when an empty repo URL is provided
	ErrEmptyURL = errors.New("repo URL cannot be empty")
)

// CheckoutOptions configures the checkout behavior
type CheckoutOptions struct {
	// Depth sets shallow clone depth (0 = full clone)
	Depth int

	// Branch specifies the branch to checkout (empty = default branch)
	Branch string
}

// DefaultCheckoutPath returns the default checkout path for a repo.
// For ledger repos, defaults to sibling directory of the working directory.
// For team repos, defaults to a team-specific subdirectory.
func DefaultCheckoutPath(repoName, workDir string) string {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	parentDir := filepath.Dir(workDir)
	return filepath.Join(parentDir, repoName)
}

// CheckoutLedgerWithURL clones the ledger repo to the specified path.
// The repoURL MUST be provided - it should be fetched from the cloud API.
// Uses stored credentials for authentication only.
func CheckoutLedgerWithURL(ctx context.Context, repoURL, path string, opts *CheckoutOptions) error {
	if repoURL == "" {
		return ErrEmptyURL
	}

	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	if creds == nil {
		return ErrNoCredentials
	}

	if path == "" {
		path = DefaultCheckoutPath("ledger", "")
	}

	return cloneRepo(ctx, repoURL, path, creds, opts)
}

// CheckoutTeamContextWithURL clones a team context repo to the specified path.
// The repoURL MUST be provided - it should be fetched from the cloud API.
// The teamID is used for logging/tracking purposes only.
// Uses stored credentials for authentication only.
func CheckoutTeamContextWithURL(ctx context.Context, teamID, repoURL, path string, opts *CheckoutOptions) error {
	if repoURL == "" {
		return ErrEmptyURL
	}

	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	if creds == nil {
		return ErrNoCredentials
	}

	if path == "" {
		// derive repo name from URL for default path
		repoName := repoNameFromURL(repoURL)
		if repoName == "" {
			repoName = fmt.Sprintf("team-%s-context", teamID)
		}
		path = DefaultCheckoutPath(repoName, "")
	}

	logger.Debug("checkout team context", "team_id", teamID, "path", path)
	return cloneRepo(ctx, repoURL, path, creds, opts)
}

// repoNameFromURL extracts the repository name from a git URL.
// Returns empty string if parsing fails or URL is empty.
func repoNameFromURL(repoURL string) string {
	if repoURL == "" {
		return ""
	}

	// handle SSH URLs (git@host:path/repo.git)
	if isSSHURL(repoURL) {
		parts := strings.Split(repoURL, ":")
		if len(parts) == 2 {
			path := parts[1]
			base := filepath.Base(path)
			return strings.TrimSuffix(base, ".git")
		}
		return ""
	}

	// handle HTTPS URLs
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return ""
	}

	// url.Parse("") returns no error but has empty path
	if parsed.Path == "" {
		return ""
	}

	base := filepath.Base(parsed.Path)
	// filepath.Base returns "." for empty paths
	if base == "." || base == "/" {
		return ""
	}
	return strings.TrimSuffix(base, ".git")
}

// CloneFromURL clones a repository directly from a URL.
// Uses stored credentials for authentication but ignores the cached URL.
// This is useful when the URL is fetched fresh from the cloud API.
func CloneFromURL(ctx context.Context, repoURL, path string, opts *CheckoutOptions) error {
	if repoURL == "" {
		return ErrEmptyURL
	}

	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	if creds == nil {
		return ErrNoCredentials
	}

	return cloneRepo(ctx, repoURL, path, creds, opts)
}

// Deprecated: CheckoutTeamLedger looks up the URL from cached credentials.
// Use CheckoutTeamContextWithURL instead - fetch the URL from the cloud API first.
// This function will be removed in a future version.
func CheckoutTeamLedger(ctx context.Context, teamID, path string, opts *CheckoutOptions) error {
	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds == nil {
		return ErrNoCredentials
	}

	// look for team repo by team ID
	repoKey := fmt.Sprintf("team-%s", teamID)
	repo := creds.GetRepo(repoKey)
	if repo == nil {
		// try alternate naming patterns
		for name, entry := range creds.Repos {
			if strings.Contains(name, teamID) && entry.Type == "team-context" {
				repo = &entry
				break
			}
		}
	}
	if repo == nil {
		return fmt.Errorf("%w: team context for %s", ErrRepoNotFound, teamID)
	}

	if path == "" {
		path = DefaultCheckoutPath(repo.Name, "")
	}

	return cloneRepo(ctx, repo.URL, path, creds, opts)
}

// Deprecated: CheckoutRepo looks up the URL from cached credentials.
// Use CloneFromURL instead - fetch the URL from the cloud API first.
// This function will be removed in a future version.
func CheckoutRepo(ctx context.Context, repoName, path string, opts *CheckoutOptions) error {
	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds == nil {
		return ErrNoCredentials
	}

	repo := creds.GetRepo(repoName)
	if repo == nil {
		return fmt.Errorf("%w: %s", ErrRepoNotFound, repoName)
	}

	if path == "" {
		path = DefaultCheckoutPath(repoName, "")
	}

	return cloneRepo(ctx, repo.URL, path, creds, opts)
}

// cloneRepo performs the actual git clone operation with credentials
func cloneRepo(ctx context.Context, repoURL, path string, creds *GitCredentials, opts *CheckoutOptions) error {
	// validate path
	if err := validateCheckoutPath(path); err != nil {
		return err
	}

	// build authenticated URL
	authURL, err := buildAuthURL(repoURL, creds)
	if err != nil {
		return fmt.Errorf("failed to build authenticated URL: %w", err)
	}

	// build git clone command
	args := []string{"clone"}

	if opts != nil {
		if opts.Depth > 0 {
			args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
		}
		if opts.Branch != "" {
			args = append(args, "--branch", opts.Branch)
		}
	}

	args = append(args, authURL, path)

	// log without exposing token
	logger.Debug("git clone", "repo", sanitizeURL(repoURL), "path", path)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stderr = os.Stderr // show git progress/errors

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%w: exit code %d\n  debug (contains credentials): git %s", ErrCloneFailed, exitErr.ExitCode(), strings.Join(args, " "))
		}
		return fmt.Errorf("%w: %w\n  debug (contains credentials): git %s", ErrCloneFailed, err, strings.Join(args, " "))
	}

	logger.Info("repository cloned", "path", path)
	return nil
}

// validateCheckoutPath ensures the path is valid for checkout
func validateCheckoutPath(path string) error {
	// convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// path doesn't exist - this is fine, git will create it
			return nil
		}
		return fmt.Errorf("failed to check path: %w", err)
	}

	// path exists - check if it's an empty directory
	if !info.IsDir() {
		return fmt.Errorf("%w: path is a file", ErrPathExists)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("%w: %s", ErrPathExists, absPath)
	}

	return nil
}

// isSSHURL checks if the URL is an SSH-style git URL (git@host:path format)
func isSSHURL(repoURL string) bool {
	// SSH URLs look like: git@github.com:user/repo.git
	// They don't have a scheme and contain a colon after the host
	return strings.Contains(repoURL, "@") && !strings.Contains(repoURL, "://")
}

// buildAuthURL embeds credentials into the git URL for authentication.
// Uses the PAT token with oauth2 username for GitLab-style auth.
// SSH URLs are returned unchanged since they use SSH key auth.
// Supports https:// URLs and http://localhost URLs (for local development).
func buildAuthURL(repoURL string, creds *GitCredentials) (string, error) {
	if creds == nil || creds.Token == "" {
		return repoURL, nil
	}

	// SSH URLs use key-based auth, not token auth
	if isSSHURL(repoURL) {
		return repoURL, nil
	}

	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL: %w", err)
	}

	// add auth for https URLs
	if parsed.Scheme == "https" {
		parsed.User = url.UserPassword("oauth2", creds.Token)
		return parsed.String(), nil
	}

	// add auth for http://localhost URLs (local development)
	// this is safe because traffic never leaves the machine
	if parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1") {
		parsed.User = url.UserPassword("oauth2", creds.Token)
		return parsed.String(), nil
	}

	// don't add credentials to other http:// URLs (security risk)
	return repoURL, nil
}

// sanitizeURL removes credentials from a URL for logging.
// Returns the original string for SSH URLs or unparseable URLs.
func sanitizeURL(repoURL string) string {
	// SSH URLs don't need sanitization
	if isSSHURL(repoURL) {
		return repoURL
	}

	parsed, err := url.Parse(repoURL)
	if err != nil {
		return repoURL
	}
	parsed.User = nil
	return parsed.String()
}

// IsGitInstalled checks if git is available in PATH
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetGitVersion returns the installed git version
func GetGitVersion() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// output is "git version X.Y.Z"
	version := strings.TrimSpace(string(output))
	return strings.TrimPrefix(version, "git version "), nil
}
