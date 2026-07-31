package command

import (
	"fmt"
	"io"
)

// Version is replaced by the build system for release builds.
var Version = "devel"

// Run implements the fail-closed command surface shared by scaffold binaries.
func Run(name string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "%s %s\n", name, Version)
		return 0
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(stdout, "usage: %s --version\n", name)
		fmt.Fprintln(stdout, "scaffold only: operational commands are not implemented")
		return 0
	}
	fmt.Fprintf(stderr, "%s: operational behavior is not implemented; refusing to start\n", name)
	return 78
}
