// Package main is the entry point for the tick CLI tool.
package main

import (
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	app := &cli.App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getwd:  os.Getwd,
		IsTTY:  cli.IsTerminal(os.Stdout),
	}
	os.Exit(app.Run(os.Args))
}
