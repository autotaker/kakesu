//go:build linux

package socketactivation

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLinuxMetadataPredicatesRejectEachDrift(t *testing.T) {
	uid, gid := os.Geteuid(), syscall.Getgid()
	directory := syscall.Stat_t{Mode: syscall.S_IFDIR | 0o710, Uid: uint32(uid), Gid: uint32(gid)}
	for _, tc := range []struct {
		name string
		stat syscall.Stat_t
		want bool
	}{
		{"valid-directory", directory, true},
		{"directory-mode", syscall.Stat_t{Mode: syscall.S_IFDIR | 0o750, Uid: uint32(uid), Gid: uint32(gid)}, false},
		{"directory-type", syscall.Stat_t{Mode: syscall.S_IFSOCK | 0o710, Uid: uint32(uid), Gid: uint32(gid)}, false},
		{"directory-owner", syscall.Stat_t{Mode: syscall.S_IFDIR | 0o710, Uid: uint32(uid + 1), Gid: uint32(gid)}, false},
		{"directory-group", syscall.Stat_t{Mode: syscall.S_IFDIR | 0o710, Uid: uint32(uid), Gid: uint32(gid + 1)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validRuntimeDirectory(&tc.stat, uid, gid); got != tc.want {
				t.Fatalf("validRuntimeDirectory=%v want %v", got, tc.want)
			}
		})
	}
	socket := syscall.Stat_t{Mode: syscall.S_IFSOCK | 0o660, Uid: uint32(uid), Gid: uint32(gid)}
	for _, tc := range []struct {
		name string
		stat syscall.Stat_t
		want bool
	}{
		{"valid-socket", socket, true},
		{"socket-type", syscall.Stat_t{Mode: syscall.S_IFREG | 0o660, Uid: uint32(uid), Gid: uint32(gid)}, false},
		{"socket-owner", syscall.Stat_t{Mode: syscall.S_IFSOCK | 0o660, Uid: uint32(uid + 1), Gid: uint32(gid)}, false},
		{"socket-group", syscall.Stat_t{Mode: syscall.S_IFSOCK | 0o660, Uid: uint32(uid), Gid: uint32(gid + 1)}, false},
		{"socket-mode", syscall.Stat_t{Mode: syscall.S_IFSOCK | 0o600, Uid: uint32(uid), Gid: uint32(gid)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validSocketNode(&tc.stat, uid, gid); got != tc.want {
				t.Fatalf("validSocketNode=%v want %v", got, tc.want)
			}
		})
	}
}

func TestLinuxMetadataReaderAcceptsAndRejectsActualUnixSocket(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root broker is intentionally denied")
	}
	runtimeDir := t.TempDir()
	if err := os.Chmod(runtimeDir, 0o710); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(runtimeDir, socketName)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close(); _ = os.Remove(socketPath) })
	if err := os.Chmod(socketPath, 0o660); err != nil {
		t.Fatal(err)
	}
	if err := validatePlatform(runtimeDir, os.Geteuid(), syscall.Getgid()); err != nil {
		t.Fatalf("valid metadata rejected: %v", err)
	}
	for _, tc := range []struct {
		name      string
		brokerUID int
		agentGID  int
	}{
		{"owner-mismatch", os.Geteuid() + 1, syscall.Getgid()},
		{"group-mismatch", os.Geteuid(), syscall.Getgid() + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validatePlatform(runtimeDir, tc.brokerUID, tc.agentGID); err == nil {
				t.Fatal("identity mismatch unexpectedly accepted")
			}
		})
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validatePlatform(runtimeDir, os.Geteuid(), syscall.Getgid()); err == nil {
		t.Fatal("mode drift unexpectedly accepted")
	}
	if _, err := os.Lstat(socketPath); err != nil {
		t.Fatalf("metadata failure unlinked socket: %v", err)
	}
	if err := os.Chmod(socketPath, 0o660); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(socketPath, socketPath+".real"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socketPath, []byte("not a socket"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validatePlatform(runtimeDir, os.Geteuid(), syscall.Getgid()); err == nil {
		t.Fatal("regular file socket node unexpectedly accepted")
	}
	if _, err := os.Lstat(socketPath); err != nil {
		t.Fatalf("regular-file metadata failure removed node: %v", err)
	}
	if err := os.Remove(socketPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(socketPath+".real", socketPath); err != nil {
		t.Fatal(err)
	}
	if err := validatePlatform(runtimeDir, os.Geteuid(), syscall.Getgid()); err == nil {
		t.Fatal("symlink socket node unexpectedly accepted")
	}
	if _, err := os.Lstat(socketPath); err != nil {
		t.Fatalf("symlink metadata failure removed node: %v", err)
	}
}
