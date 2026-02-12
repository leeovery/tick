package cli

import (
	"errors"
	"os"
	"path/filepath"
)

// DiscoverTickDir walks up from the given directory looking for a .tick/ directory.
// Returns the absolute path to the .tick/ directory, or an error if not found.
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
			// Reached filesystem root without finding .tick/
			return "", errors.New("not a tick project (no .tick directory found)")
		}
		dir = parent
	}
}
