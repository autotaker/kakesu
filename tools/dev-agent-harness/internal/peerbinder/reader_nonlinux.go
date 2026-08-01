//go:build !linux

package peerbinder

import (
	"errors"
	"net"
)

var errPeerCredential = errors.New("peer credential unavailable")

// Peer credentials are deliberately unavailable outside Linux. There is no
// path, address, metadata, or self-declared identity fallback.
func readPeerUID(*net.UnixConn) (int, error) { return 0, errPeerCredential }
