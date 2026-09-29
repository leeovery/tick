package cli

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var (
	jsonlFailureLine = regexp.MustCompile(`(?m)^✗ JSONL syntax: Line (\d+):`)
	listFailureLine  = regexp.MustCompile(`failed to parse tasks.jsonl: line (\d+)`)
	listFailReason   = regexp.MustCompile(`failed to parse tasks.jsonl: line \d+(?: \([^)]*\))?: (.*)`)
)

// relatedTaskLine renders a stored task naming parent and blocker, either of
// which may be empty.
func relatedTaskLine(id, parent, blocker string) string {
	fields := ""
	if parent != "" {
		fields += `"parent":"` + parent + `",`
	}
	if blocker != "" {
		fields += `"blocked_by":["` + blocker + `"],`
	}
	return `{"id":"` + id + `","title":"T","status":"open","priority":2,` + fields +
		`"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
}

// setupListedProject writes content verbatim and runs list once, so the cache
// is built from the file as it stands.
func setupListedProject(t *testing.T, content string) string {
	t.Helper()
	dir := setupVerbatimProject(t, content)
	if stdout, stderr, code := runList(t, dir); code != 0 {
		t.Fatalf("list exit code = %d; stdout = %q, stderr = %q", code, stdout, stderr)
	}
	return dir
}

func assertDoctorNoIssues(t *testing.T, dir string) {
	t.Helper()
	stdout, stderr, code := runDoctor(t, dir)
	if code != 0 {
		t.Fatalf("doctor exit code = %d, want 0; stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.HasSuffix(stdout, "\nNo issues found.\n") {
		t.Errorf("doctor stdout = %q, want it to end with No issues found", stdout)
	}
}

func doctorJSONLFailureLines(t *testing.T, stdout string) []int {
	t.Helper()
	var nums []int
	for _, m := range jsonlFailureLine.FindAllStringSubmatch(stdout, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("line number %q: %v", m[1], err)
		}
		nums = append(nums, n)
	}
	return nums
}

func listFailureLineNum(t *testing.T, dir string) int {
	t.Helper()
	stdout, stderr, code := runList(t, dir)
	if code != 1 {
		t.Fatalf("list exit code = %d, want 1; stdout = %q", code, stdout)
	}
	m := listFailureLine.FindStringSubmatch(stderr)
	if m == nil {
		t.Fatalf("list stderr = %q, want a parse failure naming a line", stderr)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("line number %q: %v", m[1], err)
	}
	return n
}

func assertDoctorAgreesWithList(t *testing.T, dir string, wantLine int) {
	t.Helper()
	if got := listFailureLineNum(t, dir); got != wantLine {
		t.Fatalf("list names line %d, want %d", got, wantLine)
	}
	stdout, _, code := runDoctor(t, dir)
	if code != 1 {
		t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
	}
	if got := doctorJSONLFailureLines(t, stdout); !slices.Equal(got, []int{wantLine}) {
		t.Errorf("doctor JSONL failures name lines %v, want [%d]; stdout = %q", got, wantLine, stdout)
	}
}

func listFailureReason(t *testing.T, dir string) string {
	t.Helper()
	_, stderr, _ := runList(t, dir)
	m := listFailReason.FindStringSubmatch(stderr)
	if m == nil {
		t.Fatalf("list stderr = %q, want a parse failure giving a reason", stderr)
	}
	return m[1]
}

func healthyLoadableContent() string {
	return strings.Join([]string{
		plainTaskLine("tick-aaa111"),
		relatedTaskLine("tick-bbb222", "tick-aaa111", ""),
		relatedTaskLine("tick-ccc333", "", "tick-aaa111"),
	}, "\n")
}

func TestDoctorLineReading(t *testing.T) {
	t.Run("it reports no issues when tasks before a line over 64 KiB name tasks after it", func(t *testing.T) {
		content := strings.Join([]string{
			relatedTaskLine("tick-aaa111", "tick-ddd444", ""),
			relatedTaskLine("tick-bbb222", "", "tick-eee555"),
			sizedTaskLine(t, "tick-ccc333", 70000),
			plainTaskLine("tick-ddd444"),
			plainTaskLine("tick-eee555"),
		}, "\n") + "\n"
		dir := setupListedProject(t, content)

		assertDoctorNoIssues(t, dir)
	})

	t.Run("it fails a whitespace-only line 3 after an empty line 2, naming line 3 as list does", func(t *testing.T) {
		dir := setupVerbatimProject(t, plainTaskLine("tick-aaa111")+"\n\n   \n"+plainTaskLine("tick-bbb222")+"\n")

		assertDoctorAgreesWithList(t, dir, 3)
	})

	t.Run("it names the line list names across CRLF, empty and whitespace-only lines", func(t *testing.T) {
		content := "\r\n" +
			plainTaskLine("tick-aaa111") + "\r\n" +
			"\n" +
			plainTaskLine("tick-bbb222") + "\n" +
			"\r\n" +
			" \t \r\n" +
			plainTaskLine("tick-ccc333") + "\r\n"
		dir := setupVerbatimProject(t, content)

		assertDoctorAgreesWithList(t, dir, 6)
	})

	t.Run("it reports no issues for a CRLF store as for the same store with LF endings", func(t *testing.T) {
		lf := healthyLoadableContent() + "\n"
		assertDoctorNoIssues(t, setupListedProject(t, lf))
		assertDoctorNoIssues(t, setupListedProject(t, strings.ReplaceAll(lf, "\n", "\r\n")))
	})

	t.Run("it reports no issues for a store whose file ends in a newline then a carriage return", func(t *testing.T) {
		dir := setupListedProject(t, healthyLoadableContent()+"\n\r")

		assertDoctorNoIssues(t, dir)
	})

	t.Run("it fails a line list cannot load, naming the line and giving the reason list gives", func(t *testing.T) {
		tests := []struct {
			name string
			line string
		}{
			{"whitespace-only", "   "},
			{"null", "null"},
			{"array", "[]"},
			{"empty object", "{}"},
			{"malformed JSON", `{"id":"tick-ccc333",`},
			{"wrong-typed field", `{"id":"tick-ccc333","title":"T","priority":"high"}`},
			{"unparseable created timestamp", `{"id":"tick-ccc333","title":"T","created":"2026-19-01"}`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				content := plainTaskLine("tick-aaa111") + "\n" + plainTaskLine("tick-bbb222") + "\n" + tc.line + "\n"
				dir := setupVerbatimProject(t, content)

				assertDoctorAgreesWithList(t, dir, 3)

				stdout, _, _ := runDoctor(t, dir)
				want := "✗ JSONL syntax: Line 3: " + listFailureReason(t, dir) + " — " + tc.line + "\n  → Manual fix required\n"
				if !strings.Contains(stdout, want) {
					t.Errorf("doctor stdout = %q, want it to contain %q", stdout, want)
				}
			})
		}
	})

	t.Run("it reports one JSONL failure per line that does not load, each naming its own line", func(t *testing.T) {
		content := strings.Join([]string{
			plainTaskLine("tick-aaa111"),
			"null",
			plainTaskLine("tick-bbb222"),
			"{}",
			"[]",
		}, "\n") + "\n"
		dir := setupVerbatimProject(t, content)

		stdout, _, code := runDoctor(t, dir)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		if got := doctorJSONLFailureLines(t, stdout); !slices.Equal(got, []int{2, 4, 5}) {
			t.Errorf("doctor JSONL failures name lines %v, want [2 4 5]; stdout = %q", got, stdout)
		}
	})

	t.Run("it passes the JSONL check on a line that loads with values outside the allowed ones", func(t *testing.T) {
		tests := []struct {
			name string
			line string
		}{
			{"unknown status", strings.Replace(plainTaskLine("tick-ccc333"), `"status":"open"`, `"status":"bogus"`, 1)},
			{"unknown type", strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"type":"weird"`, 1)},
			{"priority out of range", strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":9`, 1)},
			{"repeated tag", strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"tags":["x","x"]`, 1)},
			{"repeated ref", strings.Replace(plainTaskLine("tick-ccc333"), `"priority":2`, `"priority":2,"refs":["gh-1","gh-1"]`, 1)},
			{"repeated blocker", relatedTaskLine("tick-ccc333", "", "tick-aaa111") + "\n" +
				strings.Replace(plainTaskLine("tick-ddd444"), `"priority":2`, `"priority":2,"blocked_by":["tick-aaa111","tick-aaa111"]`, 1)},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				content := plainTaskLine("tick-aaa111") + "\n" + plainTaskLine("tick-bbb222") + "\n" + tc.line + "\n"
				dir := setupVerbatimProject(t, content)

				stdout, _, _ := runDoctor(t, dir)

				if !strings.Contains(stdout, "✓ JSONL syntax: OK\n") {
					t.Errorf("doctor stdout = %q, want a passing JSONL syntax check", stdout)
				}
				for line := range strings.SplitSeq(stdout, "\n") {
					if strings.HasPrefix(line, "✗") && !strings.HasPrefix(line, "✗ Cache:") {
						t.Errorf("doctor reports %q; want no failure but the cache's", line)
					}
				}
			})
		}
	})

	t.Run("it reports no issues for a valid store with a fresh cache", func(t *testing.T) {
		assertDoctorNoIssues(t, setupListedProject(t, healthyLoadableContent()+"\n"))
	})

	t.Run("it leaves a line that is not an object or has no string id to the JSONL check alone", func(t *testing.T) {
		nonStringID := `{"id":7,"title":"T","status":"open","priority":2,"parent":"tick-ffffff","blocked_by":["tick-ffffff"],` +
			`"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		content := strings.Join([]string{
			plainTaskLine("tick-aaa111"),
			"[]",
			"null",
			nonStringID,
		}, "\n") + "\n"
		dir := setupVerbatimProject(t, content)

		stdout, _, code := runDoctor(t, dir)

		if code != 1 {
			t.Fatalf("doctor exit code = %d, want 1; stdout = %q", code, stdout)
		}
		if got := doctorJSONLFailureLines(t, stdout); !slices.Equal(got, []int{2, 3, 4}) {
			t.Errorf("doctor JSONL failures name lines %v, want [2 3 4]; stdout = %q", got, stdout)
		}
		for _, name := range []string{
			"Orphaned parents", "Orphaned dependencies", "Self-referential dependencies",
			"Dependency cycles", "Child blocked by parent", "Parent done with open children",
		} {
			if !strings.Contains(stdout, "✓ "+name+": OK\n") {
				t.Errorf("doctor stdout = %q, want %s passing", stdout, name)
			}
		}
	})
}
