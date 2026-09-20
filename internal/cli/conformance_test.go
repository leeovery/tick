package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
	"github.com/leeovery/tick/internal/testutil"
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

func conformanceStatsTasks() []task.Task {
	blocker := conformanceTask("tick-111aaa", "Blocker", task.StatusOpen, 0, "task")
	blocked := conformanceTask("tick-222bbb", "Blocked", task.StatusOpen, 1, "bug")
	blocked.BlockedBy = []string{blocker.ID}
	done := conformanceTask("tick-333ccc", "Finished", task.StatusDone, 3, "chore")
	done.Closed = &conformanceTime
	cancelled := conformanceTask("tick-444ddd", "Dropped", task.StatusCancelled, 4, "task")
	cancelled.Closed = &conformanceTime
	return []task.Task{
		blocker,
		blocked,
		conformanceTask("tick-555eee", "Running", task.StatusInProgress, 2, "feature"),
		done,
		cancelled,
	}
}

// conformanceDepGraph returns a three-task chain beside a task carrying no
// dependencies, covering every branch of the focused dependency view.
func conformanceDepGraph() []task.Task {
	upstream := conformanceTask("tick-a11111", "Upstream", task.StatusOpen, 2, "task")
	middle := conformanceTask("tick-b22222", "Middle", task.StatusOpen, 2, "task")
	middle.BlockedBy = []string{upstream.ID}
	downstream := conformanceTask("tick-c33333", "Downstream", task.StatusOpen, 2, "task")
	downstream.BlockedBy = []string{middle.ID}
	lone := conformanceTask("tick-d44444", "Lone", task.StatusOpen, 2, "")
	return []task.Task{upstream, middle, downstream, lone}
}

func conformanceUnconnectedTasks() []task.Task {
	return []task.Task{
		conformanceTask("tick-e11111", "First", task.StatusOpen, 2, "task"),
		conformanceTask("tick-f22222", "Second", task.StatusOpen, 2, "task"),
	}
}

func conformanceCyclePair() []task.Task {
	first := conformanceTask("tick-a99999", "First of the cycle", task.StatusOpen, 2, "task")
	second := conformanceTask("tick-b88888", "Second of the cycle", task.StatusOpen, 2, "task")
	first.BlockedBy = []string{second.ID}
	second.BlockedBy = []string{first.ID}
	return []task.Task{first, second}
}

var conformanceNotes = []task.Note{
	{Text: "First note", Created: conformanceTime},
	{Text: "Second note: with a colon", Created: conformanceTime.Add(time.Hour)},
}

// conformanceDetailTasks returns the dependency graph extended with a task
// carrying every optional field and the child it refers to; the graph supplies
// the parent and blocker it names.
func conformanceDetailTasks() []task.Task {
	full := conformanceTask("tick-e55555", "Full task", task.StatusDone, 1, "bug")
	full.Parent = "tick-a11111"
	full.Tags = []string{"backend", "ui"}
	full.Refs = []string{"https://x.dev/issues/3"}
	full.Description = "Line one\nLine two"
	full.Notes = conformanceNotes
	full.BlockedBy = []string{"tick-c33333"}
	full.Closed = &conformanceTime

	child := conformanceTask("tick-f66666", "Child task", task.StatusOpen, 3, "task")
	child.Parent = full.ID

	return append(conformanceDepGraph(), full, child)
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
	{
		Name:    "stats on a populated project",
		Command: "stats",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceStatsTasks())
			return dir, []string{"stats"}
		},
	},
	{
		Name:    "stats on an empty project",
		Command: "stats",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProject(t)
			return dir, []string{"stats"}
		},
	},
	{
		Name:    "dep tree on a populated graph",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDepGraph())
			return dir, []string{"dep", "tree"}
		},
	},
	{
		Name:    "dep tree with no dependencies",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceUnconnectedTasks())
			return dir, []string{"dep", "tree"}
		},
	},
	{
		Name:    "dep tree on a two-task cycle",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceCyclePair())
			return dir, []string{"dep", "tree"}
		},
	},
	{
		Name:    "dep tree for a task with both directions",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDepGraph())
			return dir, []string{"dep", "tree", "tick-b22222"}
		},
	},
	{
		Name:    "dep tree for a task with upstream only",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDepGraph())
			return dir, []string{"dep", "tree", "tick-c33333"}
		},
	},
	{
		Name:    "dep tree for a task with downstream only",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDepGraph())
			return dir, []string{"dep", "tree", "tick-a11111"}
		},
	},
	{
		Name:    "dep tree for a task with neither direction",
		Command: "dep tree",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDepGraph())
			return dir, []string{"dep", "tree", "tick-d44444"}
		},
	},
	{
		Name:         "dep tree under --quiet",
		Command:      "dep tree",
		NotADocument: "--quiet prints nothing at all rather than a document",
	},
	{
		Name:    "show on a task carrying every optional field",
		Command: "show",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDetailTasks())
			return dir, []string{"show", "tick-e55555"}
		},
	},
	{
		Name:    "show on a task carrying no optional field",
		Command: "show",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDetailTasks())
			return dir, []string{"show", "tick-d44444"}
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

