//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// shutdownSignals returns the OS signals that should trigger graceful shutdown.
// On Unix, includes SIGTERM (sent by process group cleanup when parent exits)
// and SIGHUP (sent when controlling terminal is closed).
func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
