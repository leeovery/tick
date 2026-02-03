// Package main is the entry point for the tick CLI.
package main

import (
	"fmt"
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	app := cli.NewApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