func TestToonStatsConformance(t *testing.T) {
	t.Run("it decodes the populated stats document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "stats on a populated project")

		assertToonFields(t, doc, map[string]any{
			"total":       float64(5),
			"open":        float64(2),
			"in_progress": float64(1),
			"done":        float64(1),
			"cancelled":   float64(1),
			"ready":       float64(2),
			"blocked":     float64(1),
		})
		assertConformancePriorityCounts(t, doc, [5]int{1, 1, 1, 1, 1})
	})

	t.Run("it decodes the stats document for an empty project", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "stats on an empty project")

		assertToonFields(t, doc, map[string]any{
			"total":       float64(0),
			"open":        float64(0),
			"in_progress": float64(0),
			"done":        float64(0),
			"cancelled":   float64(0),
			"ready":       float64(0),
			"blocked":     float64(0),
		})
		assertConformancePriorityCounts(t, doc, [5]int{})
	})
}

func TestToonDepTreeConformance(t *testing.T) {
	t.Run("it decodes the populated full dep tree document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree on a populated graph")

		assertToonEdgeRows(t, doc, "dep_tree", []toonEdgeRow{
			{From: "tick-a11111", To: "tick-b22222"},
			{From: "tick-b22222", To: "tick-c33333"},
		})
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(1),
			"longest": float64(2),
			"blocked": float64(2),
		})
	})

	t.Run("it decodes the emptied full dep tree document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree with no dependencies")

		assertToonRowsEmpty(t, doc, "dep_tree")
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(0),
			"blocked": float64(0),
		})
	})

	t.Run("it decodes the full dep tree document for a cycle", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree on a two-task cycle")

		assertToonEdgeRows(t, doc, "dep_tree", []toonEdgeRow{
			{From: "tick-a99999", To: "tick-b88888"},
			{From: "tick-b88888", To: "tick-a99999"},
		})
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(1),
			"longest": float64(2),
			"blocked": float64(2),
		})
	})

	t.Run("it decodes the focused document with both directions", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree for a task with both directions")

		assertToonFields(t, doc, map[string]any{
			"id":     "tick-b22222",
			"title":  "Middle",
			"status": "open",
		})
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{{From: "tick-a11111", To: "tick-b22222"}})
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{{From: "tick-b22222", To: "tick-c33333"}})
	})

	t.Run("it decodes the focused document with upstream only", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree for a task with upstream only")

		assertToonFields(t, doc, map[string]any{
			"id":     "tick-c33333",
			"title":  "Downstream",
			"status": "open",
		})
		assertToonEdgeRows(t, doc, "blocked_by", []toonEdgeRow{
			{From: "tick-b22222", To: "tick-c33333"},
			{From: "tick-a11111", To: "tick-b22222"},
		})
		assertToonRowsEmpty(t, doc, "blocks")
	})

	t.Run("it decodes the focused document with downstream only", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree for a task with downstream only")

		assertToonFields(t, doc, map[string]any{
			"id":     "tick-a11111",
			"title":  "Upstream",
			"status": "open",
		})
		assertToonRowsEmpty(t, doc, "blocked_by")
		assertToonEdgeRows(t, doc, "blocks", []toonEdgeRow{
			{From: "tick-a11111", To: "tick-b22222"},
			{From: "tick-b22222", To: "tick-c33333"},
		})
	})

	t.Run("it decodes the focused document with neither direction", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "dep tree for a task with neither direction")

		assertToonFields(t, doc, map[string]any{
			"id":     "tick-d44444",
			"title":  "Lone",
			"status": "open",
		})
		assertToonRowsEmpty(t, doc, "blocked_by", "blocks")
	})
}

