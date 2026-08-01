//go:build linux

package socketactivation

import (
	"io"
	"os"
	"strings"
	"syscall"
)

const rawEnvironmentLimit = 64 * 1024
const linuxOPath = 0x200000

func activationRawEnvironment() ([]string, bool) {
	file, err := os.Open("/proc/self/environ")
	if err != nil {
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, rawEnvironmentLimit+1))
	if err != nil || len(data) == 0 || len(data) > rawEnvironmentLimit || data[len(data)-1] != 0 {
		return nil, false
	}
	data = data[:len(data):len(data)]
	parts := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
	if len(parts) == 1 && parts[0] == "" {
		return nil, true
	}
	return parts, true
}

func validatePlatform(runtimeDir string, brokerUID, agentGID int) error {
	if os.Geteuid() != brokerUID || os.Geteuid() == 0 {
		return ErrDenied
	}
	dirFD, err := syscall.Open(runtimeDir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return ErrDenied
	}
	defer syscall.Close(dirFD)
	var dirStat syscall.Stat_t
	if err := syscall.Fstat(dirFD, &dirStat); err != nil || !validRuntimeDirectory(&dirStat, brokerUID, agentGID) {
		return ErrDenied
	}

	socketFD, err := openSocketNode(dirFD, socketName)
	if err != nil {
		return ErrDenied
	}
	defer syscall.Close(socketFD)
	var socketStat syscall.Stat_t
	if err := syscall.Fstat(socketFD, &socketStat); err != nil || !validSocketNode(&socketStat, brokerUID, agentGID) {
		return ErrDenied
	}
	return nil
}

func validRuntimeDirectory(stat *syscall.Stat_t, brokerUID, agentGID int) bool {
	return stat != nil && stat.Mode&syscall.S_IFMT == syscall.S_IFDIR && stat.Uid == uint32(brokerUID) && stat.Gid == uint32(agentGID) && stat.Mode&0o7777 == 0o710
}

func validSocketNode(stat *syscall.Stat_t, brokerUID, agentGID int) bool {
	return stat != nil && stat.Mode&syscall.S_IFMT == syscall.S_IFSOCK && stat.Uid == uint32(brokerUID) && stat.Gid == uint32(agentGID) && stat.Mode&0o7777 == 0o660
}

func openSocketNode(dirFD int, name string) (int, error) {
	return syscall.Openat(dirFD, name, linuxOPath|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
}
