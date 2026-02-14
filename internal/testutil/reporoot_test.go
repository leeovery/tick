package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leeovery/tick/internal/testutil"
)

func TestFindRepoRoot(t *testing.T) {
	t.Run("returns a path containing go.mod", func(t *testing.T) {
		root := testutil.FindRepoRoot(t)
		goModPath := filepath.Join(root, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			t.Fatalf("expected go.mod at %s, got error: %v", goModPath, err)
		}
	})

	t.Run("returns consistent result on repeated calls", func(t *testing.T) {
		root1 := testutil.FindRepoRoot(t)
		root2 := testutil.FindRepoRoot(t)
		if root1 != root2 {
			t.Errorf("expected consistent results, got %q and %q", root1, root2)
		}
	})
}
