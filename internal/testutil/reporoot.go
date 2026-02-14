// Package testutil provides shared test helpers for the tick project.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// FindRepoRoot walks up from the current working directory to find
// the repository root (the directory containing go.mod).
func FindRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}
