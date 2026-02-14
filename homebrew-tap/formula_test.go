package homebrew_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findRepoRoot walks up from the test file to find go.mod.
func findRepoRoot(t *testing.T) string {
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

// loadFormula reads the tick.rb formula file and returns its content.
func loadFormula(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	path := filepath.Join(root, "homebrew-tap", "Formula", "tick.rb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read tick.rb: %v", err)
	}
	return string(data)
}

// loadREADME reads the homebrew-tap README.md and returns its content.
func loadREADME(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	path := filepath.Join(root, "homebrew-tap", "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read README.md: %v", err)
	}
	return string(data)
}

func TestFormula(t *testing.T) {
	t.Run("formula file exists at homebrew-tap/Formula/tick.rb", func(t *testing.T) {
		root := findRepoRoot(t)
		path := filepath.Join(root, "homebrew-tap", "Formula", "tick.rb")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("tick.rb not found: %v", err)
		}
		if info.IsDir() {
			t.Fatal("tick.rb is a directory, expected a file")
		}
	})

	t.Run("formula class name is Tick", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, "class Tick < Formula") {
			t.Error("formula must contain 'class Tick < Formula'")
		}
	})

	t.Run("formula URL for Apple Silicon uses darwin_arm64 asset", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, "darwin_arm64.tar.gz") {
			t.Error("formula must contain a URL referencing darwin_arm64.tar.gz")
		}
	})

	t.Run("formula URL for Intel uses darwin_amd64 asset", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, "darwin_amd64.tar.gz") {
			t.Error("formula must contain a URL referencing darwin_amd64.tar.gz")
		}
	})

	t.Run("formula URL tag path includes v prefix but filename does not", func(t *testing.T) {
		content := loadFormula(t)
		// URL should have /download/v#{version}/ for the tag path
		if !strings.Contains(content, `/download/v#{version}/`) {
			t.Error("formula URL tag path must include v prefix: /download/v#{version}/")
		}
		// Filename should use tick_#{version}_ (no v prefix)
		if !strings.Contains(content, `tick_#{version}_`) {
			t.Error("formula URL filename must use tick_#{version}_ without v prefix")
		}
	})

	t.Run("formula handles both Intel and Apple Silicon via on_macos block", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, "Hardware::CPU.arm?") {
			t.Error("formula must contain 'Hardware::CPU.arm?'")
		}
		if !strings.Contains(content, "Hardware::CPU.intel?") {
			t.Error("formula must contain 'Hardware::CPU.intel?'")
		}
	})

	t.Run("formula installs tick binary to bin", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, `bin.install "tick"`) {
			t.Error("formula must contain 'bin.install \"tick\"'")
		}
	})

	t.Run("formula includes a test block", func(t *testing.T) {
		content := loadFormula(t)
		if !strings.Contains(content, "test do") {
			t.Error("formula must contain 'test do'")
		}
	})

	t.Run("formula includes sha256 for each architecture", func(t *testing.T) {
		content := loadFormula(t)
		count := strings.Count(content, "sha256")
		if count < 2 {
			t.Errorf("formula must have at least 2 sha256 declarations, found %d", count)
		}
	})

	t.Run("README includes tap and install commands", func(t *testing.T) {
		content := loadREADME(t)
		if !strings.Contains(content, "brew tap leeovery/tick") {
			t.Error("README must contain 'brew tap leeovery/tick'")
		}
		if !strings.Contains(content, "brew install tick") {
			t.Error("README must contain 'brew install tick'")
		}
	})
}
