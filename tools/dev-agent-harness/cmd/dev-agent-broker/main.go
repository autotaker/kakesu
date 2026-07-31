package main

import (
	"os"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/command"
)

func main() {
	os.Exit(command.Run("dev-agent-broker", os.Args[1:], os.Stdout, os.Stderr))
}
