package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/command"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(command.RunContext(ctx, "dev-agent-egress", os.Args[1:], os.Stdout, os.Stderr))
}
