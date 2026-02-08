package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// runInit implements the `tick init` command.
// It creates the .tick/ directory and an empty tasks.jsonl file.
func (a *App) runInit(args []string) error {
	absDir, err := filepath.Abs(a.Dir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	tickDir := filepath.Join(absDir, ".tick")

	// Check if already initialized
	if _, err := os.Stat(tickDir); err == nil {
		return fmt.Errorf("Tick already initialized in this directory")
	}

	// Create .tick/ directory
	if err := os.MkdirAll(tickDir, 0755); err != nil {
		return fmt.Errorf("failed to create .tick/ directory: %w", err)
	}

	// Create empty tasks.jsonl
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		// Clean up on failure
		os.RemoveAll(tickDir)
		return fmt.Errorf("failed to create tasks.jsonl: %w", err)
	}

	// Output via formatter
	if a.Quiet {
		return nil
	}
	msg := fmt.Sprintf("Initialized tick in %s/", tickDir)
	return a.Formatter.FormatMessage(a.Stdout, msg)
}
