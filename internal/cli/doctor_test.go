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

	_ "modernc.org/sqlite"
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
	db, err := sql.Open("sqlite", cachePath)
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

// setupDoctorProjectWithContent creates a .tick/ directory with the given tasks.jsonl content
// and a cache.db whose hash matches the content (fresh cache).
func setupDoctorProjectWithContent(t *testing.T, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick/: %v", err)
	}
	raw := []byte(content)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, raw, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	h := sha256.Sum256(raw)
	hash := hex.EncodeToString(h[:])
	createDoctorCache(t, tickDir, hash)
	return dir, tickDir
}

// setupDoctorProjectWithContentStale creates a .tick/ directory with the given tasks.jsonl
// content and a cache.db whose hash does NOT match (stale cache).
func setupDoctorProjectWithContentStale(t *testing.T, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	tickDir := filepath.Join(dir, ".tick")
	if err := os.Mkdir(tickDir, 0755); err != nil {
		t.Fatalf("failed to create .tick/: %v", err)
	}
	raw := []byte(content)
	jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
	if err := os.WriteFile(jsonlPath, raw, 0644); err != nil {
		t.Fatalf("failed to create tasks.jsonl: %v", err)
	}
	createDoctorCache(t, tickDir, "stale-hash-value")
	return dir, tickDir
}

