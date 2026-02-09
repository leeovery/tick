package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// runInit implements the "tick init" command. It creates the .tick/ directory
// and an empty tasks.jsonl file in the current working directory.
func runInit(ctx *Context) error {
	tickDir := filepath.Join(ctx.WorkDir, ".tick")

	// Check if .tick/ already exists (even corrupted)
	if _, err := os.Stat(tickDir); err == nil {
		return fmt.Errorf("Tick already initialized in this directory")
	}

	// Create .tick/ directory
	if err := os.Mkdir(tickDir, 0755); err != nil {
		return fmt.Errorf("creating .tick directory: %w", err)
	}

	// Create empty tasks.jsonl
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("creating tasks.jsonl: %w", err)
	}

	// Print confirmation (unless --quiet)
	if !ctx.Quiet {
		absTickDir, err := filepath.Abs(tickDir)
		if err != nil {
			return fmt.Errorf("resolving absolute path: %w", err)
		}
		fmt.Fprintf(ctx.Stdout, "Initialized tick in %s/\n", absTickDir)
	}

	return nil
}
