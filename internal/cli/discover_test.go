package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverTickDir(t *testing.T) {
	t.Run("it discovers .tick/ directory by walking up from cwd", func(t *testing.T) {
		// Create a temp directory with .tick/ at the root
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		// Create a nested subdirectory
		nested := filepath.Join(root, "sub", "deep")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatalf("failed to create nested dirs: %v", err)
		}

		// Discover from the nested directory should find .tick/ at root
		got, err := DiscoverTickDir(nested)
		if err != nil {
			t.Fatalf("DiscoverTickDir() returned error: %v", err)
		}

		want := tickDir
		if got != want {
			t.Errorf("DiscoverTickDir() = %q, want %q", got, want)
		}
	})

	t.Run("it finds .tick/ in the starting directory itself", func(t *testing.T) {
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		got, err := DiscoverTickDir(root)
		if err != nil {
			t.Fatalf("DiscoverTickDir() returned error: %v", err)
		}

		if got != tickDir {
			t.Errorf("DiscoverTickDir() = %q, want %q", got, tickDir)
		}
	})

	t.Run("it errors when no .tick/ directory found (not a tick project)", func(t *testing.T) {
		// Use a temp directory with no .tick/ anywhere in its ancestry
		dir := t.TempDir()

		_, err := DiscoverTickDir(dir)
		if err == nil {
			t.Fatal("DiscoverTickDir() expected error, got nil")
		}

		want := "Not a tick project (no .tick directory found)"
		if err.Error() != want {
			t.Errorf("DiscoverTickDir() error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it stops at the first .tick/ match walking up", func(t *testing.T) {
		// Create two .tick/ directories at different levels
		root := t.TempDir()
		rootTick := filepath.Join(root, ".tick")
		if err := os.Mkdir(rootTick, 0755); err != nil {
			t.Fatalf("failed to create root .tick: %v", err)
		}

		sub := filepath.Join(root, "sub")
		if err := os.Mkdir(sub, 0755); err != nil {
			t.Fatalf("failed to create sub: %v", err)
		}
		subTick := filepath.Join(sub, ".tick")
		if err := os.Mkdir(subTick, 0755); err != nil {
			t.Fatalf("failed to create sub .tick: %v", err)
		}

		// From sub, should find the closer .tick/
		got, err := DiscoverTickDir(sub)
		if err != nil {
			t.Fatalf("DiscoverTickDir() returned error: %v", err)
		}

		if got != subTick {
			t.Errorf("DiscoverTickDir() = %q, want %q (closest match)", got, subTick)
		}
	})
}
