//go:build windows

package daemon

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x0002
	lockfileFailImmediately = 0x0001
	errorLockViolation      = 33 // ERROR_LOCK_VIOLATION
)

// tryLock attempts to acquire an exclusive lock on the file.
// Returns true if lock was acquired, false if already locked.
func tryLock(f *os.File) (bool, error) {
	handle := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)

	// try to acquire exclusive lock without blocking
	r1, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0, // lock 1 byte
		uintptr(unsafe.Pointer(ol)),
	)

	if r1 == 0 {
		// check if error is "another process has locked"
		if errno, ok := err.(syscall.Errno); ok && errno == errorLockViolation {
			return false, nil // already locked
		}
		return false, err
	}
	return true, nil
}

// unlock releases the lock on the file.
func unlock(f *os.File) error {
	handle := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)

	r1, _, err := procUnlockFileEx.Call(
		uintptr(handle),
		0,
		1, 0, // unlock 1 byte
		uintptr(unsafe.Pointer(ol)),
	)

	if r1 == 0 {
		return err
	}
	return nil
}
