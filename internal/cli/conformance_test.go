package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
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
// output is not a document under any driver; an entry carrying one carries no
// Setup. ToonNotADocument states why the toon driver skips an entry the json
// driver runs.
type conformanceDoc struct {
	Name             string
	Command          string
	Setup            func(t *testing.T) (dir string, args []string)
	NotADocument     string
	ToonNotADocument string
}

// The drivers that run the inventory, one per machine format.
const (
	conformanceToonDriver = "toon"
	conformanceJSONDriver = "json"
)

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

func conformanceStatusTask(id, title string, status task.Status) task.Task {
	tk := conformanceTask(id, title, status, 2, "task")
	if status == task.StatusDone || status == task.StatusCancelled {
		tk.Closed = &conformanceTime
	}
	return tk
}

func conformanceLoneTask(status task.Status) []task.Task {
	return []task.Task{conformanceStatusTask("tick-a00001", "Lone", status)}
}

func conformanceStatusFamily(parentStatus, childStatus task.Status) []task.Task {
	parent := conformanceStatusTask("tick-p00001", "Parent", parentStatus)
	child := conformanceStatusTask("tick-c00001", "Child", childStatus)
	child.Parent = parent.ID
	return []task.Task{parent, child}
}

// conformanceDoneLineage returns a done task under a done parent, so creating
// a child under it reopens both.
func conformanceDoneLineage() []task.Task {
	grandparent := conformanceStatusTask("tick-g00001", "Grandparent", task.StatusDone)
	parent := conformanceStatusTask("tick-p00001", "Parent", task.StatusDone)
	parent.Parent = grandparent.ID
	return []task.Task{grandparent, parent}
}

func conformanceNotedTask() []task.Task {
	noted := conformanceTask("tick-t00001", "Noted", task.StatusOpen, 2, "task")
	noted.Notes = conformanceNotes
	return []task.Task{noted}
}

const conformanceSelectedID = "tick-x00001"

const conformanceSelectedDescription = "Line one\nLine two"

func conformanceSelectedTask() []task.Task {
	selected := conformanceTask(conformanceSelectedID, "Selected", task.StatusOpen, 2, "task")
	selected.Description = conformanceSelectedDescription
	selected.Notes = conformanceNotes
	return []task.Task{selected}
}

func conformanceMovingTask() task.Task {
	return conformanceTask("tick-m00001", "Moving", task.StatusOpen, 2, "task")
}

// conformanceCompletingParent returns a parent under parentID whose only
// other child is finished, so moving its open child away completes it.
func conformanceCompletingParent(parentID string) []task.Task {
	parent := conformanceTask("tick-o00001", "Old parent", task.StatusOpen, 2, "task")
	parent.Parent = parentID
	moving := conformanceMovingTask()
	moving.Parent = parent.ID
	sibling := conformanceStatusTask("tick-s00001", "Sibling", task.StatusDone)
	sibling.Parent = parent.ID
	return []task.Task{parent, moving, sibling}
}

