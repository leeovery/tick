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
}
