package cli

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupDoctorProject creates a .tick/ directory with an empty tasks.jsonl
// and a cache.db whose hash matches the empty content.
func setupDoctorProject(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick/: %v", err)
	}
	content := []byte{}
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	h := sha256.Sum256(content)
	hash := hex.EncodeToString(h[:])
	createDoctorCache(t, tickDir, hash)
	return dir, tickDir
}

// setupDoctorProjectStale creates a .tick/ directory with tasks.jsonl
// and a cache.db whose hash does NOT match (stale cache).
func setupDoctorProjectStale(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick/: %v", err)
	}
	content := []byte(`{"id":"tick-aaa111","title":"Test"}`)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, content, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	createDoctorCache(t, tickDir, "stale-hash-value")
	return dir, tickDir
}

// createDoctorCache creates a cache.db with a metadata table containing the given hash.
func createDoctorCache(t *testing.T, tickDir string, hash string) {
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

// runDoctor runs the tick doctor command via the App dispatcher and returns stdout, stderr, exit code.
func runDoctor(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  false,
	}
	fullArgs := append([]string{"tick", "doctor"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestDoctor(t *testing.T) {
	t.Run("it exits 0 and prints all-pass output when data store is healthy", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, stderr, exitCode := runDoctor(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if stderr != "" {
			t.Errorf("stderr should be empty, got %q", stderr)
		}
		if !strings.Contains(stdout, "OK") {
			t.Errorf("stdout should contain 'OK', got %q", stdout)
		}
	})

	t.Run("it exits 1 and prints failure output when cache is stale", func(t *testing.T) {
		dir, _ := setupDoctorProjectStale(t)

		stdout, stderr, exitCode := runDoctor(t, dir)
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "stale") {
			t.Errorf("stdout should contain 'stale', got %q", stdout)
		}
		_ = stderr
	})

	t.Run("it prints formatted output to stdout with check markers and summary line", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "\u2713") {
			t.Errorf("stdout should contain check mark, got %q", stdout)
		}
		if !strings.Contains(stdout, "No issues found.") {
			t.Errorf("stdout should contain 'No issues found.', got %q", stdout)
		}
	})

	t.Run("it errors with exit code 1 when .tick directory is not found", func(t *testing.T) {
		dir := t.TempDir() // No .tick directory

		_, _, exitCode := runDoctor(t, dir)
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it prints Not a tick project error to stderr when .tick directory is missing", func(t *testing.T) {
		dir := t.TempDir() // No .tick directory

		stdout, stderr, _ := runDoctor(t, dir)

		if !strings.Contains(stderr, "Not a tick project") {
			t.Errorf("stderr should contain 'Not a tick project', got %q", stderr)
		}
		// Stdout should be empty — no diagnostic output.
		if stdout != "" {
			t.Errorf("stdout should be empty when .tick missing, got %q", stdout)
		}
	})

	t.Run("it does not modify tasks.jsonl (file unchanged after doctor runs)", func(t *testing.T) {
		dir, tickDir := setupDoctorProject(t)
		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")

		before, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before doctor: %v", err)
		}

		runDoctor(t, dir)

		after, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after doctor: %v", err)
		}
		if string(before) != string(after) {
			t.Error("tasks.jsonl was modified by doctor")
		}
	})

	t.Run("it does not modify cache.db (file unchanged after doctor runs)", func(t *testing.T) {
		dir, tickDir := setupDoctorProject(t)
		cachePath := filepath.Join(tickDir, "cache.db")

		before, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db before doctor: %v", err)
		}

		runDoctor(t, dir)

		after, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db after doctor: %v", err)
		}
		if string(before) != string(after) {
			t.Error("cache.db was modified by doctor")
		}
	})

	t.Run("it does not create any new files in .tick directory", func(t *testing.T) {
		dir, tickDir := setupDoctorProject(t)

		entriesBefore, err := os.ReadDir(tickDir)
		if err != nil {
			t.Fatalf("failed to read .tick dir before doctor: %v", err)
		}
		namesBefore := make(map[string]bool)
		for _, e := range entriesBefore {
			namesBefore[e.Name()] = true
		}

		runDoctor(t, dir)

		entriesAfter, err := os.ReadDir(tickDir)
		if err != nil {
			t.Fatalf("failed to read .tick dir after doctor: %v", err)
		}
		for _, e := range entriesAfter {
			if !namesBefore[e.Name()] {
				t.Errorf("doctor created new file in .tick/: %s", e.Name())
			}
		}
	})

	t.Run("it does not trigger a cache rebuild when cache is stale", func(t *testing.T) {
		dir, tickDir := setupDoctorProjectStale(t)
		cachePath := filepath.Join(tickDir, "cache.db")

		before, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db before doctor: %v", err)
		}

		runDoctor(t, dir)

		after, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db after doctor: %v", err)
		}
		if string(before) != string(after) {
			t.Error("cache.db was modified — doctor should not trigger rebuild")
		}
	})

	t.Run("it registers and runs the cache staleness check", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, _, exitCode := runDoctor(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout, "Cache") {
			t.Errorf("stdout should contain 'Cache' check name, got %q", stdout)
		}
	})

	t.Run("it shows No issues found summary when all checks pass", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "No issues found.") {
			t.Errorf("stdout should contain 'No issues found.', got %q", stdout)
		}
	})

	t.Run("it shows 1 issue found summary when cache staleness fails", func(t *testing.T) {
		dir, _ := setupDoctorProjectStale(t)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "1 issue found.") {
			t.Errorf("stdout should contain '1 issue found.', got %q", stdout)
		}
	})
}
