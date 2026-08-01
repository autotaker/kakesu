//go:build darwin || freebsd || openbsd || netbsd || dragonfly

package brokercredentials

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
)

// This reader is for development tests on non-Linux Unix systems. Production
// support is intentionally limited to the Linux descriptor/openat path.
func readSecretFiles(dir string) ([][]byte, error) {
	if !validDirectoryPath(dir) || os.Geteuid() == 0 {
		return nil, ErrLoad
	}
	dirInfo, err := os.Lstat(dir)
	if err != nil || !validNonLinuxDirectory(dirInfo, uint64(os.Geteuid())) {
		return nil, ErrLoad
	}
	directory, err := os.Open(dir)
	if err != nil {
		return nil, ErrLoad
	}
	defer directory.Close()
	files := make([][]byte, 0, len(basenames))
	for _, basename := range basenames {
		data, err := readNonLinuxFile(filepath.Join(dir, basename), uint64(os.Geteuid()))
		if err != nil {
			return nil, ErrLoad
		}
		files = append(files, data)
	}
	return files, nil
}

func readNonLinuxFile(path string, uid uint64) ([]byte, error) {
	lstat, err := os.Lstat(path)
	if err != nil || !validNonLinuxFile(lstat, uid) {
		return nil, ErrLoad
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrLoad
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !sameNonLinuxNode(lstat, before) || !validNonLinuxFile(before, uid) || before.Size() > MaxFileSize {
		return nil, ErrLoad
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, ErrLoad
	}
	if readCompleteHook != nil {
		readCompleteHook()
	}
	after, err := f.Stat()
	if err != nil || !sameNonLinuxMetadata(before, after) || len(data) != int(after.Size()) || len(data) > MaxFileSize {
		return nil, ErrLoad
	}
	return data, nil
}

func validNonLinuxDirectory(info os.FileInfo, uid uint64) bool {
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || ownerID(info) != uid {
		return false
	}
	perm := info.Mode().Perm()
	return perm&0o077 == 0 && perm&0o500 == 0o500
}

func validNonLinuxFile(info os.FileInfo, uid uint64) bool {
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || ownerID(info) != uid {
		return false
	}
	perm := info.Mode().Perm()
	return perm&0o077 == 0 && perm&0o400 != 0 && perm&0o111 == 0 && info.Size() >= 0
}

func ownerID(info os.FileInfo) uint64 {
	value := reflect.ValueOf(info.Sys())
	if value.IsValid() && value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ^uint64(0)
	}
	field := value.FieldByName("Uid")
	if !field.IsValid() {
		return ^uint64(0)
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint()
	default:
		return ^uint64(0)
	}
}

func sameNonLinuxNode(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && ownerID(a) == ownerID(b)
}

func sameNonLinuxMetadata(a, b os.FileInfo) bool {
	return sameNonLinuxNode(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) && a.Mode().Perm() == b.Mode().Perm()
}
