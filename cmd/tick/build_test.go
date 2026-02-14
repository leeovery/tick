package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/testutil"
)

func TestBuild(t *testing.T) {
	// Build once for all subtests.
	repoRoot := testutil.FindRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "tick")

	cmd := exec.Command("go", "build", "-o", binary, "./cmd/tick/")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	t.Run("go build produces a tick binary without errors", func(t *testing.T) {
		info, err := os.Stat(binary)
		if err != nil {
			t.Fatalf("binary not found: %v", err)
		}
		if info.Size() == 0 {
			t.Fatal("binary is empty")
		}
		// Verify it is executable.
		if info.Mode()&0111 == 0 {
			t.Fatal("binary is not executable")
		}
	})

	t.Run("tick binary outputs version string to stdout", func(t *testing.T) {
		cmd := exec.Command(binary)
		out, _ := cmd.Output()
		stdout := string(out)
		if !strings.Contains(strings.ToLower(stdout), "tick") {
			t.Errorf("expected stdout to contain 'tick', got: %q", stdout)
		}
	})

	t.Run("tick binary exits with code 0", func(t *testing.T) {
		cmd := exec.Command(binary)
		err := cmd.Run()
		if err != nil {
			t.Errorf("expected exit code 0, got error: %v", err)
		}
	})
}
