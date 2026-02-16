package cli

import (
	"runtime"
	"testing"
)

func TestIsHeadless_SSHDetection(t *testing.T) {
	// clear all SSH vars (t.Setenv restores on cleanup)
	for _, key := range []string{"SSH_CLIENT", "SSH_CONNECTION", "SSH_TTY"} {
		t.Setenv(key, "")
	}

	// no SSH vars → not headless (on darwin or with DISPLAY)
	if runtime.GOOS == "darwin" {
		if IsHeadless() {
			t.Error("expected IsHeadless()=false on darwin with no SSH vars")
		}
	}

	// SSH_CLIENT set → headless
	t.Setenv("SSH_CLIENT", "192.168.1.1 54321 22")
	if !IsHeadless() {
		t.Error("expected IsHeadless()=true when SSH_CLIENT is set")
	}
	t.Setenv("SSH_CLIENT", "")

	// SSH_TTY set → headless
	t.Setenv("SSH_TTY", "/dev/pts/0")
	if !IsHeadless() {
		t.Error("expected IsHeadless()=true when SSH_TTY is set")
	}
	t.Setenv("SSH_TTY", "")

	// SSH_CONNECTION set → headless
	t.Setenv("SSH_CONNECTION", "192.168.1.1 54321 192.168.1.2 22")
	if !IsHeadless() {
		t.Error("expected IsHeadless()=true when SSH_CONNECTION is set")
	}
}
