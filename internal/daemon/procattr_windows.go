//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets the process attributes for background daemon.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
