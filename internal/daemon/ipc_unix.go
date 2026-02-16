//go:build !windows

package daemon

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// listen creates a Unix socket listener with owner-only permissions.
// SECURITY: Socket is created mode 0600 to prevent other users from connecting.
// Only the socket owner can send/receive messages, including credentials.
func listen(path string) (net.Listener, error) {
	// ensure parent directory exists (owner-only for security)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	// remove existing socket file
	os.Remove(path)

	// set restrictive umask before creating socket (mode 0600)
	oldMask := syscall.Umask(0077)
	listener, err := net.Listen("unix", path)
	syscall.Umask(oldMask)

	return listener, err
}

// dial connects to a Unix socket with a timeout.
// Uses 5 second timeout to prevent indefinite hangs if daemon is stuck.
func dial(path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, 5*time.Second)
}

// cleanupSocket removes the socket file.
func cleanupSocket(path string) {
	os.Remove(path)
}
