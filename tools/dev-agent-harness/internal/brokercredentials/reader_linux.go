//go:build linux

package brokercredentials

import (
	"io"
	"os"
	"syscall"
)

type linuxMeta struct {
	dev, ino            uint64
	mode, uid, gid      uint64
	size, nlink         int64
	mtimeSec, mtimeNsec int64
	ctimeSec, ctimeNsec int64
}

func readSecretFiles(dir string) ([][]byte, error) {
	if !validDirectoryPath(dir) || os.Geteuid() == 0 {
		return nil, ErrLoad
	}
	fd, err := syscall.Open(dir, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_DIRECTORY, 0)
	if err != nil {
		return nil, ErrLoad
	}
	directory := os.NewFile(uintptr(fd), "broker-credentials-directory")
	if directory == nil {
		_ = syscall.Close(fd)
		return nil, ErrLoad
	}
	defer directory.Close()
	var directoryStat syscall.Stat_t
	if syscall.Fstat(fd, &directoryStat) != nil || !validLinuxDirectory(directoryStat) {
		return nil, ErrLoad
	}

	files := make([][]byte, 0, len(basenames))
	for _, basename := range basenames {
		data, err := readSecretAt(fd, basename, uint64(os.Geteuid()))
		if err != nil {
			return nil, ErrLoad
		}
		files = append(files, data)
	}
	return files, nil
}

func readSecretAt(dirfd int, basename string, uid uint64) ([]byte, error) {
	fd, err := syscall.Openat(dirfd, basename, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrLoad
	}
	f := os.NewFile(uintptr(fd), "broker-credential")
	if f == nil {
		_ = syscall.Close(fd)
		return nil, ErrLoad
	}
	defer f.Close()

	before, err := linuxFileMeta(fd)
	if err != nil || !validLinuxFile(before, uid) || before.size < 0 || before.size > MaxFileSize {
		return nil, ErrLoad
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, ErrLoad
	}
	if readCompleteHook != nil {
		readCompleteHook()
	}
	after, err := linuxFileMeta(fd)
	if err != nil || !sameLinuxMeta(before, after) || len(data) != int(after.size) || len(data) > MaxFileSize {
		return nil, ErrLoad
	}
	return data, nil
}

func linuxFileMeta(fd int) (linuxMeta, error) {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return linuxMeta{}, ErrLoad
	}
	return linuxMeta{
		dev: uint64(st.Dev), ino: st.Ino, mode: uint64(st.Mode), uid: uint64(st.Uid), gid: uint64(st.Gid),
		size: st.Size, nlink: int64(st.Nlink), mtimeSec: st.Mtim.Sec, mtimeNsec: int64(st.Mtim.Nsec),
		ctimeSec: st.Ctim.Sec, ctimeNsec: int64(st.Ctim.Nsec),
	}, nil
}

func validLinuxDirectory(st syscall.Stat_t) bool {
	if uint64(st.Uid) != uint64(os.Geteuid()) || st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		return false
	}
	perm := uint64(st.Mode) & 0o777
	return perm&0o077 == 0 && perm&0o500 == 0o500
}

func validLinuxFile(st linuxMeta, uid uint64) bool {
	if st.uid != uid || st.mode&uint64(syscall.S_IFMT) != uint64(syscall.S_IFREG) || st.nlink != 1 {
		return false
	}
	perm := st.mode & 0o777
	return perm&0o077 == 0 && perm&0o400 != 0 && perm&0o111 == 0
}

func sameLinuxMeta(a, b linuxMeta) bool {
	return a.dev == b.dev && a.ino == b.ino && a.mode == b.mode && a.uid == b.uid && a.gid == b.gid &&
		a.size == b.size && a.nlink == b.nlink && a.mtimeSec == b.mtimeSec && a.mtimeNsec == b.mtimeNsec &&
		a.ctimeSec == b.ctimeSec && a.ctimeNsec == b.ctimeNsec
}
