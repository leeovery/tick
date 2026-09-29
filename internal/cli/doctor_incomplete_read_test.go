package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/doctor"
)

var lineConsumingChecks = []string{
	"JSONL syntax",
	"ID format",
	"ID uniqueness",
	"Orphaned parents",
	"Orphaned dependencies",
	"Self-referential dependencies",
	"Dependency cycles",
	"Child blocked by parent",
	"Parent done with open children",
	"Sequence uniqueness",
}

// cutShortReader yields data, then fails every later read with err.
type cutShortReader struct {
	data []byte
	err  error
}

func (r *cutShortReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func (r *cutShortReader) Close() error { return nil }

func runDoctorWithOpener(t *testing.T, dir string, open func(string) (io.ReadCloser, error)) (string, int) {
	t.Helper()
	ctx := context.Background()
	if open != nil {
		ctx = doctor.WithTasksOpener(ctx, open)
	}
	var stdout, stderr bytes.Buffer
	code := RunDoctor(ctx, &stdout, &stderr, filepath.Join(dir, ".tick"))
	return stdout.String(), code
}

func reportFor(stdout, check string) []string {
	var got []string
	for line := range strings.SplitSeq(stdout, "\n") {
		if strings.HasPrefix(line, "✓ "+check+":") || strings.HasPrefix(line, "✗ "+check+":") {
			got = append(got, line)
		}
	}
	return got
}

func TestDoctorIncompleteRead(t *testing.T) {
	twoLines := plainTaskLine("tick-aaa111") + "\n" + plainTaskLine("tick-bbb222") + "\n"
	boom := errors.New("device went away")
	cutAtLine3 := func(string) (io.ReadCloser, error) {
		return &cutShortReader{data: []byte(twoLines), err: boom}, nil
	}
	wantDetail := "tasks.jsonl could not be read in full: read line 3: device went away"

	t.Run("it fails naming line 3 when the read stops there after lines 1 and 2", func(t *testing.T) {
		dir := setupListedProject(t, twoLines)

		stdout, code := runDoctorWithOpener(t, dir, cutAtLine3)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		if !strings.Contains(stdout, wantDetail) {
			t.Errorf("doctor stdout = %q, want it to contain %q", stdout, wantDetail)
		}
	})

	t.Run("it fails every line-consuming check once with the read error in place of its verdict", func(t *testing.T) {
		dir := setupListedProject(t, twoLines)

		stdout, _ := runDoctorWithOpener(t, dir, cutAtLine3)

		for _, check := range lineConsumingChecks {
			want := []string{"✗ " + check + ": " + wantDetail}
			if got := reportFor(stdout, check); !slices.Equal(got, want) {
				t.Errorf("%s report = %q, want %q; stdout = %q", check, got, want, stdout)
			}
		}
		if got, want := reportFor(stdout, "Cache"), []string{"✓ Cache: OK"}; !slices.Equal(got, want) {
			t.Errorf("Cache report = %q, want %q", got, want)
		}
		wantSummary := "\n10 issues found.\n"
		if !strings.HasSuffix(stdout, wantSummary) {
			t.Errorf("doctor stdout = %q, want it to end with %q", stdout, wantSummary)
		}
	})

	t.Run("it never reports tasks.jsonl not found for a read cut short", func(t *testing.T) {
		dir := setupListedProject(t, twoLines)

		stdout, _ := runDoctorWithOpener(t, dir, cutAtLine3)

		if strings.Contains(stdout, "tasks.jsonl not found") {
			t.Errorf("doctor stdout = %q, want no not-found result", stdout)
		}
	})

	t.Run("it reports tasks.jsonl not found from every line-consuming check when the file is missing", func(t *testing.T) {
		dir := setupListedProject(t, twoLines)
		if err := os.Remove(filepath.Join(dir, ".tick", "tasks.jsonl")); err != nil {
			t.Fatalf("remove tasks.jsonl: %v", err)
		}

		stdout, code := runDoctorWithOpener(t, dir, nil)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		for _, check := range lineConsumingChecks {
			want := []string{"✗ " + check + ": tasks.jsonl not found"}
			if got := reportFor(stdout, check); !slices.Equal(got, want) {
				t.Errorf("%s report = %q, want %q; stdout = %q", check, got, want, stdout)
			}
		}
	})
}

// countingOpener opens tasks.jsonl for real, except that each open faultOn
// reports true for is cut short at line 3 with boom. It counts every open.
func countingOpener(twoLines string, boom error, faultOn func(open int) bool) (func(string) (io.ReadCloser, error), *int) {
	opens := 0
	return func(path string) (io.ReadCloser, error) {
		opens++
		if faultOn(opens) {
			return &cutShortReader{data: []byte(twoLines), err: boom}, nil
		}
		return os.Open(path)
	}, &opens
}

func TestDoctorSharesOneScan(t *testing.T) {
	twoLines := plainTaskLine("tick-aaa111") + "\n" + plainTaskLine("tick-bbb222") + "\n"
	boom := errors.New("device went away")
	wantDetail := "tasks.jsonl could not be read in full: read line 3: device went away"
	firstOnly := func(open int) bool { return open == 1 }
	every := func(int) bool { return true }
	never := func(int) bool { return false }

	t.Run("it fails every line-consuming check with the read error when only the first read faults", func(t *testing.T) {
		dir := setupListedProject(t, twoLines)
		open, _ := countingOpener(twoLines, boom, firstOnly)

		stdout, code := runDoctorWithOpener(t, dir, open)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		for _, check := range lineConsumingChecks {
			want := []string{"✗ " + check + ": " + wantDetail}
			if got := reportFor(stdout, check); !slices.Equal(got, want) {
				t.Errorf("%s report = %q, want %q; stdout = %q", check, got, want, stdout)
			}
		}
	})

	cacheCases := []struct {
		name       string
		breakCache func(t *testing.T, tickDir string)
		wantCache  string
	}{
		{
			name: "missing",
			breakCache: func(t *testing.T, tickDir string) {
				if err := os.Remove(filepath.Join(tickDir, "cache.db")); err != nil {
					t.Fatalf("remove cache.db: %v", err)
				}
			},
			wantCache: "✗ Cache: cache.db not found — cache has not been built",
		},
		{
			name: "stale",
			breakCache: func(t *testing.T, tickDir string) {
				content := twoLines + plainTaskLine("tick-ccc333") + "\n"
				if err := os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte(content), 0o644); err != nil {
					t.Fatalf("rewrite tasks.jsonl: %v", err)
				}
			},
			wantCache: "✗ Cache: cache.db is stale — hash mismatch between tasks.jsonl and cache",
		},
	}
	for _, tc := range cacheCases {
		t.Run("it points a "+tc.name+" cache at the failing lines, never at rebuild, when only the first read faults", func(t *testing.T) {
			dir := setupListedProject(t, twoLines)
			tc.breakCache(t, filepath.Join(dir, ".tick"))
			open, _ := countingOpener(twoLines, boom, firstOnly)

			stdout, _ := runDoctorWithOpener(t, dir, open)

			if got, want := reportFor(stdout, "Cache"), []string{tc.wantCache}; !slices.Equal(got, want) {
				t.Errorf("Cache report = %q, want %q; stdout = %q", got, want, stdout)
			}
			wantSuggestion := "  → Fix the tasks.jsonl lines the JSONL syntax check names, then run tick doctor again"
			if !slices.Contains(strings.Split(stdout, "\n"), wantSuggestion) {
				t.Errorf("doctor stdout = %q, want the line %q", stdout, wantSuggestion)
			}
			for line := range strings.SplitSeq(stdout, "\n") {
				if strings.Contains(line, "tick rebuild") {
					t.Errorf("doctor report line %q names tick rebuild; stdout = %q", line, stdout)
				}
			}
		})
	}

	openCases := []struct {
		name    string
		faultOn func(int) bool
	}{
		{"the store is readable", never},
		{"every read faults", every},
		{"only the first read faults", firstOnly},
	}
	for _, tc := range openCases {
		t.Run("it opens tasks.jsonl through the line reader once when "+tc.name, func(t *testing.T) {
			dir := setupListedProject(t, twoLines)
			open, opens := countingOpener(twoLines, boom, tc.faultOn)

			stdout, _ := runDoctorWithOpener(t, dir, open)

			if *opens != 1 {
				t.Errorf("tasks.jsonl opens = %d, want 1; stdout = %q", *opens, stdout)
			}
		})
	}
}
