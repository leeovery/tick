package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
)

// conformanceDoc is one document the tool can produce. Command is the
// fully-qualified command name as commandFlags spells it, carrying no
// arguments and no flags. Setup seeds a project and returns its directory
// plus the arguments following "tick". NotADocument states why an entry's
// output is not a document; an entry carrying one carries no Setup.
type conformanceDoc struct {
	Name         string
	Command      string
	Setup        func(t *testing.T) (dir string, args []string)
	NotADocument string
}

var conformanceTime = time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC)

func conformanceTask(id, title string, status task.Status, priority int, taskType string) task.Task {
	return task.Task{
		ID:       id,
		Title:    title,
		Status:   status,
		Priority: priority,
		Type:     taskType,
		Created:  conformanceTime,
		Updated:  conformanceTime,
	}
}

var conformanceListTasks = []task.Task{
	conformanceTask("tick-aaa111", "Write, the parser", task.StatusInProgress, 1, "feature"),
	conformanceTask("tick-bbb222", "Fix the header", task.StatusOpen, 2, "bug"),
	conformanceTask("tick-ccc333", "Untyped task", task.StatusOpen, 3, ""),
}

func conformanceBlockedPair() []task.Task {
	blocker := conformanceTask("tick-ddd444", "Blocker", task.StatusOpen, 1, "task")
	blocked := conformanceTask("tick-eee555", "Blocked", task.StatusOpen, 2, "task")
	blocked.BlockedBy = []string{blocker.ID}
	return []task.Task{blocker, blocked}
}

var conformanceDocs = []conformanceDoc{
	{
		Name:    "list on a populated project",
		Command: "list",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceListTasks)
			return dir, []string{"list"}
		},
	},
	{
		Name:    "list on an empty project",
		Command: "list",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProject(t)
			return dir, []string{"list"}
		},
	},
	{
		Name:    "list with a filter matching nothing",
		Command: "list",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceListTasks)
			return dir, []string{"list", "--status", "done"}
		},
	},
	{
		Name:         "list under --quiet",
		Command:      "list",
		NotADocument: "--quiet prints bare task IDs, one per line, rather than a document",
	},
	{
		Name:    "ready with results",
		Command: "ready",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceBlockedPair())
			return dir, []string{"ready"}
		},
	},
	{
		Name:    "ready with no results",
		Command: "ready",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, []task.Task{conformanceTask("tick-fff666", "Closed", task.StatusDone, 1, "task")})
			return dir, []string{"ready"}
		},
	},
	{
		Name:    "blocked with results",
		Command: "blocked",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceBlockedPair())
			return dir, []string{"blocked"}
		},
	},
	{
		Name:    "blocked with no results",
		Command: "blocked",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceListTasks)
			return dir, []string{"blocked"}
		},
	},
}

// runTickConformance runs a tick command under --toon. IsTTY is true so the
// format the driver passes is what resolves the format.
func runTickConformance(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	code := app.Run(append([]string{"tick", "--toon"}, args...))
	return stdoutBuf.String(), stderrBuf.String(), code
}

// decodeConformanceDoc decodes a document from the inventory, naming the entry
// and quoting the document when it will not parse.
func decodeConformanceDoc(name, doc string) (map[string]any, error) {
	value, err := toon.DecodeString(doc)
	if err != nil {
		return nil, fmt.Errorf("%s: decode failed: %w\ndocument:\n%s", name, err, doc)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: decoded value is %T, want map[string]any\ndocument:\n%s", name, value, doc)
	}
	return obj, nil
}

