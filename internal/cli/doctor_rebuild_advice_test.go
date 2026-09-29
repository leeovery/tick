package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	rebuildAdvice     = "Run `tick rebuild` to refresh cache"
	fixLinesAdvice    = "Fix the tasks.jsonl lines the JSONL syntax check names, then run tick doctor again"
	staleCacheDetails = "cache.db is stale — hash mismatch between tasks.jsonl and cache"
)

func appendToTasks(t *testing.T, dir, text string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, ".tick", "tasks.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open tasks.jsonl: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatalf("append to tasks.jsonl: %v", err)
	}
}

func writeTasks(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".tick", "tasks.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("write tasks.jsonl: %v", err)
	}
}

func doctorSuggestions(stdout string) []string {
	var got []string
	for line := range strings.SplitSeq(stdout, "\n") {
		if s, ok := strings.CutPrefix(line, "  → "); ok {
			got = append(got, s)
		}
	}
	return got
}

// cacheReport returns the Cache check's result line and the suggestion line
// that follows it, if any.
func cacheReport(stdout string) (string, string) {
	lines := strings.Split(stdout, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "✓ Cache:") || strings.HasPrefix(line, "✗ Cache:") {
			if i+1 < len(lines) {
				if s, ok := strings.CutPrefix(lines[i+1], "  → "); ok {
					return line, s
				}
			}
			return line, ""
		}
	}
	return "", ""
}

func assertNoRebuildAdvice(t *testing.T, stdout string) {
	t.Helper()
	for _, s := range doctorSuggestions(stdout) {
		if strings.Contains(s, "tick rebuild") {
			t.Errorf("doctor suggests %q; want no suggestion naming tick rebuild; stdout = %q", s, stdout)
		}
	}
}

func assertCacheReport(t *testing.T, stdout, wantLine, wantSuggestion string) {
	t.Helper()
	line, suggestion := cacheReport(stdout)
	if line != wantLine {
		t.Errorf("Cache report = %q, want %q; stdout = %q", line, wantLine, stdout)
	}
	if suggestion != wantSuggestion {
		t.Errorf("Cache suggestion = %q, want %q; stdout = %q", suggestion, wantSuggestion, stdout)
	}
}

func twoTaskContent() string {
	return plainTaskLine("tick-aaa111") + "\n" + plainTaskLine("tick-bbb222") + "\n"
}

func TestDoctorRebuildAdvice(t *testing.T) {
	t.Run("it reports a stale cache but points at the named lines while a line fails to load", func(t *testing.T) {
		dir := setupListedProject(t, twoTaskContent())
		appendToTasks(t, dir, "null\n")

		stdout, _, code := runDoctor(t, dir)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		assertCacheReport(t, stdout, "✗ Cache: "+staleCacheDetails, fixLinesAdvice)
		assertNoRebuildAdvice(t, stdout)
	})

	t.Run("it reports cache.db not found but points at the named lines while a line fails to load", func(t *testing.T) {
		dir := setupVerbatimProject(t, twoTaskContent()+"null\n")

		stdout, _, _ := runDoctor(t, dir)

		assertCacheReport(t, stdout, "✗ Cache: cache.db not found — cache has not been built", fixLinesAdvice)
		assertNoRebuildAdvice(t, stdout)
	})

	t.Run("it names no tick rebuild for each line the store cannot load", func(t *testing.T) {
		tests := []struct {
			name string
			line string
		}{
			{"whitespace-only", "   "},
			{"null", "null"},
			{"array", "[]"},
			{"empty object", "{}"},
			{"wrong-typed field", `{"id":"tick-ccc333","title":"T","priority":"high"}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				dir := setupListedProject(t, twoTaskContent())
				appendToTasks(t, dir, tc.line+"\n")
				if _, _, code := runList(t, dir); code != 1 {
					t.Fatalf("list exit code = %d, want 1", code)
				}

				stdout, _, code := runDoctor(t, dir)

				if code != 1 {
					t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
				}
				assertCacheReport(t, stdout, "✗ Cache: "+staleCacheDetails, fixLinesAdvice)
				assertNoRebuildAdvice(t, stdout)
			})
		}
	})

	t.Run("it names no tick rebuild when the read of tasks.jsonl stops partway and the cache is stale", func(t *testing.T) {
		dir := setupListedProject(t, twoTaskContent())
		appendToTasks(t, dir, plainTaskLine("tick-ccc333")+"\n")
		cutShort := func(string) (io.ReadCloser, error) {
			return &cutShortReader{data: []byte(twoTaskContent()), err: errors.New("device went away")}, nil
		}

		stdout, code := runDoctorWithOpener(t, dir, cutShort)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		assertCacheReport(t, stdout, "✗ Cache: "+staleCacheDetails, fixLinesAdvice)
		assertNoRebuildAdvice(t, stdout)
	})

	t.Run("it gives the usual rebuild advice once the failing line is fixed by hand and the cache is still stale", func(t *testing.T) {
		dir := setupListedProject(t, twoTaskContent())
		appendToTasks(t, dir, "null\n")
		if stdout, _, _ := runDoctor(t, dir); slices.Contains(doctorSuggestions(stdout), rebuildAdvice) {
			t.Fatalf("doctor stdout = %q, want no rebuild advice before the fix", stdout)
		}

		writeTasks(t, dir, twoTaskContent()+plainTaskLine("tick-ccc333")+"\n")
		stdout, _, code := runDoctor(t, dir)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		assertCacheReport(t, stdout, "✗ Cache: "+staleCacheDetails, rebuildAdvice)
	})

	t.Run("it keeps the rebuild advice for a repeated list value, and that rebuild fails naming the task and value", func(t *testing.T) {
		tests := []struct {
			name      string
			line      string
			wantError string
		}{
			{
				"repeated tag",
				strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"tags":["x","x"]`, 1),
				"failed to insert tag tick-ccc333 -> x",
			},
			{
				"repeated ref",
				strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"refs":["gh-1","gh-1"]`, 1),
				"failed to insert ref tick-ccc333 -> gh-1",
			},
			{
				"repeated blocker",
				strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"blocked_by":["tick-aaa111","tick-aaa111"]`, 1),
				"failed to insert dependency tick-ccc333 -> tick-aaa111",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				dir := setupListedProject(t, twoTaskContent())
				appendToTasks(t, dir, tc.line+"\n")

				stdout, _, _ := runDoctor(t, dir)
				assertCacheReport(t, stdout, "✗ Cache: "+staleCacheDetails, rebuildAdvice)

				_, stderr, code := runRebuild(t, dir)
				if code != 1 {
					t.Fatalf("rebuild exit code = %d, want 1; stderr = %q", code, stderr)
				}
				if !strings.Contains(stderr, tc.wantError) {
					t.Errorf("rebuild stderr = %q, want it to contain %q", stderr, tc.wantError)
				}
			})
		}
	})
}
