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
