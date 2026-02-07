package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverTickDir(t *testing.T) {
	t.Run("it discovers .tick/ directory by walking up from cwd", func(t *testing.T) {
		// Create structure: tmpdir/.tick/ and tmpdir/sub/deep/
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		deepDir := filepath.Join(root, "sub", "deep")
		if err := os.MkdirAll(deepDir, 0755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		found, err := DiscoverTickDir(deepDir)
		if err != nil {
			t.Fatalf("expected to find .tick/, got error: %v", err)
		}

		expected := tickDir
		if found != expected {
			t.Errorf("expected %q, got %q", expected, found)
		}
	})

	t.Run("it finds .tick/ in current directory", func(t *testing.T) {
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		if err := os.MkdirAll(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick: %v", err)
		}

		found, err := DiscoverTickDir(root)
		if err != nil {
			t.Fatalf("expected to find .tick/, got error: %v", err)
		}

		if found != tickDir {
			t.Errorf("expected %q, got %q", tickDir, found)
		}
	})

	t.Run("it errors when no .tick/ directory found (not a tick project)", func(t *testing.T) {
		// Use a temp dir with no .tick/ anywhere up the tree
		dir := t.TempDir()

		_, err := DiscoverTickDir(dir)
		if err == nil {
			t.Fatal("expected error when no .tick/ found, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "Not a tick project") {
			t.Errorf("expected error to mention 'Not a tick project', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "no .tick directory found") {
			t.Errorf("expected error to mention 'no .tick directory found', got %q", errMsg)
		}
	})
}