// conformanceSharedAncestor returns a done root above both a parent that
// completes when its open child leaves and a done parent that reopens when
// that child arrives.
func conformanceSharedAncestor() []task.Task {
	root := conformanceStatusTask("tick-r00001", "Root", task.StatusDone)
	newParent := conformanceStatusTask("tick-n00001", "New parent", task.StatusDone)
	newParent.Parent = root.ID
	return append([]task.Task{root, newParent}, conformanceCompletingParent(root.ID)...)
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
	{
		Name:    "show with a multi-field selection",
		Command: "show",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceSelectedTask())
			return dir, []string{"show", conformanceSelectedID, "--field", "description,notes"}
		},
	},
	{
		Name:    "show with a selection narrowed by position",
		Command: "show",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceSelectedTask())
			return dir, []string{"show", conformanceSelectedID, "--field", "description,notes.2"}
		},
	},
	{
		Name:    "show with a single list section selected",
		Command: "show",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceSelectedTask())
			return dir, []string{"show", conformanceSelectedID, "--field", "notes"}
		},
	},
	{
		Name:         "show with a single non-list field selected",
		Command:      "show",
		NotADocument: "a selection naming one field with a bare form prints that value's own bytes and a newline rather than a document",
	},
	{
		Name:         "show with a selection naming only absent fields",
		Command:      "show",
		NotADocument: "a selection whose every name prints nothing prints no bytes at all, which is nothing rather than an empty document",
	},
	{
		Name:         "show under --quiet with a selection",
		Command:      "show",
		NotADocument: "--quiet alongside a selection is refused, so there is neither a document nor a bare value",
	},
	{
		Name:    "start with no cascade",
		Command: "start",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceLoneTask(task.StatusOpen))
			return dir, []string{"start", "tick-a00001"}
		},
	},
	{
		Name:    "done with no cascade",
		Command: "done",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceLoneTask(task.StatusInProgress))
			return dir, []string{"done", "tick-a00001"}
		},
	},
	{
		Name:    "cancel with no cascade",
		Command: "cancel",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceLoneTask(task.StatusOpen))
			return dir, []string{"cancel", "tick-a00001"}
		},
	},
	{
		Name:    "reopen with no cascade",
		Command: "reopen",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceLoneTask(task.StatusDone))
			return dir, []string{"reopen", "tick-a00001"}
		},
	},
	{
		Name:    "start cascading to an open parent",
		Command: "start",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceStatusFamily(task.StatusOpen, task.StatusOpen))
			return dir, []string{"start", "tick-c00001"}
		},
	},
	{
		Name:    "done cascading to an open child",
		Command: "done",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceStatusFamily(task.StatusOpen, task.StatusOpen))
			return dir, []string{"done", "tick-p00001"}
		},
	},
	{
		Name:    "cancel cascading to an open child",
		Command: "cancel",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceStatusFamily(task.StatusOpen, task.StatusOpen))
			return dir, []string{"cancel", "tick-p00001"}
		},
	},
	{
		Name:    "reopen cascading to a done parent",
		Command: "reopen",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceStatusFamily(task.StatusDone, task.StatusDone))
			return dir, []string{"reopen", "tick-c00001"}
		},
	},
	{
		Name:    "create with no parent",
		Command: "create",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProject(t)
			return dir, []string{"create", "A new task"}
		},
	},
	{
		Name:    "create under a done parent",
		Command: "create",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceDoneLineage())
			return dir, []string{"create", "A new child", "--parent", "tick-p00001"}
		},
	},
	{
		Name:         "create under --quiet",
		Command:      "create",
		NotADocument: "--quiet prints the bare ID of the created task rather than a document",
	},
	{
		Name:    "note add on a task carrying a note",
		Command: "note add",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceNotedTask())
			return dir, []string{"note", "add", "tick-t00001", "Third note"}
		},
	},
	{
		Name:    "note remove on a task carrying a note",
		Command: "note remove",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceNotedTask())
			return dir, []string{"note", "remove", "tick-t00001", "1"}
		},
	},
	{
		Name:    "update with no status movement",
		Command: "update",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, []task.Task{conformanceMovingTask()})
			return dir, []string{"update", "tick-m00001", "--title", "Renamed"}
		},
	},
	{
		Name:    "update moving a task under a done parent",
		Command: "update",
		Setup: func(t *testing.T) (string, []string) {
			tasks := []task.Task{conformanceMovingTask(), conformanceStatusTask("tick-n00001", "New parent", task.StatusDone)}
			dir, _ := setupTickProjectWithTasks(t, tasks)
			return dir, []string{"update", "tick-m00001", "--parent", "tick-n00001"}
		},
	},
	{
		Name:    "update moving a task away from a completed parent",
		Command: "update",
		Setup: func(t *testing.T) (string, []string) {
			tasks := append(conformanceCompletingParent(""), conformanceTask("tick-k00001", "Other parent", task.StatusOpen, 2, "task"))
			dir, _ := setupTickProjectWithTasks(t, tasks)
			return dir, []string{"update", "tick-m00001", "--parent", "tick-k00001"}
		},
	},
	{
		Name:    "update where reopen and completion meet on a shared ancestor",
		Command: "update",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceSharedAncestor())
			return dir, []string{"update", "tick-m00001", "--parent", "tick-n00001"}
		},
	},
	{
		Name:             "dep add joining two unconnected tasks",
		Command:          "dep add",
		ToonNotADocument: "toon confirms the dependency change in a line of prose rather than a document",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceUnconnectedTasks())
			return dir, []string{"dep", "add", "tick-f22222", "tick-e11111"}
		},
	},
	{
		Name:             "dep remove releasing a blocked task",
		Command:          "dep remove",
		ToonNotADocument: "toon confirms the dependency change in a line of prose rather than a document",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceBlockedPair())
			return dir, []string{"dep", "remove", "tick-eee555", "tick-ddd444"}
		},
	},
	{
		Name:             "remove on a task nothing depends on",
		Command:          "remove",
		ToonNotADocument: "toon confirms what was removed in a line of prose rather than a document",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceListTasks)
			return dir, []string{"remove", "tick-bbb222", "--force"}
		},
	},
	{
		Name:             "init in an uninitialised directory",
		Command:          "init",
		ToonNotADocument: "toon confirms the initialisation in a line of prose rather than a document",
		Setup: func(t *testing.T) (string, []string) {
			return t.TempDir(), []string{"init"}
		},
	},
	{
		Name:             "rebuild on a populated project",
		Command:          "rebuild",
		ToonNotADocument: "toon confirms the rebuild in a line of prose rather than a document",
		Setup: func(t *testing.T) (string, []string) {
			dir, _ := setupTickProjectWithTasks(t, conformanceListTasks)
			return dir, []string{"rebuild"}
		},
	},
}

// runTickConformance runs a tick command under the given format flag. IsTTY is
// true so the format the driver passes is what resolves the format.
func runTickConformance(t *testing.T, dir, format string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	code := app.Run(append([]string{"tick", format}, args...))
	return stdoutBuf.String(), stderrBuf.String(), code
}

