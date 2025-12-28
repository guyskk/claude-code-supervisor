// ccc - Claude Code Configuration Switcher
// A CLI tool to switch between multiple Claude Code API provider configurations.
package main

import (
	"fmt"
	"os"

	"github.com/user/ccc/internal/cli"
)

// Version is set by build flags during release.
var Version = "dev"

// BuildTime is set by build flags during release (ISO 8601 format).
var BuildTime = "unknown"

func init() {
	// Set version info for cli package
	cli.Version = Version
	cli.BuildTime = BuildTime
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return cli.Execute()
}
