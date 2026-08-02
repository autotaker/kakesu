//go:build darwin || linux

package approvalstate

import (
	"errors"
	"os"
	"syscall"
)

func platformSupported() bool { return true }

func openDirectoryNoFollow(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Uid) == uint64(os.Geteuid())
}

func openProcessLock(root *os.Root) (*os.File, error) {
	existing := false
	var original os.FileInfo
	if info, err := root.Lstat(lockName); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) {
			return nil, newError(ClassPermission)
		}
		existing = true
		original = info
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, newError(ClassPermission)
	}
	flags := os.O_RDWR
	if !existing {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := root.OpenFile(lockName, flags, 0o600)
	if err != nil {
		return nil, newError(ClassPermission)
	}
	if !existing {
		if err := f.Chmod(0o600); err != nil {
			_ = f.Close()
			return nil, newError(ClassPermission)
		}
	}
	current, currentErr := root.Lstat(lockName)
	info, statErr := f.Stat()
	if currentErr != nil || statErr != nil || !os.SameFile(current, info) || (existing && !os.SameFile(original, info)) || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) {
		f.Close()
		return nil, newError(ClassPermission)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, newError(ClassLocked)
	}
	return f, nil
}

func releaseProcessLock(f *os.File) error {
	if f == nil {
		return nil
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
