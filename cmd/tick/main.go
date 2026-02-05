package main

import (
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	app := &cli.App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Cwd:    getCwd(),
	}

	os.Exit(app.Run(os.Args))
}

func getCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
