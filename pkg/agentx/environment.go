package agentx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Environment provides access to system environment for agent detection.
// This abstraction enables testing without real file system access.
type Environment interface {
	// GetEnv retrieves an environment variable value
	GetEnv(key string) string

	// LookupEnv retrieves an environment variable and reports if it exists
	LookupEnv(key string) (string, bool)

	// HomeDir returns the user's home directory
	HomeDir() (string, error)

	// ConfigDir returns the XDG config directory
	ConfigDir() (string, error)

	// DataDir returns the XDG data directory
	DataDir() (string, error)

	// CacheDir returns the XDG cache directory
	CacheDir() (string, error)

	// GOOS returns the operating system name
	GOOS() string

	// LookPath searches for an executable in PATH
	LookPath(name string) (string, error)

	// FileExists checks if a file or directory exists
	FileExists(path string) bool

	// IsDir checks if a path is a directory
	IsDir(path string) bool
}

// SystemEnvironment implements Environment using the real system.
type SystemEnvironment struct{}

// NewSystemEnvironment creates a new system environment.
func NewSystemEnvironment() Environment {
	return &SystemEnvironment{}
}

func (e *SystemEnvironment) GetEnv(key string) string {
	return os.Getenv(key)
}

func (e *SystemEnvironment) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (e *SystemEnvironment) HomeDir() (string, error) {
	return os.UserHomeDir()
}

func (e *SystemEnvironment) ConfigDir() (string, error) {
	// XDG_CONFIG_HOME or default
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: prefer ~/.config for CLI tools (XDG-style)
		return filepath.Join(home, ".config"), nil
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData, nil
		}
		return filepath.Join(home, "AppData", "Roaming"), nil
	default:
		return filepath.Join(home, ".config"), nil
	}
}

func (e *SystemEnvironment) DataDir() (string, error) {
	// XDG_DATA_HOME or default
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return dataHome, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".local", "share"), nil
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	default:
		return filepath.Join(home, ".local", "share"), nil
	}
}

func (e *SystemEnvironment) CacheDir() (string, error) {
	// XDG_CACHE_HOME or default
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return cacheHome, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches"), nil
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "Cache"), nil
		}
		return filepath.Join(home, "AppData", "Local", "Cache"), nil
	default:
		return filepath.Join(home, ".cache"), nil
	}
}

func (e *SystemEnvironment) GOOS() string {
	return runtime.GOOS
}

func (e *SystemEnvironment) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (e *SystemEnvironment) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (e *SystemEnvironment) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// MockEnvironment is a test implementation of Environment.
type MockEnvironment struct {
	EnvVars   map[string]string
	Home      string
	Config    string
	Data      string
	Cache     string
	OS        string
	HomeError error
}

// NewMockEnvironment creates a mock environment for testing.
func NewMockEnvironment(envVars map[string]string) *MockEnvironment {
	return &MockEnvironment{
		EnvVars: envVars,
		Home:    "/home/test",
		Config:  "/home/test/.config",
		Data:    "/home/test/.local/share",
		Cache:   "/home/test/.cache",
		OS:      "linux",
	}
}

func (e *MockEnvironment) GetEnv(key string) string {
	if e.EnvVars == nil {
		return ""
	}
	return e.EnvVars[key]
}

func (e *MockEnvironment) LookupEnv(key string) (string, bool) {
	if e.EnvVars == nil {
		return "", false
	}
	val, ok := e.EnvVars[key]
	return val, ok
}

func (e *MockEnvironment) HomeDir() (string, error) {
	if e.HomeError != nil {
		return "", e.HomeError
	}
	return e.Home, nil
}

func (e *MockEnvironment) ConfigDir() (string, error) {
	return e.Config, nil
}

func (e *MockEnvironment) DataDir() (string, error) {
	return e.Data, nil
}

func (e *MockEnvironment) CacheDir() (string, error) {
	return e.Cache, nil
}

func (e *MockEnvironment) GOOS() string {
	if e.OS == "" {
		return "linux"
	}
	return e.OS
}

func (e *MockEnvironment) LookPath(name string) (string, error) {
	// Mock implementation - override in tests if needed
	return "", exec.ErrNotFound
}

func (e *MockEnvironment) FileExists(path string) bool {
	// Mock implementation - override in tests if needed
	return false
}

func (e *MockEnvironment) IsDir(path string) bool {
	// Mock implementation - override in tests if needed
	return false
}
