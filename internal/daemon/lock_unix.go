//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// tryLock attempts to acquire an exclusive lock on the file.
// Returns true if lock was acquired, false if already locked.
func tryLock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return false, nil // already locked
		}
		return false, err
	}
	return true, nil
}

// unlock releases the lock on the file.
func unlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
