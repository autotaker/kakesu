//go:build !linux

package socketactivation

import "os"

func activationRawEnvironment() ([]string, bool) { return os.Environ(), true }

// Activation is intentionally unavailable outside Linux. There is no
// self-declared identity or filesystem fallback for the inherited descriptor.
func validatePlatform(string, int, int) error { return ErrDenied }
