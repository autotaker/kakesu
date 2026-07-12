package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kakesu/core/internal/app"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New().Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kakesu: %v\n", err)
		os.Exit(1)
	}
}
