package auth

import (
	"strings"

	"github.com/sageox/ox/internal/config"
	"github.com/sageox/ox/internal/endpoint"
)

// AuthClient provides authentication operations with configurable storage location.
// Use NewAuthClient() for production or NewAuthClientWithDir() for testing.
type AuthClient struct {
	configDir string // base config directory (e.g., ~/.config or test temp dir)
	endpoint  string // API endpoint URL
}

// NewAuthClient creates an AuthClient using the default config directory and endpoint.
func NewAuthClient() *AuthClient {
	return &AuthClient{
		configDir: "", // empty means use default from config.GetUserConfigDir()
		endpoint:  endpoint.Get(),
	}
}

// NewAuthClientWithDir creates an AuthClient with a custom config directory.
// Primarily used for testing to isolate each test's auth storage.
func NewAuthClientWithDir(configDir string) *AuthClient {
	return &AuthClient{
		configDir: configDir,
		endpoint:  endpoint.Get(),
	}
}

// WithEndpoint returns a new AuthClient using the specified API endpoint.
// Trailing slashes are stripped to prevent double slashes in URL paths.
func (c *AuthClient) WithEndpoint(ep string) *AuthClient {
	return &AuthClient{
		configDir: c.configDir,
		endpoint:  strings.TrimSuffix(ep, "/"),
	}
}

// getConfigDir returns the effective config directory.
func (c *AuthClient) getConfigDir() string {
	if c.configDir != "" {
		return c.configDir
	}
	return config.GetUserConfigDir()
}

// Endpoint returns the configured API endpoint.
func (c *AuthClient) Endpoint() string {
	return c.endpoint
}

// defaultClient is the package-level client for backward compatibility.
var defaultClient = NewAuthClient()
