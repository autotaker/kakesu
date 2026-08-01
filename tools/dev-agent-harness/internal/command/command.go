package command

import (
	"fmt"
	"io"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
)

// Version is replaced by the build system for release builds.
var Version = "devel"

// Run implements the fail-closed command surface shared by scaffold binaries.
func Run(name string, args []string, stdout, stderr io.Writer) int {
	if name == "dev-agent-harness-setup" && len(args) > 0 && args[0] == "check-config" {
		return checkConfig(args[1:], stdout, stderr)
	}
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

func checkConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "--config" || args[1] == "" {
		fmt.Fprintln(stderr, "dev-agent-harness-setup: invalid check-config arguments")
		return 2
	}
	if _, err := config.Load(args[1]); err != nil {
		fmt.Fprintf(stderr, "dev-agent-harness-setup: config validation failed (%s)\n", config.ClassOf(err))
		return 1
	}
	fmt.Fprintln(stdout, "config version=1 network.default=deny validated")
	return 0
}
