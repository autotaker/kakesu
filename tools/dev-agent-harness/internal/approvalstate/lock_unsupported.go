//go:build !darwin && !linux

package approvalstate

import "os"

func platformSupported() bool { return false }

func openDirectoryNoFollow(string) (*os.File, error) {
	return nil, newError(ClassUnsupported)
}

func ownedByCurrentUser(os.FileInfo) bool { return false }

func openProcessLock(*os.Root) (*os.File, error) {
	return nil, newError(ClassUnsupported)
}

func releaseProcessLock(*os.File) error { return nil }
