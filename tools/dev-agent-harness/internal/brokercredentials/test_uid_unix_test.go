//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package brokercredentials

import "os"

func testIsRoot() bool { return os.Geteuid() == 0 }
