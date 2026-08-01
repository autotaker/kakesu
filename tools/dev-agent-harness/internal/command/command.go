package command

import (
	"context"
	"fmt"
	"io"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egressservice"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/provision"
)

// Version is replaced by the build system for release builds.
var Version = "devel"

// runEgressService is package-private so command tests can exercise argument,
// cancellation, and fixed-diagnostic handling without starting systemd.
var runEgressService = egressservice.Serve

// Run implements the fail-closed command surface shared by scaffold binaries.
func Run(name string, args []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), name, args, stdout, stderr)
}

// RunContext is the command boundary used by binaries that translate
// termination signals into cooperative context cancellation.
func RunContext(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
	if name == "dev-agent-harness-setup" && len(args) > 0 && args[0] == "check-config" {
		return checkConfig(args[1:], stdout, stderr)
	}
	if name == "dev-agent-harness-setup" && len(args) > 0 && args[0] == "plan-provision" {
		return planProvision(args[1:], stdout, stderr)
	}
	if name == "dev-agent-harness-setup" && len(args) > 0 && args[0] == "verify-provision" {
		return verifyProvision(args[1:], stdout, stderr)
	}
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "%s %s\n", name, Version)
		return 0
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(stdout, "usage: %s --version\n", name)
		if name == "dev-agent-egress" {
			fmt.Fprintln(stdout, "usage: dev-agent-egress serve --config PATH")
		} else {
			fmt.Fprintln(stdout, "scaffold only: operational commands are not implemented")
		}
		return 0
	}
	if name == "dev-agent-egress" && len(args) > 0 {
		if args[0] != "serve" {
			fmt.Fprintln(stderr, "dev-agent-egress: invalid serve arguments")
			return 2
		}
		return serveEgress(ctx, args[1:], stderr)
	}
	fmt.Fprintf(stderr, "%s: operational behavior is not implemented; refusing to start\n", name)
	return 78
}

func serveEgress(ctx context.Context, args []string, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "--config" || args[1] == "" {
		fmt.Fprintln(stderr, "dev-agent-egress: invalid serve arguments")
		return 2
	}
	if err := runEgressService(ctx, args[1]); err != nil {
		fmt.Fprintln(stderr, "dev-agent-egress: service start failed")
		return 1
	}
	return 0
}

func planProvision(args []string, stdout, stderr io.Writer) int {
	if len(args) != 4 || args[0] != "--config" || args[1] == "" || args[2] != "--target-root" || args[3] == "" {
		fmt.Fprintln(stderr, "dev-agent-harness-setup: invalid plan-provision arguments")
		return 2
	}
	c, err := config.Load(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "dev-agent-harness-setup: config validation failed (%s)\n", config.ClassOf(err))
		return 1
	}
	if err := provision.Write(c, args[3], stdout); err != nil {
		fmt.Fprintf(stderr, "dev-agent-harness-setup: provision planning failed (%s)\n", provision.ClassOf(err))
		return 1
	}
	return 0
}

func verifyProvision(args []string, stdout, stderr io.Writer) int {
	if len(args) != 6 || args[0] != "--config" || args[1] == "" ||
		args[2] != "--manifest" || args[3] == "" ||
		args[4] != "--target-root" || args[5] == "" {
		fmt.Fprintln(stderr, "dev-agent-harness-setup: invalid verify-provision arguments")
		return 2
	}
	c, err := config.Load(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "dev-agent-harness-setup: config validation failed (%s)\n", config.ClassOf(err))
		return 1
	}
	if err := provision.Verify(c, args[3], args[5]); err != nil {
		fmt.Fprintf(stderr, "dev-agent-harness-setup: provision verification failed (%s)\n", provision.ClassOf(err))
		return 1
	}
	fmt.Fprintln(stdout, "provision manifest version=1 actions=11 verified")
	return 0
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
