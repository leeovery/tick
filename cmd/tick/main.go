package main

import (
	"os"

	"github.com/leeovery/tick/internal/cli"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		os.Stderr.WriteString("Error: could not determine working directory\n")
		os.Exit(1)
	}

	app := cli.NewApp(os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args, cwd))
}
