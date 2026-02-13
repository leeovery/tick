package doctor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// CacheStalenessCheck verifies that the SQLite cache (cache.db) is in sync
// with the JSONL source of truth (tasks.jsonl) by comparing SHA256 content hashes.
// It implements the Check interface and is read-only — it never modifies any files.
type CacheStalenessCheck struct{}

// Run executes the cache staleness check. It computes the SHA256 hash of
// tasks.jsonl and compares it to the hash stored in cache.db's metadata table.
func (c *CacheStalenessCheck) Run(_ context.Context, tickDir string) []CheckResult {
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	cachePath := filepath.Join(tickDir, "cache.db")

	// Step 1: Read tasks.jsonl.
	rawJSONL, err := os.ReadFile(jsonlPath)
	if err != nil {
		return []CheckResult{{
			Name:       "Cache",
			Passed:     false,
			Severity:   SeverityError,
			Details:    fmt.Sprintf("tasks.jsonl not found or unreadable: %v", err),
			Suggestion: "Run tick init or verify .tick directory",
		}}
	}

	// Step 2: Compute SHA256 hash of raw JSONL content.
	h := sha256.Sum256(rawJSONL)
	jsonlHash := hex.EncodeToString(h[:])

	// Step 3: Check if cache.db exists.
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return []CheckResult{{
			Name:       "Cache",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "cache.db not found — cache has not been built",
			Suggestion: "Run `tick rebuild` to refresh cache",
		}}
	}

	// Step 4: Open cache.db in read-only mode and query the stored hash.
	storedHash, err := queryStoredHash(cachePath)
	if err != nil {
		// Corrupted, missing table, missing key — treat as stale.
		return []CheckResult{{
			Name:       "Cache",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "cache.db is stale — hash mismatch between tasks.jsonl and cache",
			Suggestion: "Run `tick rebuild` to refresh cache",
		}}
	}

	// Step 5: Compare hashes.
	if jsonlHash != storedHash {
		return []CheckResult{{
			Name:       "Cache",
			Passed:     false,
			Severity:   SeverityError,
			Details:    "cache.db is stale — hash mismatch between tasks.jsonl and cache",
			Suggestion: "Run `tick rebuild` to refresh cache",
		}}
	}

	return []CheckResult{{
		Name:   "Cache",
		Passed: true,
	}}
}

// queryStoredHash opens cache.db in read-only mode, queries the metadata table
// for the jsonl_hash key, and returns the stored value. It returns an error if
// the database cannot be opened, the metadata table does not exist, or the key
// is missing.
func queryStoredHash(cachePath string) (string, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", cachePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return "", fmt.Errorf("failed to open cache.db: %w", err)
	}
	defer db.Close()

	var storedHash string
	err = db.QueryRow("SELECT value FROM metadata WHERE key = 'jsonl_hash'").Scan(&storedHash)
	if err != nil {
		return "", fmt.Errorf("failed to query jsonl_hash: %w", err)
	}

	return storedHash, nil
}
