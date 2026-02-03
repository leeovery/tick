// Package cli implements the command-line interface for Tick.
package cli

import (
	"errors"
	"os"
	"path/filepath"
)

// DiscoverTickDir walks up from startDir looking for a .tick/ directory.
// It returns the absolute path to the first .tick/ found, or an error if none exists.
func DiscoverTickDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".tick")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", errors.New("Not a tick project (no .tick directory found)")
		}
		dir = parent
	}
}
