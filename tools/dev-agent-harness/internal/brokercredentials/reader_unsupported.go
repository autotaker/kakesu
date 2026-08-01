//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly

package brokercredentials

// Unsupported systems do not silently weaken the ownership and descriptor
// boundary. Linux is the production implementation; Unix development readers
// are selected in reader_nonlinux.go.
func readSecretFiles(string) ([][]byte, error) { return nil, ErrLoad }