// runConformanceCommand runs one inventory entry under a format flag and
// returns what it printed to stdout.
func runConformanceCommand(t *testing.T, entry conformanceDoc, format string) string {
	t.Helper()
	dir, args := entry.Setup(t)
	stdout, stderr, exitCode := runTickConformance(t, dir, format, args...)
	if exitCode != 0 {
		t.Fatalf("%s: exit code = %d, want 0; stderr = %q", entry.Name, exitCode, stderr)
	}
	return stdout
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
	stdout := runConformanceCommand(t, entry, "--toon")
	doc, err := decodeConformanceDoc(entry.Name, stdout)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// conformanceEntry returns the inventory entry of the given name.
func conformanceEntry(t *testing.T, name string) conformanceDoc {
	t.Helper()
	for _, entry := range conformanceDocs {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("no conformance entry named %q", name)
	return conformanceDoc{}
}

// decodeConformanceEntry runs the named inventory entry and returns its
// decoded document.
func decodeConformanceEntry(t *testing.T, name string) map[string]any {
	t.Helper()
	return runConformanceDoc(t, conformanceEntry(t, name))
}

// conformanceSkipReason reports why the named driver skips an entry, empty
// when it runs it. NotADocument holds for every driver, ToonNotADocument for
// the toon driver alone.
func conformanceSkipReason(driver string, entry conformanceDoc) string {
	if entry.NotADocument != "" {
		return entry.NotADocument
	}
	if driver == conformanceToonDriver {
		return entry.ToonNotADocument
	}
	return ""
}

// driveConformanceEntry hands one inventory entry to drive, skipping the
// entries the named driver does not run.
func driveConformanceEntry(t *testing.T, driver string, entry conformanceDoc, drive func(*testing.T, conformanceDoc)) {
	t.Helper()
	if reason := conformanceSkipReason(driver, entry); reason != "" {
		t.Skip(reason)
	}
	drive(t, entry)
}

func driveConformanceInventory(t *testing.T, driver string, drive func(*testing.T, conformanceDoc)) {
	t.Helper()
	for _, entry := range conformanceDocs {
		t.Run(entry.Name, func(t *testing.T) {
			driveConformanceEntry(t, driver, entry, drive)
		})
	}
}

// drivenConformanceEntries returns the names of the entries the named driver
// runs, in inventory order.
func drivenConformanceEntries(t *testing.T, driver string) []string {
	t.Helper()
	var driven []string
	driveConformanceInventory(t, driver, func(_ *testing.T, entry conformanceDoc) {
		driven = append(driven, entry.Name)
	})
	return driven
}

func TestToonOutputConformance(t *testing.T) {
	driveConformanceInventory(t, conformanceToonDriver, func(t *testing.T, entry conformanceDoc) {
		runConformanceDoc(t, entry)
	})
}

// jsonConformanceListKeys are the document keys whose value is a list. Each
// must unmarshal to a non-nil slice, never JSON null.
var jsonConformanceListKeys = []string{
	"changed", "trees", "blocked_by", "blocks",
	"tags", "refs", "notes", "children", "by_priority",
	"removed", "deps_updated",
}

// decodeSingleJSONValue decodes a stream carrying exactly one JSON value,
// naming the entry and quoting the stream when it will not parse or when a
// second value follows the first.
func decodeSingleJSONValue(name, stream string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(stream))
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%s: decode failed: %w\nstream:\n%s", name, err, stream)
	}
	var second any
	switch err := decoder.Decode(&second); {
	case errors.Is(err, io.EOF):
		return value, nil
	case err != nil:
		return nil, fmt.Errorf("%s: trailing bytes after the document: %w\nstream:\n%s", name, err, stream)
	default:
		return nil, fmt.Errorf("%s: a second value follows the document: %#v\nstream:\n%s", name, second, stream)
	}
}

// jsonTopLevelIsArray states whether a command's JSON document is a top-level
// array. The task-list commands are, because FormatTaskList marshals an array;
// every other document marshals an object.
func jsonTopLevelIsArray(command string) bool {
	return command == "list" || command == "ready" || command == "blocked"
}

// jsonShapeProblem reports the top-level shape the decoded value should have
// carried, empty when it carries it.
func jsonShapeProblem(name, command string, value any) string {
	if jsonTopLevelIsArray(command) {
		if _, ok := value.([]any); !ok {
			return fmt.Sprintf("%s: decoded value is %T, want []any", name, value)
		}
		return ""
	}
	if _, ok := value.(map[string]any); !ok {
		return fmt.Sprintf("%s: decoded value is %T, want map[string]any", name, value)
	}
	return ""
}

// jsonInvariantProblems reports every invariant the decoded value breaks,
// anywhere in it: a list key carrying null, a non-boolean auto, a non-numeric
// index.
func jsonInvariantProblems(name string, value any) []string {
	var problems []string
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(typed)) {
			if problem := jsonMemberProblem(name, key, typed[key]); problem != "" {
				problems = append(problems, problem)
			}
			problems = append(problems, jsonInvariantProblems(name, typed[key])...)
		}
	case []any:
		for _, item := range typed {
			problems = append(problems, jsonInvariantProblems(name, item)...)
		}
	}
	return problems
}

// jsonMemberProblem reports the invariant one key of a decoded object breaks,
// empty when it breaks none.
func jsonMemberProblem(name, key string, value any) string {
	switch {
	case slices.Contains(jsonConformanceListKeys, key):
		if _, ok := value.([]any); !ok {
			return fmt.Sprintf("%s: %q = %#v, want a list", name, key, value)
		}
	case key == "auto":
		if _, ok := value.(bool); !ok {
			return fmt.Sprintf("%s: %q = %#v, want a bool", name, key, value)
		}
	case key == "index":
		if _, ok := value.(float64); !ok {
			return fmt.Sprintf("%s: %q = %#v, want a number", name, key, value)
		}
	}
	return ""
}

