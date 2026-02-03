package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// runInit implements the `tick init` command.
// It creates the .tick/ directory with an empty tasks.jsonl file.
func (a *App) runInit() error {
	absDir, err := filepath.Abs(a.workDir)
	if err != nil {
		return fmt.Errorf("Could not determine absolute path: %w", err)
	}

	tickDir := filepath.Join(absDir, ".tick")

	// Check if .tick/ already exists (even corrupted)
	if _, err := os.Stat(tickDir); err == nil {
		return fmt.Errorf("Tick already initialized in this directory")
	}

	// Create .tick/ directory
	if err := os.Mkdir(tickDir, 0755); err != nil {
		return fmt.Errorf("Could not create .tick/ directory: %w", err)
	}

	// Create empty tasks.jsonl
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("Could not create tasks.jsonl: %w", err)
	}

	// Print confirmation (unless quiet)
	if !a.config.Quiet {
		msg := fmt.Sprintf("Initialized tick in %s/", tickDir)
		return a.formatter.FormatMessage(a.stdout, msg)
	}

	return nil
}