func TestDoctorFourChecks(t *testing.T) {
	t.Run("it registers all four checks (cache staleness, JSONL syntax, ID format, duplicate ID)", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, _, _ := runDoctor(t, dir)

		for _, label := range []string{"Cache", "JSONL syntax", "ID format", "ID uniqueness"} {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain check label %q, got %q", label, stdout)
			}
		}
	})

	t.Run("it runs all four checks in a single tick doctor invocation", func(t *testing.T) {
		dir, _ := setupDoctorProject(t)

		stdout, _, _ := runDoctor(t, dir)

		// Count check marks — should have 10 passing checks (4 original + 6 relationship/hierarchy).
		checkCount := strings.Count(stdout, "\u2713")
		if checkCount != 10 {
			t.Errorf("expected 10 check marks, got %d; stdout = %q", checkCount, stdout)
		}
	})

	t.Run("it exits 0 when all four checks pass", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, `{"id":"tick-aaa111","title":"Test"}`)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
	})

	t.Run("it exits 1 when only the JSONL syntax check fails (other three pass)", func(t *testing.T) {
		// Invalid JSON but valid cache hash (cache sees the raw bytes and hashes them).
		dir, _ := setupDoctorProjectWithContent(t, "not valid json")

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the ID format check fails (other three pass)", func(t *testing.T) {
		// Valid JSON, valid cache hash, but invalid ID format.
		dir, _ := setupDoctorProjectWithContent(t, `{"id":"bad-id","title":"Test"}`)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the duplicate ID check fails (other three pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test1"}
{"id":"tick-aaa111","title":"Test2"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when all three new checks fail but cache check passes", func(t *testing.T) {
		// Line 1: invalid JSON (syntax fails, id format skips, dup skips).
		// Line 2: valid JSON, bad ID (id format fails).
		// Line 3 & 4: valid JSON, duplicate IDs (dup check fails).
		content := `not valid json
{"id":"bad-id","title":"Test"}
{"id":"tick-bbb222","title":"Dup1"}
{"id":"tick-bbb222","title":"Dup2"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when all four checks fail", func(t *testing.T) {
		// Invalid JSON + bad ID + duplicates + stale cache.
		content := `not valid json
{"id":"bad-id","title":"Test"}
{"id":"tick-bbb222","title":"Dup1"}
{"id":"tick-bbb222","title":"Dup2"}`
		dir, _ := setupDoctorProjectWithContentStale(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when cache check fails but all three new checks pass", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContentStale(t, `{"id":"tick-aaa111","title":"Test"}`)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it reports mixed results correctly — passing checks show checkmark, failing checks show details", func(t *testing.T) {
		// Only ID format fails: valid JSON, fresh cache, no duplicates, but bad ID.
		dir, _ := setupDoctorProjectWithContent(t, `{"id":"bad-id","title":"Test"}`)

		stdout, _, _ := runDoctor(t, dir)

		// Cache, JSONL syntax, and ID uniqueness should pass with checkmarks.
		if !strings.Contains(stdout, "\u2713 Cache: OK") {
			t.Errorf("stdout should contain passing Cache check, got %q", stdout)
		}
		if !strings.Contains(stdout, "\u2713 JSONL syntax: OK") {
			t.Errorf("stdout should contain passing JSONL syntax check, got %q", stdout)
		}
		if !strings.Contains(stdout, "\u2713 ID uniqueness: OK") {
			t.Errorf("stdout should contain passing ID uniqueness check, got %q", stdout)
		}
		// ID format should fail with cross mark and details.
		if !strings.Contains(stdout, "\u2717 ID format:") {
			t.Errorf("stdout should contain failing ID format check, got %q", stdout)
		}
	})

	t.Run("it displays results for all four checks in output (four check labels visible)", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, `{"id":"tick-aaa111","title":"Test"}`)

		stdout, _, _ := runDoctor(t, dir)

		labels := []string{"Cache", "JSONL syntax", "ID format", "ID uniqueness"}
		for _, label := range labels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q, got %q", label, stdout)
			}
		}
	})

	t.Run("it shows correct summary count reflecting errors from all checks combined", func(t *testing.T) {
		// Line 1: invalid JSON (1 syntax error) + bad ID skipped by id check + skipped by dup.
		// Line 2: valid JSON, bad ID (1 id format error).
		// Line 3 & 4: valid JSON, duplicate IDs (1 dup error).
		// Cache: fresh.
		// Total: 3 errors (syntax=1, id=1, dup=1).
		content := `not valid json
{"id":"bad-id","title":"Test"}
{"id":"tick-bbb222","title":"Dup1"}
{"id":"tick-bbb222","title":"Dup2"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "3 issues found.") {
			t.Errorf("stdout should contain '3 issues found.', got %q", stdout)
		}
	})

	t.Run("it runs all four checks even when the first check fails (no short-circuit)", func(t *testing.T) {
		// Stale cache (first check fails), but the other three checks should still run.
		dir, _ := setupDoctorProjectWithContentStale(t, `{"id":"tick-aaa111","title":"Test"}`)

		stdout, _, _ := runDoctor(t, dir)

		// All four check labels should be present in output.
		labels := []string{"Cache", "JSONL syntax", "ID format", "ID uniqueness"}
		for _, label := range labels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q even when earlier check fails, got %q", label, stdout)
			}
		}
	})

	t.Run("it handles empty tasks.jsonl — all checks report their respective results", func(t *testing.T) {
		dir, _ := setupDoctorProject(t) // Empty tasks.jsonl, fresh cache.

		stdout, _, exitCode := runDoctor(t, dir)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		// All four checks should produce results.
		labels := []string{"Cache", "JSONL syntax", "ID format", "ID uniqueness"}
		for _, label := range labels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q for empty file, got %q", label, stdout)
			}
		}
	})

	t.Run("it does not modify tasks.jsonl or cache.db (read-only invariant preserved with four checks)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Test1"}
{"id":"tick-bbb222","title":"Test2"}`
		dir, tickDir := setupDoctorProjectWithContent(t, content)

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		cachePath := filepath.Join(tickDir, "cache.db")

		jsonlBefore, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}
		cacheBefore, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db before: %v", err)
		}

		runDoctor(t, dir)

		jsonlAfter, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		cacheAfter, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db after: %v", err)
		}

		if string(jsonlBefore) != string(jsonlAfter) {
			t.Error("tasks.jsonl was modified by doctor with four checks")
		}
		if string(cacheBefore) != string(cacheAfter) {
			t.Error("cache.db was modified by doctor with four checks")
		}
	})
}

// healthyTenCheckContent returns tasks.jsonl content that passes all 10 checks:
// valid IDs, no duplicates, valid JSON, no orphaned parents/deps, no self-refs,
// no cycles, no child-blocked-by-parent, and no done parent with open children.
func healthyTenCheckContent() string {
	return `{"id":"tick-aaa111","title":"Parent","status":"open"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}
{"id":"tick-ccc333","title":"Independent","status":"done"}`
}

func TestDoctorTenChecks(t *testing.T) {
	allLabels := []string{
		"Cache", "JSONL syntax", "ID format", "ID uniqueness",
		"Orphaned parents", "Orphaned dependencies",
		"Self-referential dependencies", "Dependency cycles",
		"Child blocked by parent", "Parent done with open children",
	}

	t.Run("it registers all 10 checks (4 existing + 6 new relationship/hierarchy checks)", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		stdout, _, _ := runDoctor(t, dir)

		for _, label := range allLabels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain check label %q, got %q", label, stdout)
			}
		}
	})

	t.Run("it runs all 10 checks in a single tick doctor invocation", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		stdout, _, _ := runDoctor(t, dir)

		checkCount := strings.Count(stdout, "\u2713")
		if checkCount != 10 {
			t.Errorf("expected 10 check marks, got %d; stdout = %q", checkCount, stdout)
		}
	})

	t.Run("it exits 0 when all 10 checks pass (healthy store)", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
	})

	t.Run("it exits 1 when only the orphaned parent check fails (other 9 pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the orphaned dependency check fails (other 9 pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task","status":"open","blocked_by":["tick-ffffff"]}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the self-referential dependency check fails (other 9 pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task","status":"open","blocked_by":["tick-aaa111"]}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the dependency cycle check fails (other 9 pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task A","status":"open","blocked_by":["tick-bbb222"]}
{"id":"tick-bbb222","title":"Task B","status":"open","blocked_by":["tick-aaa111"]}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when only the child-blocked-by-parent check fails (other 9 pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Parent","status":"open"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111","blocked_by":["tick-aaa111"]}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 0 when only the parent-done-with-open-children warning fires (all 9 error checks pass)", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Parent","status":"done"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		stdout, _, exitCode := runDoctor(t, dir)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stdout = %q", exitCode, stdout)
		}
	})

	t.Run("it exits 1 when both error checks and warning check produce failures", func(t *testing.T) {
		// Orphaned parent (error) + done parent with open child (warning).
		content := `{"id":"tick-aaa111","title":"Parent","status":"done"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}
{"id":"tick-ccc333","title":"Orphan","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when a Phase 1 error (cache stale) and a Phase 3 error (orphaned parent) both fire", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContentStale(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it exits 1 when a Phase 2 error (duplicate ID) and a Phase 3 error (dependency cycle) both fire", func(t *testing.T) {
		content := `{"id":"tick-aaa111","title":"Task A","status":"open","blocked_by":["tick-bbb222"]}
{"id":"tick-bbb222","title":"Task B","status":"open","blocked_by":["tick-aaa111"]}
{"id":"tick-aaa111","title":"Dup","status":"open"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		_, _, exitCode := runDoctor(t, dir)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it reports mixed results correctly - passing checks show checkmark, failing error checks show cross with details, warning check shows cross with details", func(t *testing.T) {
		// Orphaned parent (error) + done parent with open child (warning).
		// All other checks pass.
		content := `{"id":"tick-aaa111","title":"Parent","status":"done"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}
{"id":"tick-ccc333","title":"Orphan","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		stdout, _, _ := runDoctor(t, dir)

		// Passing checks should show checkmarks.
		if !strings.Contains(stdout, "\u2713 Cache: OK") {
			t.Errorf("stdout should contain passing Cache check, got %q", stdout)
		}
		if !strings.Contains(stdout, "\u2713 JSONL syntax: OK") {
			t.Errorf("stdout should contain passing JSONL syntax check, got %q", stdout)
		}
		// Orphaned parent should fail with cross.
		if !strings.Contains(stdout, "\u2717 Orphaned parents:") {
			t.Errorf("stdout should contain failing Orphaned parents check, got %q", stdout)
		}
		// Warning check should fail with cross.
		if !strings.Contains(stdout, "\u2717 Parent done with open children:") {
			t.Errorf("stdout should contain failing warning check, got %q", stdout)
		}
	})

	t.Run("it displays results for all 10 checks in output (10 check labels visible when all pass)", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		stdout, _, _ := runDoctor(t, dir)

		for _, label := range allLabels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q, got %q", label, stdout)
			}
		}
	})

	t.Run("it shows correct summary count reflecting errors and warnings from all checks combined", func(t *testing.T) {
		// Orphaned parent (1 error) + done parent with open child (1 warning) = 2 issues.
		content := `{"id":"tick-aaa111","title":"Parent","status":"done"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}
{"id":"tick-ccc333","title":"Orphan","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "2 issues found.") {
			t.Errorf("stdout should contain '2 issues found.', got %q", stdout)
		}
	})

	t.Run("it shows summary count including warnings (e.g., 1 error + 1 warning = '2 issues found.')", func(t *testing.T) {
		// Same scenario: 1 error + 1 warning.
		content := `{"id":"tick-aaa111","title":"Parent","status":"done"}
{"id":"tick-bbb222","title":"Child","status":"open","parent":"tick-aaa111"}
{"id":"tick-ccc333","title":"Orphan","status":"open","parent":"tick-ffffff"}`
		dir, _ := setupDoctorProjectWithContent(t, content)

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "2 issues found.") {
			t.Errorf("stdout should contain '2 issues found.', got %q", stdout)
		}
	})

	t.Run("it runs all 10 checks even when early checks fail (no short-circuit)", func(t *testing.T) {
		// Stale cache (first check fails), but all 10 should still run.
		dir, _ := setupDoctorProjectWithContentStale(t, healthyTenCheckContent())

		stdout, _, _ := runDoctor(t, dir)

		for _, label := range allLabels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q even when earlier check fails, got %q", label, stdout)
			}
		}
	})

	t.Run("it handles empty tasks.jsonl - all 10 checks report their respective passing/failing results", func(t *testing.T) {
		dir, _ := setupDoctorProject(t) // Empty tasks.jsonl, fresh cache.

		stdout, _, exitCode := runDoctor(t, dir)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0", exitCode)
		}
		for _, label := range allLabels {
			if !strings.Contains(stdout, label) {
				t.Errorf("stdout should contain %q for empty file, got %q", label, stdout)
			}
		}
	})

	t.Run("it does not modify tasks.jsonl or cache.db (read-only invariant preserved with 10 checks)", func(t *testing.T) {
		dir, tickDir := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		jsonlPath := filepath.Join(tickDir, "tasks.jsonl")
		cachePath := filepath.Join(tickDir, "cache.db")

		jsonlBefore, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl before: %v", err)
		}
		cacheBefore, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db before: %v", err)
		}

		runDoctor(t, dir)

		jsonlAfter, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl after: %v", err)
		}
		cacheAfter, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("failed to read cache.db after: %v", err)
		}

		if string(jsonlBefore) != string(jsonlAfter) {
			t.Error("tasks.jsonl was modified by doctor with 10 checks")
		}
		if string(cacheBefore) != string(cacheAfter) {
			t.Error("cache.db was modified by doctor with 10 checks")
		}
	})

	t.Run("it shows 'No issues found.' summary when all 10 checks pass", func(t *testing.T) {
		dir, _ := setupDoctorProjectWithContent(t, healthyTenCheckContent())

		stdout, _, _ := runDoctor(t, dir)

		if !strings.Contains(stdout, "No issues found.") {
			t.Errorf("stdout should contain 'No issues found.', got %q", stdout)
		}
	})
}
