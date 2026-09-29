package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	sizedLinePrefix = `","title":"T","status":"open","priority":2,"description":"`
	sizedLineSuffix = `","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
)

// sizedTaskLine renders a stored task record padded through its description
// to exactly size bytes.
func sizedTaskLine(t *testing.T, id string, size int) string {
	t.Helper()
	head := `{"id":"` + id + sizedLinePrefix
	pad := size - len(head) - len(sizedLineSuffix)
	if pad < 0 {
		t.Fatalf("size %d too small for a task line", size)
	}
	return head + strings.Repeat("d", pad) + sizedLineSuffix
}

func plainTaskLine(id string) string {
	return `{"id":"` + id + `","title":"T","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

// setupVerbatimProject writes content to .tick/tasks.jsonl byte for byte.
func setupVerbatimProject(t *testing.T, content string) string {
	t.Helper()
	dir, tickDir := setupTickProject(t)
	if err := os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
	return dir
}

func assertListedIDs(t *testing.T, dir string, want ...string) {
	t.Helper()
	if got := listedIDs(t, dir, "list"); !slices.Equal(got, want) {
		t.Errorf("listed ids = %v, want %v", got, want)
	}
}

func TestStoreLineReading(t *testing.T) {
	t.Run("it lists a task whose line is exactly 65,536 bytes", func(t *testing.T) {
		dir, _ := setupRawProject(t, sizedTaskLine(t, "tick-aaa111", 65536))
		assertListedIDs(t, dir, "tick-aaa111")
	})

	t.Run("it lists tasks whose lines are 65,535, 65,536 and about 1 MiB bytes", func(t *testing.T) {
		dir, _ := setupRawProject(t,
			sizedTaskLine(t, "tick-aaa111", 65535),
			sizedTaskLine(t, "tick-bbb222", 65536),
			sizedTaskLine(t, "tick-ccc333", 1<<20),
		)
		assertListedIDs(t, dir, "tick-aaa111", "tick-bbb222", "tick-ccc333")
	})

	t.Run("it lists a CRLF store exactly as the same store with LF endings", func(t *testing.T) {
		lf := plainTaskLine("tick-aaa111") + "\n" + sizedTaskLine(t, "tick-bbb222", 70000) + "\n"
		lfOut := runToonCommand(t, setupVerbatimProject(t, lf), "list")
		crlfOut := runToonCommand(t, setupVerbatimProject(t, strings.ReplaceAll(lf, "\n", "\r\n")), "list")
		if crlfOut != lfOut {
			t.Errorf("CRLF list = %q, want %q", crlfOut, lfOut)
		}
		assertListedIDs(t, setupVerbatimProject(t, lf), "tick-aaa111", "tick-bbb222")
	})

	t.Run("it lists a final task line with no trailing newline", func(t *testing.T) {
		dir := setupVerbatimProject(t, plainTaskLine("tick-aaa111")+"\n"+plainTaskLine("tick-bbb222"))
		assertListedIDs(t, dir, "tick-aaa111", "tick-bbb222")
	})

	t.Run("it lists a final task line ending in one carriage return and no newline", func(t *testing.T) {
		dir := setupVerbatimProject(t, plainTaskLine("tick-aaa111")+"\n"+plainTaskLine("tick-bbb222")+"\r")
		assertListedIDs(t, dir, "tick-aaa111", "tick-bbb222")
	})

	t.Run("it lists a store whose file ends in a newline then a carriage return", func(t *testing.T) {
		dir := setupVerbatimProject(t, plainTaskLine("tick-aaa111")+"\n"+plainTaskLine("tick-bbb222")+"\n\r")
		assertListedIDs(t, dir, "tick-aaa111", "tick-bbb222")
	})

	t.Run("it skips empty lines between tasks", func(t *testing.T) {
		dir := setupVerbatimProject(t, "\n"+plainTaskLine("tick-aaa111")+"\n\n\n"+plainTaskLine("tick-bbb222")+"\n\n")
		assertListedIDs(t, dir, "tick-aaa111", "tick-bbb222")
	})

	t.Run("it fails on a whitespace-only line 3 after an empty line 2, naming line 3", func(t *testing.T) {
		dir := setupVerbatimProject(t, plainTaskLine("tick-aaa111")+"\n\n   \n"+plainTaskLine("tick-bbb222")+"\n")
		stdout, stderr, code := runList(t, dir)
		if code != 1 {
			t.Fatalf("exit code = %d, want 1; stdout = %q", code, stdout)
		}
		want := "Error: failed to parse tasks.jsonl: line 3: unexpected end of JSON input\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})
}
