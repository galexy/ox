package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TryConnectOrDirect attempts to connect to the daemon.
// Returns nil if daemon is not running or unreachable.
// On transient connection failures, retries once with exponential backoff.
func TryConnectOrDirect() *Client {
	return TryConnectWithRetry(1, 100*time.Millisecond)
}

// TryConnectOrDirectForSync is like TryConnectOrDirect but uses longer timeouts for sync operations.
func TryConnectOrDirectForSync() *Client {
	return TryConnectForSyncWithRetry(1, 100*time.Millisecond)
}

// TryConnectOrDirectForCheckout is like TryConnectOrDirect but uses longer timeouts for checkout operations.
func TryConnectOrDirectForCheckout() *Client {
	return TryConnectForCheckoutWithRetry(1, 100*time.Millisecond)
}

// TryConnectWithRetry attempts to connect to the daemon with retry logic.
// On transient failures, retries up to maxRetries times with exponential backoff.
// initialDelay is the delay before the first retry.
func TryConnectWithRetry(maxRetries int, initialDelay time.Duration) *Client {
	client := NewClient()

	// first attempt
	if err := client.Ping(); err == nil {
		return client
	}

	// retry with exponential backoff
	delay := initialDelay
	for i := 0; i < maxRetries; i++ {
		time.Sleep(delay)

		if err := client.Ping(); err == nil {
			return client
		}

		// exponential backoff: 100ms -> 200ms -> 400ms ...
		delay *= 2
	}

	return nil
}

// TryConnectForSyncWithRetry is like TryConnectWithRetry but uses longer timeouts for sync operations.
func TryConnectForSyncWithRetry(maxRetries int, initialDelay time.Duration) *Client {
	client := NewClientWithTimeout(30 * time.Second)

	// first attempt
	if err := client.Ping(); err == nil {
		return client
	}

	// retry with exponential backoff
	delay := initialDelay
	for i := 0; i < maxRetries; i++ {
		time.Sleep(delay)

		if err := client.Ping(); err == nil {
			return client
		}

		delay *= 2
	}

	return nil
}

// TryConnectForCheckoutWithRetry is like TryConnectWithRetry but uses longer timeouts for checkout operations.
func TryConnectForCheckoutWithRetry(maxRetries int, initialDelay time.Duration) *Client {
	client := NewClientWithTimeout(60 * time.Second)

	// first attempt
	if err := client.Ping(); err == nil {
		return client
	}

	// retry with exponential backoff
	delay := initialDelay
	for i := 0; i < maxRetries; i++ {
		time.Sleep(delay)

		if err := client.Ping(); err == nil {
			return client
		}

		delay *= 2
	}

	return nil
}

// ShouldUseDaemon returns true if we should attempt to use the daemon.
// Returns true if the daemon is currently running.
func ShouldUseDaemon() bool {
	return IsRunning()
}

// EnsureDaemon ensures the daemon is running, starting it if necessary.
// The daemon is started as a detached process (Setsid on Unix) so it survives
// after the parent exits. Used by `ox daemon start` for standalone operation.
// Returns nil on success (daemon is running), or an error if it couldn't be started.
// This is a no-op if daemon is already running or disabled via SAGEOX_DAEMON=false.
//
// The function waits up to 2 seconds for the daemon to become available after starting.
func EnsureDaemon() error {
	if IsDaemonDisabled() {
		return nil
	}
	return ensureDaemonInternal(true)
}

// EnsureDaemonAttached ensures the daemon is running as a child process of the caller.
// Unlike EnsureDaemon, it does NOT detach (no Setsid), so the daemon stays in the
// caller's process group. However, once the caller (ox agent prime) exits, the daemon
// is reparented to PID 1 (launchd/init). macOS has no PR_SET_PDEATHSIG equivalent,
// so there is no automatic signal delivery when the grandparent (coding agent) exits.
// The daemon relies on its inactivity timeout to self-exit when no heartbeats arrive.
// Returns nil on success (daemon is running), or an error if it couldn't be started.
// This is a no-op if daemon is already running or disabled via SAGEOX_DAEMON=false.
func EnsureDaemonAttached() error {
	if IsDaemonDisabled() {
		return nil
	}
	return ensureDaemonInternal(false)
}

// ensureDaemonInternal is the shared implementation for EnsureDaemon and EnsureDaemonAttached.
// When detach is true, the daemon is started in a new session (Setsid) so it survives parent exit.
// When detach is false, the daemon stays in the caller's process group.
func ensureDaemonInternal(detach bool) error {
	if IsRunning() {
		return nil // already running
	}

	// get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// create log directory
	logPath := LogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// start daemon process
	cmd := exec.Command(exe, "daemon", "start", "--foreground")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if detach {
		setSysProcAttr(cmd) // platform-specific detach (Setsid on Unix)
	}
	// NOTE: Pdeathsig won't help here — it tracks the immediate parent
	// (ox agent prime, which exits immediately), not the grandparent (claude).
	// The daemon relies on inactivity timeout to self-exit when claude dies.

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// don't wait for the process
	logFile.Close()

	// wait for daemon to be ready (up to 2 seconds)
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if IsRunning() {
			return nil
		}
	}

	return fmt.Errorf("daemon started but not responding")
}
