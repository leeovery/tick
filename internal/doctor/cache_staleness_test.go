package doctor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTickDir creates a temp .tick directory and returns its path.
func setupTickDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick: %v", err)
	}
	return tickDir
}

// writeJSONL writes the given content to tasks.jsonl in the tick directory.
func writeJSONL(t *testing.T, tickDir string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), content, 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
}

// createCacheWithHash creates a cache.db with a metadata table containing the given hash.
func createCacheWithHash(t *testing.T, tickDir string, hash string) {
	t.Helper()
	cachePath := filepath.Join(tickDir, "cache.db")
	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("failed to create metadata table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO metadata (key, value) VALUES ('jsonl_hash', ?)`, hash); err != nil {
		t.Fatalf("failed to insert hash: %v", err)
	}
}

// createCacheWithoutHash creates a cache.db with an empty metadata table (no jsonl_hash key).
func createCacheWithoutHash(t *testing.T, tickDir string) {
	t.Helper()
	cachePath := filepath.Join(tickDir, "cache.db")
	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("failed to create metadata table: %v", err)
	}
}

// computeTestHash returns the hex-encoded SHA256 hash of the given data.
func computeTestHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ctxWithTickDir returns a background context. Retained for test compatibility.
// tickDir is now passed as an explicit parameter to Run.
func ctxWithTickDir(_ string) context.Context {
	return context.Background()
}

