package main

import (
	"os"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/gitcredential"
)

func main() {
	os.Exit(gitcredential.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