func assertJSONInvariants(t *testing.T, name string, value any) {
	t.Helper()
	for _, problem := range jsonInvariantProblems(name, value) {
		t.Error(problem)
	}
}

// runJSONConformanceDoc runs one inventory entry under --json and returns the
// single value its stream carries.
func runJSONConformanceDoc(t *testing.T, entry conformanceDoc) any {
	t.Helper()
	stdout := runConformanceCommand(t, entry, "--json")
	value, err := decodeSingleJSONValue(entry.Name, stdout)
	if err != nil {
		t.Fatal(err)
	}
	if problem := jsonShapeProblem(entry.Name, entry.Command, value); problem != "" {
		t.Errorf("%s\nstream:\n%s", problem, stdout)
	}
	assertJSONInvariants(t, entry.Name, value)
	return value
}

// decodeJSONConformanceEntry runs the named inventory entry under --json and
// returns its decoded object.
func decodeJSONConformanceEntry(t *testing.T, name string) map[string]any {
	t.Helper()
	value := runJSONConformanceDoc(t, conformanceEntry(t, name))
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s: decoded value is %T, want map[string]any", name, value)
	}
	return obj
}

// declaredCommandProblem reports the disagreement between an entry's declared
// command and the args its setup returns, empty when the args begin with the
// command's words.
func declaredCommandProblem(t *testing.T, entry conformanceDoc) string {
	t.Helper()
	_, args := entry.Setup(t)
	words := strings.Fields(entry.Command)
	if len(args) >= len(words) && slices.Equal(args[:len(words)], words) {
		return ""
	}
	return fmt.Sprintf("entry %q declares command %q but its setup runs %v", entry.Name, entry.Command, args)
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
			if (entry.NotADocument != "") != (entry.Setup == nil) {
				if entry.Setup == nil {
					problems = append(problems, fmt.Sprintf("entry %q carries neither a setup nor a reason", entry.Name))
					continue
				}
				problems = append(problems, fmt.Sprintf("entry %q carries both a setup and a reason", entry.Name))
				continue
			}
			if entry.Setup == nil {
				continue
			}
			if problem := declaredCommandProblem(t, entry); problem != "" {
				problems = append(problems, problem)
			}
		}
		return problems
	}

	t.Run("it accepts the inventory", func(t *testing.T) {
		for _, problem := range inventoryProblems(t, conformanceDocs) {
			t.Error(problem)
		}
	})

	t.Run("it rejects a duplicate document name", func(t *testing.T) {
		setup := func(t *testing.T) (string, []string) { return "", []string{"list"} }
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

	t.Run("it rejects an entry whose setup runs a different command", func(t *testing.T) {
		docs := []conformanceDoc{{
			Name:    "ready with results",
			Command: "ready",
			Setup:   func(t *testing.T) (string, []string) { return "", []string{"list", "--ready"} },
		}}

		problems := inventoryProblems(t, docs)

		if len(problems) != 1 {
			t.Fatalf("problems = %v, want one command-mismatch problem", problems)
		}
		if !strings.Contains(problems[0], `"ready"`) || !strings.Contains(problems[0], "[list --ready]") {
			t.Errorf("problem %q names neither the declared command nor the args", problems[0])
		}
	})

	t.Run("it rejects an entry whose setup args are shorter than its declared command", func(t *testing.T) {
		docs := []conformanceDoc{{
			Name:    "dep tree",
			Command: "dep tree",
			Setup:   func(t *testing.T) (string, []string) { return "", []string{"dep"} },
		}}

		problems := inventoryProblems(t, docs)

		if len(problems) != 1 {
			t.Fatalf("problems = %v, want one command-mismatch problem", problems)
		}
		if !strings.Contains(problems[0], `"dep tree"`) || !strings.Contains(problems[0], "[dep]") {
			t.Errorf("problem %q names neither the declared command nor the args", problems[0])
		}
	})

	t.Run("it accepts an entry whose setup args carry flags after the command", func(t *testing.T) {
		docs := []conformanceDoc{{
			Name:    "show with a field selection",
			Command: "show",
			Setup: func(t *testing.T) (string, []string) {
				return "", []string{"show", "tick-a00001", "--field", "notes"}
			},
		}}

		if problems := inventoryProblems(t, docs); len(problems) != 0 {
			t.Errorf("problems = %v, want none", problems)
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
			{From: "tick-b88888", To: "tick-a99999"},
			{From: "tick-a99999", To: "tick-b88888"},
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
			{From: "tick-a11111", To: "tick-b22222"},
			{From: "tick-b22222", To: "tick-c33333"},
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

	t.Run("it decodes a multi-field selection document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "show with a multi-field selection")

		assertConformanceKeys(t, doc, "description", "notes")
		assertToonFields(t, doc, map[string]any{"description": conformanceSelectedDescription})
		assertToonNoteRows(t, doc, conformanceNotes)
	})

	t.Run("it decodes a position-narrowed selection document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "show with a selection narrowed by position")

		assertConformanceKeys(t, doc, "description", "notes")
		assertToonFields(t, doc, map[string]any{"description": conformanceSelectedDescription})
		rows := toonRows(t, doc, "notes")
		if len(rows) != 1 {
			t.Fatalf("notes has %d rows, want 1", len(rows))
		}
		assertToonFields(t, rows[0], map[string]any{
			"index":   float64(2),
			"text":    conformanceNotes[1].Text,
			"created": task.FormatTimestamp(conformanceNotes[1].Created),
		})
	})

	t.Run("it decodes a single list-section selection document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "show with a single list section selected")

		assertConformanceKeys(t, doc, "notes")
		assertToonNoteRows(t, doc, conformanceNotes)
	})
}

// conformanceFormatSpellings are the four ways a format reaches a command: no
// flag at all, and each of the three explicit ones.
var conformanceFormatSpellings = [][]string{nil, {"--toon"}, {"--pretty"}, {"--json"}}

// runConformanceShow runs a show command under the given format spelling.
func runConformanceShow(t *testing.T, dir string, format []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return runTick(t, dir, slices.Concat(format, []string{"show"}, args)...)
}

func TestFieldSelectionExemptions(t *testing.T) {
	setup := func(t *testing.T) string {
		t.Helper()
		dir, _ := setupTickProjectWithTasks(t, conformanceSelectedTask())
		return dir
	}
	bareValue := conformanceSelectedDescription + "\n"

	t.Run("it prints a bare value as its bytes plus one newline", func(t *testing.T) {
		dir := setup(t)

		stdout, stderr, code := runConformanceShow(t, dir, nil, conformanceSelectedID, "--field", "description")

		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		if stdout != bareValue {
			t.Errorf("stdout = %q, want %q", stdout, bareValue)
		}
	})

	t.Run("it prints the same bare bytes under every format flag", func(t *testing.T) {
		dir := setup(t)

		for _, format := range conformanceFormatSpellings {
			stdout, stderr, code := runConformanceShow(t, dir, format, conformanceSelectedID, "--field", "description")

			if code != 0 {
				t.Fatalf("exit code under %v = %d, want 0; stderr = %q", format, code, stderr)
			}
			if stdout != bareValue {
				t.Errorf("stdout under %v = %q, want %q", format, stdout, bareValue)
			}
		}
	})

	t.Run("it prints zero bytes for a selection that renders nothing", func(t *testing.T) {
		dir := setup(t)

		stdout, stderr, code := runConformanceShow(t, dir, nil, conformanceSelectedID, "--field", "parent,closed")

		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want no bytes at all", stdout)
		}
	})

	t.Run("it prints zero bytes in every format for a selection that renders nothing", func(t *testing.T) {
		dir := setup(t)

		for _, format := range conformanceFormatSpellings {
			stdout, stderr, code := runConformanceShow(t, dir, format, conformanceSelectedID, "--field", "parent,closed")

			if code != 0 {
				t.Fatalf("exit code under %v = %d, want 0; stderr = %q", format, code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout under %v = %q, want no bytes at all", format, stdout)
			}
		}
	})

	t.Run("it refuses quiet alongside a selection", func(t *testing.T) {
		dir := setup(t)

		stdout, _, code := runConformanceShow(t, dir, nil, conformanceSelectedID, "--quiet", "--field", "title")

		if code == 0 {
			t.Errorf("exit code = 0, want non-zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want no bytes at all", stdout)
		}
	})

	t.Run("it declares the bare value exempt in the inventory", func(t *testing.T) {
		entry := conformanceEntry(t, "show with a single non-list field selected")

		if entry.NotADocument == "" {
			t.Errorf("entry %q carries no exemption reason", entry.Name)
		}
	})

	t.Run("it declares the zero-byte selection exempt in the inventory", func(t *testing.T) {
		entry := conformanceEntry(t, "show with a selection naming only absent fields")

		if entry.NotADocument == "" {
			t.Errorf("entry %q carries no exemption reason", entry.Name)
		}
	})

	t.Run("it skips exempt entries in the toon driver", func(t *testing.T) {
		attempted := false
		entry := conformanceDoc{
			Name:         "exempt fixture",
			Command:      "show",
			NotADocument: "a bare value is not a document",
			Setup: func(t *testing.T) (string, []string) {
				attempted = true
				return "", nil
			},
		}

		t.Run(entry.Name, func(t *testing.T) {
			driveConformanceEntry(t, conformanceToonDriver, entry, func(t *testing.T, entry conformanceDoc) {
				runConformanceDoc(t, entry)
			})
		})

		if attempted {
			t.Error("the driver ran the setup of an entry carrying an exemption reason")
		}
	})
}

// assertConformanceKeys asserts the decoded document carries exactly the given
// keys.
func assertConformanceKeys(t *testing.T, doc map[string]any, want ...string) {
	t.Helper()
	got := slices.Sorted(maps.Keys(doc))
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("decoded keys = %v, want %v", got, want)
	}
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

// conformanceChange is one row of a decoded changed section.
type conformanceChange struct {
	From string
	To   string
	Auto bool
}

// conformanceChanges decodes the changed section, keyed by task id, failing
// when a task appears in more than one row.
func conformanceChanges(t *testing.T, doc map[string]any) map[string]conformanceChange {
	t.Helper()
	rows := toonRows(t, doc, "changed")
	changes := make(map[string]conformanceChange, len(rows))
	for _, row := range rows {
		id, ok := row["id"].(string)
		if !ok {
			t.Fatalf("changed row %#v carries no id", row)
		}
		if _, seen := changes[id]; seen {
			t.Errorf("changed carries task %q more than once", id)
		}
		auto, ok := row["auto"].(bool)
		if !ok {
			t.Fatalf("changed row %q has auto = %#v, want a bool", id, row["auto"])
		}
		changes[id] = conformanceChange{From: fmt.Sprint(row["from"]), To: fmt.Sprint(row["to"]), Auto: auto}
	}
	return changes
}

func assertConformanceChanges(t *testing.T, doc map[string]any, want map[string]conformanceChange) {
	t.Helper()
	got := conformanceChanges(t, doc)
	if len(got) != len(want) {
		t.Fatalf("changed has %d rows, want %d: %#v", len(got), len(want), got)
	}
	for id, wantChange := range want {
		gotChange, ok := got[id]
		if !ok {
			t.Errorf("changed carries no row for %q: %#v", id, got)
			continue
		}
		if gotChange != wantChange {
			t.Errorf("changed row %q = %#v, want %#v", id, gotChange, wantChange)
		}
	}
}

func TestToonStatusCommandConformance(t *testing.T) {
	t.Run("it decodes a start document with no cascade", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "start with no cascade")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-a00001": {From: "open", To: "in_progress", Auto: false},
		})
	})

	t.Run("it decodes a done document with no cascade", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "done with no cascade")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-a00001": {From: "in_progress", To: "done", Auto: false},
		})
	})

	t.Run("it decodes a cancel document with no cascade", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "cancel with no cascade")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-a00001": {From: "open", To: "cancelled", Auto: false},
		})
	})

	t.Run("it decodes a reopen document with no cascade", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "reopen with no cascade")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-a00001": {From: "done", To: "open", Auto: false},
		})
	})

	t.Run("it decodes a cascading start document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "start cascading to an open parent")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-c00001": {From: "open", To: "in_progress", Auto: false},
			"tick-p00001": {From: "open", To: "in_progress", Auto: true},
		})
	})

	t.Run("it decodes a cascading done document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "done cascading to an open child")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-p00001": {From: "open", To: "done", Auto: false},
			"tick-c00001": {From: "open", To: "done", Auto: true},
		})
	})

	t.Run("it decodes a cascading cancel document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "cancel cascading to an open child")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-p00001": {From: "open", To: "cancelled", Auto: false},
			"tick-c00001": {From: "open", To: "cancelled", Auto: true},
		})
	})

	t.Run("it decodes a cascading reopen document", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "reopen cascading to a done parent")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-c00001": {From: "done", To: "open", Auto: false},
			"tick-p00001": {From: "done", To: "open", Auto: true},
		})
	})
}