func TestCacheStalenessCheck(t *testing.T) {
	t.Run("it returns passing result when tasks.jsonl and cache.db hashes match", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"abc","title":"Test"}`)
		writeJSONL(t, tickDir, content)
		createCacheWithHash(t, tickDir, computeTestHash(content))

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true, got false; details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result with stale details when hashes do not match", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"abc","title":"Test"}`)
		writeJSONL(t, tickDir, content)
		createCacheWithHash(t, tickDir, "stale-hash-value")

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for stale cache")
		}
		if results[0].Details != "cache.db is stale — hash mismatch between tasks.jsonl and cache" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when cache.db does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when cache.db missing")
		}
		if results[0].Details != "cache.db not found — cache has not been built" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns failing result when tasks.jsonl does not exist", func(t *testing.T) {
		tickDir := setupTickDir(t)
		// No tasks.jsonl created

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when tasks.jsonl missing")
		}
		if results[0].Suggestion != "Run tick init or verify .tick directory" {
			t.Errorf("unexpected Suggestion: %s", results[0].Suggestion)
		}
	})

	t.Run("it returns failing result when metadata table has no jsonl_hash key", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
		createCacheWithoutHash(t, tickDir)

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when jsonl_hash key missing")
		}
		if results[0].Details != "cache.db is stale — hash mismatch between tasks.jsonl and cache" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})

	t.Run("it returns passing result for empty tasks.jsonl with matching hash", func(t *testing.T) {
		tickDir := setupTickDir(t)
		emptyContent := []byte{}
		writeJSONL(t, tickDir, emptyContent)
		// SHA256 of empty content: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
		createCacheWithHash(t, tickDir, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Passed {
			t.Errorf("expected Passed true for empty tasks.jsonl with matching hash; details: %s", results[0].Details)
		}
	})

	t.Run("it suggests Run tick rebuild to refresh cache when cache is stale", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
		createCacheWithHash(t, tickDir, "wrong-hash")

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Run `tick rebuild` to refresh cache"
		if results[0].Suggestion != expected {
			t.Errorf("expected Suggestion %q, got %q", expected, results[0].Suggestion)
		}
	})

	t.Run("it suggests Run tick rebuild to refresh cache when cache.db is missing", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		expected := "Run `tick rebuild` to refresh cache"
		if results[0].Suggestion != expected {
			t.Errorf("expected Suggestion %q, got %q", expected, results[0].Suggestion)
		}
	})

	t.Run("it uses CheckResult Name Cache for all results", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := []byte(`{"id":"abc"}`)
					writeJSONL(t, tickDir, content)
					createCacheWithHash(t, tickDir, computeTestHash(content))
					return tickDir
				},
			},
			{
				name: "stale hash",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					createCacheWithHash(t, tickDir, "wrong")
					return tickDir
				},
			},
			{
				name: "missing cache.db",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					return tickDir
				},
			},
			{
				name: "missing tasks.jsonl",
				setup: func(t *testing.T) string {
					return setupTickDir(t)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := tc.setup(t)
				check := &CacheStalenessCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Name != "Cache" {
					t.Errorf("expected Name %q, got %q", "Cache", results[0].Name)
				}
			})
		}
	})

	t.Run("it uses SeverityError for all failure cases", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "stale hash",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					createCacheWithHash(t, tickDir, "wrong")
					return tickDir
				},
			},
			{
				name: "missing cache.db",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					return tickDir
				},
			},
			{
				name: "missing tasks.jsonl",
				setup: func(t *testing.T) string {
					return setupTickDir(t)
				},
			},
			{
				name: "missing jsonl_hash key",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					createCacheWithoutHash(t, tickDir)
					return tickDir
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := tc.setup(t)
				check := &CacheStalenessCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Passed {
					t.Error("expected Passed false for failure case")
				}
				if results[0].Severity != SeverityError {
					t.Errorf("expected Severity %q, got %q", SeverityError, results[0].Severity)
				}
			})
		}
	})

	t.Run("it does not modify tasks.jsonl or cache.db (read-only verification)", func(t *testing.T) {
		tickDir := setupTickDir(t)
		content := []byte(`{"id":"abc","title":"Test"}`)
		writeJSONL(t, tickDir, content)
		createCacheWithHash(t, tickDir, computeTestHash(content))

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		cachePath := filepath.Join(tickDir, "cache.db")

		jsonlBefore, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before check: %v", err)
		}
		cacheBefore, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db before check: %v", err)
		}

		check := &CacheStalenessCheck{}
		check.Run(ctxWithTickDir(tickDir), tickDir)

		jsonlAfter, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after check: %v", err)
		}
		cacheAfter, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db after check: %v", err)
		}

		if string(jsonlBefore) != string(jsonlAfter) {
			t.Error("tasks.jsonl was modified by the check")
		}
		if string(cacheBefore) != string(cacheAfter) {
			t.Error("cache.db was modified by the check")
		}
	})

	t.Run("it returns exactly one CheckResult (single result, not multiple)", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(t *testing.T) string
		}{
			{
				name: "passing",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					content := []byte(`{"id":"abc"}`)
					writeJSONL(t, tickDir, content)
					createCacheWithHash(t, tickDir, computeTestHash(content))
					return tickDir
				},
			},
			{
				name: "stale",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					createCacheWithHash(t, tickDir, "wrong")
					return tickDir
				},
			},
			{
				name: "missing cache",
				setup: func(t *testing.T) string {
					tickDir := setupTickDir(t)
					writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
					return tickDir
				},
			},
			{
				name: "missing jsonl",
				setup: func(t *testing.T) string {
					return setupTickDir(t)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tickDir := tc.setup(t)
				check := &CacheStalenessCheck{}
				results := check.Run(ctxWithTickDir(tickDir), tickDir)

				if len(results) != 1 {
					t.Errorf("expected exactly 1 result, got %d", len(results))
				}
			})
		}
	})

	t.Run("it handles corrupted cache.db without panicking", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
		// Write invalid data as cache.db
		cachePath := filepath.Join(tickDir, "cache.db")
		if err := os.WriteFile(cachePath, []byte("not a sqlite database"), 0644); err != nil {
			t.Fatalf("failed to write corrupted cache.db: %v", err)
		}

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false for corrupted cache.db")
		}
		if results[0].Severity != SeverityError {
			t.Errorf("expected SeverityError, got %q", results[0].Severity)
		}
	})

	t.Run("it returns failing result when tick directory is empty string", func(t *testing.T) {
		check := &CacheStalenessCheck{}
		results := check.Run(context.Background(), "")

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when tick directory is empty")
		}
		if results[0].Severity != SeverityError {
			t.Errorf("expected SeverityError, got %q", results[0].Severity)
		}
		if results[0].Name != "Cache" {
			t.Errorf("expected Name %q, got %q", "Cache", results[0].Name)
		}
	})

	t.Run("it handles cache.db with no metadata table", func(t *testing.T) {
		tickDir := setupTickDir(t)
		writeJSONL(t, tickDir, []byte(`{"id":"abc"}`))
		// Create a valid SQLite file but without metadata table
		cachePath := filepath.Join(tickDir, "cache.db")
		db, err := sql.Open("sqlite3", cachePath)
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		if _, err := db.Exec(`CREATE TABLE tasks (id TEXT)`); err != nil {
			t.Fatalf("failed to create tasks table: %v", err)
		}
		db.Close()

		check := &CacheStalenessCheck{}
		results := check.Run(ctxWithTickDir(tickDir), tickDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("expected Passed false when metadata table missing")
		}
		if results[0].Details != "cache.db is stale — hash mismatch between tasks.jsonl and cache" {
			t.Errorf("unexpected Details: %s", results[0].Details)
		}
	})
}
