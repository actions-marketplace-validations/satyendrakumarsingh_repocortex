//go:build windows

package fs

import (
	"fmt"
	"os"
	"syscall"
)

// FileLock represents an exclusive cross-process advisory lock.
type FileLock struct {
	file *os.File
	path string
}

// TryLock attempts to acquire an exclusive, non-blocking lock on path on Windows.
func TryLock(path string) (*FileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lockfile %s: %w", path, err)
	}
	h := syscall.Handle(f.Fd())
	var ol syscall.Overlapped
	err = syscall.LockFileEx(h, syscall.LOCKFILE_EXCLUSIVE_LOCK|syscall.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
	if err != nil {
		_ = f.Close()
		return nil, ErrLocked
	}
	return &FileLock{file: f, path: path}, nil
}

// Unlock releases the lock and closes the lockfile descriptor.
func (l *FileLock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	defer l.file.Close()
	h := syscall.Handle(l.file.Fd())
	var ol syscall.Overlapped
	return syscall.UnlockFileEx(h, 0, 1, 0, &ol)
}