func TestToonMutationConformance(t *testing.T) {
	t.Run("it decodes a create document with no parent", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "create with no parent")

		assertToonKeysPresent(t, doc, "changed")
		assertToonRowsEmpty(t, doc, "changed")
	})

	t.Run("it decodes a create document under a done parent", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "create under a done parent")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-p00001": {From: "done", To: "open", Auto: true},
			"tick-g00001": {From: "done", To: "open", Auto: true},
		})
	})

	t.Run("it decodes an update document with no status movement", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "update with no status movement")

		assertToonRowsEmpty(t, doc, "changed")
	})

	t.Run("it decodes an update document for rule 6 alone", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "update moving a task under a done parent")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-n00001": {From: "done", To: "open", Auto: true},
		})
	})

	t.Run("it decodes an update document for rule 3 alone", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "update moving a task away from a completed parent")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-o00001": {From: "open", To: "done", Auto: true},
		})
	})

	t.Run("it decodes an update document where both rules meet on a shared ancestor", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "update where reopen and completion meet on a shared ancestor")

		assertConformanceChanges(t, doc, map[string]conformanceChange{
			"tick-n00001": {From: "done", To: "open", Auto: true},
			"tick-o00001": {From: "open", To: "done", Auto: true},
			"tick-r00001": {From: "done", To: "open", Auto: true},
		})
	})

	t.Run("it decodes a note add document with no changed section", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "note add on a task carrying a note")

		if rows := toonRows(t, doc, "notes"); len(rows) != len(conformanceNotes)+1 {
			t.Errorf("notes has %d rows, want %d", len(rows), len(conformanceNotes)+1)
		}
		assertToonKeysAbsent(t, doc, "changed")
	})

	t.Run("it decodes a note remove document with no changed section", func(t *testing.T) {
		doc := decodeConformanceEntry(t, "note remove on a task carrying a note")

		if rows := toonRows(t, doc, "notes"); len(rows) != len(conformanceNotes)-1 {
			t.Errorf("notes has %d rows, want %d", len(rows), len(conformanceNotes)-1)
		}
		assertToonKeysAbsent(t, doc, "changed")
	})
}