func TestToonDetailConformance(t *testing.T) {
	t.Run("it decodes the full detail document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "show on a task carrying every optional field")

		assertToonFields(t, doc, map[string]any{
			"id":          "tick-e55555",
			"title":       "Full task",
			"status":      "done",
			"priority":    float64(1),
			"type":        "bug",
			"parent":      "tick-a11111",
			"created":     task.FormatTimestamp(conformanceTime),
			"updated":     task.FormatTimestamp(conformanceTime),
			"closed":      task.FormatTimestamp(conformanceTime),
			"description": "Line one\nLine two",
		})
		assertToonStringList(t, doc, "tags", []string{"backend", "ui"})
		assertToonStringList(t, doc, "refs", []string{"https://x.dev/issues/3"})
		assertToonRelatedRow(t, doc, "blocked_by", RelatedTask{ID: "tick-c33333", Title: "Downstream", Status: "open"})
		assertToonRelatedRow(t, doc, "children", RelatedTask{ID: "tick-f66666", Title: "Child task", Status: "open"})
		assertToonNoteRows(t, doc, conformanceNotes)
	})

	t.Run("it decodes the bare detail document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "show on a task carrying no optional field")

		assertToonFields(t, doc, map[string]any{
			"id":       "tick-d44444",
			"title":    "Lone",
			"status":   "open",
			"priority": float64(2),
			"created":  task.FormatTimestamp(conformanceTime),
			"updated":  task.FormatTimestamp(conformanceTime),
		})
		assertToonKeysAbsent(t, doc, "type", "parent", "closed", "tags", "refs", "description")
		assertToonRowsEmpty(t, doc, "blocked_by", "children", "notes")
	})
}

func TestConformanceScopeBoundary(t *testing.T) {
	t.Run("it keeps the phase 1 and phase 3 decoded assertions", func(t *testing.T) {
		root := testutil.FindRepoRoot(t)
		required := map[string][]string{
			"toon_decode_test.go": {
				"func TestToonTaskDetailConformance",
				"func TestToonDepTreeFocusedConformance",
			},
			"toon_formatter_test.go": {
				"it emits stats counts as top-level named fields",
				"it emits the dep tree summary as top-level named fields",
			},
			"stats_test.go":    {"it decodes stats for a project with no tasks"},
			"dep_tree_test.go": {"it returns the emptied document when no task has dependencies"},
		}

		for file, names := range required {
			source, err := os.ReadFile(filepath.Join(root, "internal", "cli", file))
			if err != nil {
				t.Fatalf("reading %s: %v", file, err)
			}
			for _, name := range names {
				if !strings.Contains(string(source), name) {
					t.Errorf("%s no longer carries %q", file, name)
				}
			}
		}
	})
}

func assertConformancePriorityCounts(t *testing.T, doc map[string]any, want [5]int) {
	t.Helper()
	rows := toonRows(t, doc, "by_priority")
	if len(rows) != len(want) {
		t.Fatalf("by_priority has %d rows, want %d", len(rows), len(want))
	}
	for i, row := range rows {
		assertToonFields(t, row, map[string]any{
			"priority": float64(i),
			"count":    float64(want[i]),
		})
	}
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
