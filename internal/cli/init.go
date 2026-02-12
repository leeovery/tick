// Package cli implements the tick CLI framework including command routing, flags, and handlers.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RunInit initializes a new tick project in the given directory.
// It creates the .tick/ directory and an empty tasks.jsonl file.
// If quiet, no output is produced on success. Otherwise, a message is formatted via the Formatter.
func RunInit(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("could not resolve absolute path: %w", err)
	}

	tickDir := filepath.Join(absDir, ".tick")

	if _, err := os.Stat(tickDir); err == nil {
		return fmt.Errorf("tick already initialized in %s", absDir)
	}

	if err := os.Mkdir(tickDir, 0755); err != nil {
		return fmt.Errorf("could not create .tick/ directory: %w", err)
	}

	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("could not create tasks.jsonl: %w", err)
	}

	if !fc.Quiet {
		msg := fmt.Sprintf("Initialized tick in %s/.tick/", absDir)
		fmt.Fprintln(stdout, fmtr.FormatMessage(msg))
	}

	return nil
}
