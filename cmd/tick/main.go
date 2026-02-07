// Package main is the entry point for the tick CLI.
package main

import (
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	app := &cli.App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Dir:    ".",
	}

	// Resolve working directory
	if wd, err := os.Getwd(); err == nil {
		app.Dir = wd
	}

	os.Exit(app.Run(os.Args))
}
