//go:build linux

package peerbinder

import (
	"errors"
	"net"
	"syscall"
)

var errPeerCredential = errors.New("peer credential unavailable")

func readPeerUID(conn *net.UnixConn) (int, error) {
	if conn == nil {
		return 0, errPeerCredential
	}
	raw, err := conn.SyscallConn()
	if err != nil || raw == nil {
		return 0, errPeerCredential
	}
	var uid uint32
	var readErr error
	if err := raw.Control(func(fd uintptr) {
		credential, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil || credential == nil {
			readErr = errPeerCredential
			return
		}
		uid = credential.Uid
	}); err != nil || readErr != nil || uint64(uid) > uint64(^uint(0)>>1) {
		return 0, errPeerCredential
	}
	return int(uid), nil
}