// conformanceProseCommands are the commands whose toon and pretty output is a
// confirmation message rather than a document. Under json each returns an
// object, which the json driver parses like any other document.
var conformanceProseCommands = []string{"dep add", "dep remove", "remove", "init", "rebuild"}

// conformanceDriverProseCommands returns the prose commands the named driver
// declares.
func conformanceDriverProseCommands(driver string) []string {
	if driver == conformanceToonDriver {
		return conformanceProseCommands
	}
	return nil
}

// conformanceOutOfScopeCommands are the commands that bypass the formatter and
// print straight to the terminal.
var conformanceOutOfScopeCommands = []string{"doctor", "migrate"}

// conformanceMustParseCommands returns the commands the named driver runs, in
// the order the inventory first names them.
func conformanceMustParseCommands(driver string) []string {
	var commands []string
	seen := make(map[string]bool, len(conformanceDocs))
	for _, entry := range conformanceDocs {
		if seen[entry.Command] || conformanceSkipReason(driver, entry) != "" {
			continue
		}
		seen[entry.Command] = true
		commands = append(commands, entry.Command)
	}
	return commands
}

// conformanceCoverageProblems reports every registered command the three sets
// fail to claim exactly once, and every declared name no command registers.
func conformanceCoverageProblems(registered CommandFlags, mustParse, prose, outOfScope []string) []string {
	var problems []string
	declared := make(map[string]int, len(registered))
	for _, set := range [][]string{mustParse, prose, outOfScope} {
		for _, name := range set {
			declared[name]++
			if _, ok := registered[name]; !ok {
				problems = append(problems, fmt.Sprintf("declared command %q is not registered in commandFlags", name))
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(registered)) {
		switch count := declared[name]; count {
		case 1:
		case 0:
			problems = append(problems, fmt.Sprintf("command %q is declared in none of the must-parse, prose and out-of-scope sets", name))
		default:
			problems = append(problems, fmt.Sprintf("command %q is declared in %d sets, want exactly one", name, count))
		}
	}
	return problems
}

// assertConformanceCoverage asserts the sets the named driver declares claim
// every registered command exactly once.
func assertConformanceCoverage(t *testing.T, driver string) {
	t.Helper()
	problems := conformanceCoverageProblems(
		commandFlags,
		conformanceMustParseCommands(driver),
		conformanceDriverProseCommands(driver),
		conformanceOutOfScopeCommands,
	)
	for _, problem := range problems {
		t.Error(problem)
	}
}

func TestConformanceInventoryCoversEveryCommand(t *testing.T) {
	t.Run("it covers every registered command under toon", func(t *testing.T) {
		assertConformanceCoverage(t, conformanceToonDriver)
	})

	t.Run("it covers every registered command under json", func(t *testing.T) {
		assertConformanceCoverage(t, conformanceJSONDriver)
	})

	t.Run("it fails when a command is declared nowhere", func(t *testing.T) {
		registered := CommandFlags{"list": {}, "ghost": {}}

		problems := conformanceCoverageProblems(registered, []string{"list"}, nil, nil)

		if len(problems) != 1 {
			t.Errorf("problems = %v, want one undeclared-command problem", problems)
		}
	})

	t.Run("it fails when a command is declared twice", func(t *testing.T) {
		registered := CommandFlags{"list": {}}

		problems := conformanceCoverageProblems(registered, []string{"list"}, []string{"list"}, nil)

		if len(problems) != 1 {
			t.Errorf("problems = %v, want one twice-declared problem", problems)
		}
	})

	t.Run("it fails when a declared command is not registered", func(t *testing.T) {
		registered := CommandFlags{"list": {}}

		problems := conformanceCoverageProblems(registered, []string{"list"}, []string{"phantom"}, nil)

		if len(problems) != 1 {
			t.Errorf("problems = %v, want one unregistered-declaration problem", problems)
		}
	})
}

func TestJSONOutputConformance(t *testing.T) {
	driveConformanceInventory(t, conformanceJSONDriver, func(t *testing.T, entry conformanceDoc) {
		runJSONConformanceDoc(t, entry)
	})
}

func TestJSONConformanceChecks(t *testing.T) {
	t.Run("it parses a lone document as one JSON value", func(t *testing.T) {
		value, err := decodeSingleJSONValue("lone", "{\n  \"id\": \"tick-a00001\"\n}\n")

		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if _, ok := value.(map[string]any); !ok {
			t.Errorf("decoded value is %T, want map[string]any", value)
		}
	})

	t.Run("it fails when a document is followed by a second value", func(t *testing.T) {
		stream := "{\"id\": \"tick-a00001\"}\n{\"id\": \"tick-b00002\"}\n"

		_, err := decodeSingleJSONValue("two documents", stream)

		if err == nil {
			t.Fatal("decode of a two-document stream succeeded, want an error")
		}
		if !strings.Contains(err.Error(), "two documents") {
			t.Errorf("error %q does not name the entry", err)
		}
		if !strings.Contains(err.Error(), stream) {
			t.Errorf("error %q does not carry the stream text", err)
		}
	})

	t.Run("it expects task list documents to be arrays", func(t *testing.T) {
		for _, command := range []string{"list", "ready", "blocked"} {
			if !jsonTopLevelIsArray(command) {
				t.Errorf("jsonTopLevelIsArray(%q) = false, want true", command)
			}
			if problem := jsonShapeProblem("fixture", command, []any{}); problem != "" {
				t.Errorf("an array document for %q reported %q", command, problem)
			}
			if jsonShapeProblem("fixture", command, map[string]any{}) == "" {
				t.Errorf("an object document for %q reported no problem", command)
			}
		}
	})

	t.Run("it expects every other document to be an object", func(t *testing.T) {
		for _, command := range []string{"show", "stats", "dep tree", "create", "update", "note add"} {
			if jsonTopLevelIsArray(command) {
				t.Errorf("jsonTopLevelIsArray(%q) = true, want false", command)
			}
			if problem := jsonShapeProblem("fixture", command, map[string]any{}); problem != "" {
				t.Errorf("an object document for %q reported %q", command, problem)
			}
			if jsonShapeProblem("fixture", command, []any{}) == "" {
				t.Errorf("an array document for %q reported no problem", command)
			}
		}
	})

	t.Run("it accepts a document meeting every invariant", func(t *testing.T) {
		doc := map[string]any{
			"changed":  []any{map[string]any{"id": "tick-a00001", "auto": true}},
			"tags":     []any{},
			"children": []any{map[string]any{"children": []any{}}},
			"notes":    []any{map[string]any{"index": float64(1)}},
		}

		if problems := jsonInvariantProblems("fixture", doc); len(problems) != 0 {
			t.Errorf("problems = %v, want none", problems)
		}
	})

	t.Run("it rejects a null list value", func(t *testing.T) {
		for _, key := range jsonConformanceListKeys {
			doc := map[string]any{key: nil}

			if problems := jsonInvariantProblems("fixture", doc); len(problems) != 1 {
				t.Errorf("problems for a null %q = %v, want one list problem", key, problems)
			}
		}
	})

	t.Run("it requires auto to be a boolean", func(t *testing.T) {
		doc := map[string]any{"changed": []any{map[string]any{"auto": "true"}}}

		if problems := jsonInvariantProblems("fixture", doc); len(problems) != 1 {
			t.Errorf("problems = %v, want one auto problem", problems)
		}
	})

	t.Run("it requires index to be a number", func(t *testing.T) {
		doc := map[string]any{"notes": []any{map[string]any{"index": "1"}}}

		if problems := jsonInvariantProblems("fixture", doc); len(problems) != 1 {
			t.Errorf("problems = %v, want one index problem", problems)
		}
	})

	t.Run("it skips a toon-exempt entry in the toon driver and runs it under json", func(t *testing.T) {
		var toonExempt, sharedByBoth []string
		for _, entry := range conformanceDocs {
			switch {
			case entry.NotADocument != "":
			case entry.ToonNotADocument != "":
				toonExempt = append(toonExempt, entry.Name)
			default:
				sharedByBoth = append(sharedByBoth, entry.Name)
			}
		}
		if len(toonExempt) == 0 {
			t.Fatal("no inventory entry carries a toon exemption")
		}

		toonDriven := drivenConformanceEntries(t, conformanceToonDriver)
		jsonDriven := drivenConformanceEntries(t, conformanceJSONDriver)

		if !slices.Equal(toonDriven, sharedByBoth) {
			t.Errorf("toon driver ran %v, want %v", toonDriven, sharedByBoth)
		}
		if !slices.Equal(jsonDriven, slices.Concat(sharedByBoth, toonExempt)) {
			t.Errorf("json driver ran %v, want every entry carrying a setup", jsonDriven)
		}
	})
}

func TestJSONTaskListConformance(t *testing.T) {
	t.Run("it parses task list documents as arrays", func(t *testing.T) {
		names := []string{
			"list on a populated project",
			"ready with results",
			"blocked with results",
		}

		for _, name := range names {
			value := runJSONConformanceDoc(t, conformanceEntry(t, name))

			items, ok := value.([]any)
			if !ok {
				t.Errorf("%s: decoded value is %T, want []any", name, value)
				continue
			}
			if len(items) == 0 {
				t.Errorf("%s: decoded array is empty", name)
			}
		}
	})
}

func TestJSONRemovalConformance(t *testing.T) {
	t.Run("it carries a one-element removed list and an empty deps_updated list", func(t *testing.T) {
		doc := decodeJSONConformanceEntry(t, "remove on a task nothing depends on")

		rows := toonRows(t, doc, "removed")
		if len(rows) != 1 {
			t.Fatalf("removed has %d rows, want 1", len(rows))
		}
		if id := rows[0]["id"]; id != "tick-bbb222" {
			t.Errorf("removed[0][\"id\"] = %#v, want \"tick-bbb222\"", id)
		}
		depsUpdated, ok := doc["deps_updated"].([]any)
		if !ok {
			t.Fatalf("deps_updated = %#v, want a list", doc["deps_updated"])
		}
		if len(depsUpdated) != 0 {
			t.Errorf("deps_updated = %v, want an empty list", depsUpdated)
		}
	})
}

func TestJSONDetailConformance(t *testing.T) {
	t.Run("it carries json's always-present detail keys", func(t *testing.T) {
		doc := decodeJSONConformanceEntry(t, "show on a task carrying no optional field")

		assertToonKeysPresent(t, doc, "type", "tags", "refs", "description")
		assertToonFields(t, doc, map[string]any{
			"id":          "tick-d44444",
			"type":        "",
			"description": "",
		})
	})

	t.Run("it omits parent and closed from a bare task's json detail", func(t *testing.T) {
		doc := decodeJSONConformanceEntry(t, "show on a task carrying no optional field")

		assertToonKeysAbsent(t, doc, "parent", "closed")
	})

	t.Run("it carries exactly the selected keys in a filtered json document", func(t *testing.T) {
		doc := decodeJSONConformanceEntry(t, "show with a multi-field selection")

		assertConformanceKeys(t, doc, "description", "notes")
	})
}

func TestJSONStatsConformance(t *testing.T) {
	t.Run("it keeps the nested stats shape in json", func(t *testing.T) {
		doc := decodeJSONConformanceEntry(t, "stats on a populated project")

		assertConformanceKeys(t, doc, "total", "by_status", "workflow", "by_priority")
		assertToonFields(t, doc, map[string]any{"total": float64(5)})
	})
}
