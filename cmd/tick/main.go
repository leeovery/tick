// Package main is the entry point for the tick CLI.
package main

import (
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	isTTY := cli.DetectTTY(os.Stdout)
	cwd, err := os.Getwd()
	if err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
	code := cli.Run(os.Args, cwd, os.Stdout, os.Stderr, isTTY)
	os.Exit(code)
}
