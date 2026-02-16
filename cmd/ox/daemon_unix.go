//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets the process attributes for background daemon.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // create new session (detach from terminal)
	}
}
