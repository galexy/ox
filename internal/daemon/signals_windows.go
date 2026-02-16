//go:build windows

package daemon

import (
	"os"
	"syscall"
)

// shutdownSignals returns the OS signals that should trigger graceful shutdown.
// On Windows, SIGHUP doesn't exist, so we only handle SIGINT and SIGTERM.
func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