// runConformanceDoc runs one inventory entry and returns its decoded document.
func runConformanceDoc(t *testing.T, entry conformanceDoc) map[string]any {
	t.Helper()
	dir, args := entry.Setup(t)
	stdout, stderr, exitCode := runTickConformance(t, dir, args...)
	if exitCode != 0 {
		t.Fatalf("%s: exit code = %d, want 0; stderr = %q", entry.Name, exitCode, stderr)
	}
	doc, err := decodeConformanceDoc(entry.Name, stdout)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// decodeConformanceEntry runs the named inventory entry and returns its
// decoded document.
func decodeConformanceEntry(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, entry := range conformanceDocs {
		if entry.Name == name {
			return runConformanceDoc(t, entry)
		}
	}
	t.Fatalf("no conformance entry named %q", name)
	return nil
}

func TestToonOutputConformance(t *testing.T) {
	for _, entry := range conformanceDocs {
		t.Run(entry.Name, func(t *testing.T) {
			if entry.NotADocument != "" {
				t.Skip(entry.NotADocument)
			}
			runConformanceDoc(t, entry)
		})
	}
}

func TestConformanceInventoryWellFormed(t *testing.T) {
	inventoryProblems := func(t *testing.T, docs []conformanceDoc) []string {
		t.Helper()
		var problems []string
		seen := make(map[string]bool, len(docs))
		for _, entry := range docs {
			if seen[entry.Name] {
				problems = append(problems, fmt.Sprintf("duplicate entry name %q", entry.Name))
			}
			seen[entry.Name] = true
			if (entry.NotADocument != "") == (entry.Setup == nil) {
				continue
			}
			if entry.Setup == nil {
				problems = append(problems, fmt.Sprintf("entry %q carries neither a setup nor a reason", entry.Name))
				continue
			}
			problems = append(problems, fmt.Sprintf("entry %q carries both a setup and a reason", entry.Name))
		}
		return problems
	}

	t.Run("it accepts the inventory", func(t *testing.T) {
		for _, problem := range inventoryProblems(t, conformanceDocs) {
			t.Error(problem)
		}
	})

	t.Run("it rejects a duplicate document name", func(t *testing.T) {
		setup := func(t *testing.T) (string, []string) { return "", nil }
		docs := []conformanceDoc{
			{Name: "twice", Command: "list", Setup: setup},
			{Name: "twice", Command: "list", Setup: setup},
		}

		if problems := inventoryProblems(t, docs); len(problems) != 1 {
			t.Errorf("problems = %v, want one duplicate-name problem", problems)
		}
	})

	t.Run("it rejects an entry carrying both a setup and an exemption reason", func(t *testing.T) {
		docs := []conformanceDoc{{
			Name:         "both",
			Command:      "list",
			Setup:        func(t *testing.T) (string, []string) { return "", nil },
			NotADocument: "bare IDs",
		}}

		if problems := inventoryProblems(t, docs); len(problems) != 1 {
			t.Errorf("problems = %v, want one setup-and-reason problem", problems)
		}
	})

	t.Run("it rejects an entry carrying neither a setup nor an exemption reason", func(t *testing.T) {
		docs := []conformanceDoc{{Name: "neither", Command: "list"}}

		if problems := inventoryProblems(t, docs); len(problems) != 1 {
			t.Errorf("problems = %v, want one missing-both problem", problems)
		}
	})
}

func TestConformanceDecodeFailureReporting(t *testing.T) {
	t.Run("it names the failing document when a decode fails", func(t *testing.T) {
		malformed := "tasks[2]{id,title}:\n  tick-aaa111\n"

		_, err := decodeConformanceDoc("list on a populated project", malformed)

		if err == nil {
			t.Fatal("decode of a malformed document succeeded, want an error")
		}
		if !strings.Contains(err.Error(), "list on a populated project") {
			t.Errorf("error %q does not name the entry", err)
		}
		if !strings.Contains(err.Error(), malformed) {
			t.Errorf("error %q does not carry the document text", err)
		}
	})
}

func assertConformanceTaskRows(t *testing.T, doc map[string]any, want []task.Task) {
	t.Helper()
	rows := toonRows(t, doc, "tasks")
	if len(rows) != len(want) {
		t.Fatalf("tasks has %d rows, want %d", len(rows), len(want))
	}
	for i, row := range rows {
		assertToonFields(t, row, map[string]any{
			"id":       want[i].ID,
			"title":    want[i].Title,
			"status":   string(want[i].Status),
			"priority": float64(want[i].Priority),
			"type":     want[i].Type,
		})
	}
}

func TestToonTaskListConformance(t *testing.T) {
	t.Run("it decodes the populated list document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "list on a populated project")

		assertConformanceTaskRows(t, doc, conformanceListTasks)
	})

	t.Run("it decodes an empty type as an empty string", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "list on a populated project")

		rows := toonRows(t, doc, "tasks")
		if len(rows) != len(conformanceListTasks) {
			t.Fatalf("tasks has %d rows, want %d", len(rows), len(conformanceListTasks))
		}
		untyped := rows[len(rows)-1]
		assertToonFields(t, untyped, map[string]any{"type": "", "priority": float64(3)})
	})

	t.Run("it decodes the empty-project list document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "list on an empty project")

		assertConformanceTaskRows(t, doc, nil)
	})

	t.Run("it decodes the list document when a filter matches nothing", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "list with a filter matching nothing")

		assertConformanceTaskRows(t, doc, nil)
	})

	t.Run("it decodes the ready document with results", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "ready with results")

		assertConformanceTaskRows(t, doc, conformanceBlockedPair()[:1])
	})

	t.Run("it decodes the ready document with no results", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "ready with no results")

		assertConformanceTaskRows(t, doc, nil)
	})

	t.Run("it decodes the blocked document with results", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "blocked with results")

		assertConformanceTaskRows(t, doc, conformanceBlockedPair()[1:])
	})

	t.Run("it decodes the blocked document with no results", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "blocked with no results")

		assertConformanceTaskRows(t, doc, nil)
	})
}
