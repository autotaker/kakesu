//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package brokercredentials

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRejectsFIFOWithoutBlocking(t *testing.T) {
	requireNonRoot(t)
	_, dir := validFixture(t, "RSA PRIVATE KEY")
	fifo := filepath.Join(t.TempDir(), "credential-fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("FIFO unsupported: %v", err)
	}
	path := filepath.Join(dir, proxyCACert)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(fifo, path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); !errors.Is(err, ErrLoad) {
		t.Fatalf("FIFO accepted or blocked: %v", err)
	}
}
