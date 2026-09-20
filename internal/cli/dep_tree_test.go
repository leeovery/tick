package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestDepTreeWiring(t *testing.T) {
	t.Run("it qualifies dep tree as a two-level command", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree", "tick-abc123"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 1 || rest[0] != "tick-abc123" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-abc123]", rest)
		}
	})

	t.Run("it qualifies dep tree with no args", func(t *testing.T) {
		cmd, rest := qualifyCommand("dep", []string{"tree"})
		if cmd != "dep tree" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep tree")
		}
		if len(rest) != 0 {
			t.Errorf("qualifyCommand returned rest = %v, want empty", rest)
		}
	})

	t.Run("it rejects unknown flag on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--unknown"}, commandFlags)
		if err == nil {
			t.Fatal("expected error for --unknown on dep tree, got nil")
		}
		want := `unknown flag "--unknown" for "dep tree". Run 'tick help dep' for usage.`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("it accepts global flags on dep tree", func(t *testing.T) {
		err := ValidateFlags("dep tree", []string{"--quiet"}, commandFlags)
		if err != nil {
			t.Errorf("expected nil for global flag on dep tree, got %v", err)
		}
	})

	t.Run("it dispatches dep tree without error", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "dep", "tree"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
	})

	t.Run("it shows tree in dep help text", func(t *testing.T) {
		stdout, _, code := runHelp(t, "help", "dep")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout, "tree") {
			t.Errorf("dep help should mention 'tree', got %q", stdout)
		}
	})

	t.Run("it does not qualify tree under note", func(t *testing.T) {
		cmd, rest := qualifyCommand("note", []string{"tree"})
		if cmd != "note" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "note")
		}
		if len(rest) != 1 || rest[0] != "tree" {
			t.Errorf("qualifyCommand returned rest = %v, want [tree]", rest)
		}
	})

	t.Run("it preserves args when tree is not qualified under note", func(t *testing.T) {
		cmd, rest := qualifyCommand("note", []string{"tree", "--foo"})
		if cmd != "note" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "note")
		}
		if len(rest) != 2 || rest[0] != "tree" || rest[1] != "--foo" {
			t.Errorf("qualifyCommand returned rest = %v, want [tree --foo]", rest)
		}
	})

	t.Run("it still qualifies add under note", func(t *testing.T) {
		cmd, rest := qualifyCommand("note", []string{"add", "tick-aaa", "hello"})
		if cmd != "note add" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "note add")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "hello" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa hello]", rest)
		}
	})

	t.Run("it still qualifies remove under note", func(t *testing.T) {
		cmd, rest := qualifyCommand("note", []string{"remove", "tick-aaa", "1"})
		if cmd != "note remove" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "note remove")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "1" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa 1]", rest)
		}
	})

	t.Run("it does not break existing dep add/remove dispatch", func(t *testing.T) {
		// Verify qualifyCommand still works for add and remove
		cmd, rest := qualifyCommand("dep", []string{"add", "tick-aaa", "tick-bbb"})
		if cmd != "dep add" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep add")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}

		cmd, rest = qualifyCommand("dep", []string{"remove", "tick-aaa", "tick-bbb"})
		if cmd != "dep remove" {
			t.Errorf("qualifyCommand returned cmd = %q, want %q", cmd, "dep remove")
		}
		if len(rest) != 2 || rest[0] != "tick-aaa" || rest[1] != "tick-bbb" {
			t.Errorf("qualifyCommand returned rest = %v, want [tick-aaa tick-bbb]", rest)
		}
	})
}

// runDepTree runs the tick dep tree command with the given args and returns stdout, stderr, and exit code.
func runDepTree(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", "--pretty", "dep", "tree"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

// runDepTreeJSON runs dep tree with --json and returns the unmarshalled document.
func runDepTreeJSON(t *testing.T, dir string, args ...string) map[string]any {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	fullArgs := append([]string{"tick", "--json", "dep", "tree"}, args...)
	if code := app.Run(fullArgs); code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderrBuf.String())
	}

	output := strings.TrimSpace(stdoutBuf.String())
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}
	return parsed
}

// jsonDepTreeOnlyTree returns the single tree of a full-graph dep tree JSON document.
func jsonDepTreeOnlyTree(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	trees, ok := doc["trees"].([]any)
	if !ok || len(trees) != 1 {
		t.Fatalf("trees = %#v, want one node", doc["trees"])
	}
	node, ok := trees[0].(map[string]any)
	if !ok {
		t.Fatalf("tree = %#v, want an object", trees[0])
	}
	return node
}

// jsonDepTreeOnlyChild returns the single child of a dep tree JSON node.
func jsonDepTreeOnlyChild(t *testing.T, node map[string]any) map[string]any {
	t.Helper()
	children, ok := node["children"].([]any)
	if !ok || len(children) != 1 {
		t.Fatalf("children = %#v, want one node", node["children"])
	}
	child, ok := children[0].(map[string]any)
	if !ok {
		t.Fatalf("child = %#v, want an object", children[0])
	}
	return child
}

// assertJSONDepTreeTask asserts the task fields carried by a dep tree JSON node.
func assertJSONDepTreeTask(t *testing.T, node map[string]any, id string, title string, status string) {
	t.Helper()
	got, ok := node["task"].(map[string]any)
	if !ok {
		t.Fatalf("task = %#v, want an object", node["task"])
	}
	want := map[string]any{"id": id, "title": title, "status": status}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("task %s = %v, want %q", key, got[key], wantValue)
		}
	}
}

// cycleTasks returns two tasks that block each other, so no participant is a root.
func cycleTasks(now time.Time) []task.Task {
	return []task.Task{
		{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
		{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
	}
}

// danglingBlockerTasks returns a single task blocked by an ID no task record matches.
func danglingBlockerTasks(now time.Time) []task.Task {
	return []task.Task{
		{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-ghost1"}, Created: now, Updated: now},
	}
}

// chainTasks returns a linear A -> B -> C chain rooted at an unblocked task.
func chainTasks(now time.Time) []task.Task {
	return []task.Task{
		{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		{ID: "tick-ccc333", Title: "Task C", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now.Add(2 * time.Second), Updated: now.Add(2 * time.Second)},
	}
}

// unconnectedTasks returns two tasks that carry no dependencies in either direction.
func unconnectedTasks(now time.Time) []task.Task {
	return []task.Task{
		{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
	}
}

func TestRunDepTree(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	t.Run("it still prints the no-dependencies sentence in pretty for an empty project", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "No dependencies found.\n" {
			t.Errorf("stdout = %q, want %q", stdout, "No dependencies found.\n")
		}
	})

	t.Run("it still prints the no-dependencies sentence in pretty when no task has dependencies", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, unconnectedTasks(now))

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "No dependencies found.\n" {
			t.Errorf("stdout = %q, want %q", stdout, "No dependencies found.\n")
		}
	})

	t.Run("it returns the emptied document for an empty project", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))

		assertToonRowsEmpty(t, doc, "dep_tree")
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(0),
			"blocked": float64(0),
		})
	})

	t.Run("it returns the emptied document when no task has dependencies", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, unconnectedTasks(now))

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))

		assertToonRowsEmpty(t, doc, "dep_tree")
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(0),
			"longest": float64(0),
			"blocked": float64(0),
		})
	})

	t.Run("it emits the cycle's edges beside its counts", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, cycleTasks(now))

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))

		assertToonEdgeRows(t, doc, "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-bbb222", To: "tick-aaa111"},
		})
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(1),
			"longest": float64(2),
			"blocked": float64(2),
		})
	})

	t.Run("it emits an edge from a blocker no task carries", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, danglingBlockerTasks(now))

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))

		assertToonEdgeRows(t, doc, "dep_tree", []toonEdgeRow{
			{From: "tick-ghost1", To: "tick-aaa111"},
		})
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(1),
			"longest": float64(1),
			"blocked": float64(1),
		})
	})

	t.Run("it publishes every full-graph tree under trees", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, chainTasks(now))

		doc := runDepTreeJSON(t, dir)

		if _, exists := doc["roots"]; exists {
			t.Errorf("roots key should be absent, got %#v", doc["roots"])
		}
		tree := jsonDepTreeOnlyTree(t, doc)
		assertJSONDepTreeTask(t, tree, "tick-aaa111", "Task A", "open")
		child := jsonDepTreeOnlyChild(t, tree)
		assertJSONDepTreeTask(t, child, "tick-bbb222", "Task B", "open")
		assertJSONDepTreeTask(t, jsonDepTreeOnlyChild(t, child), "tick-ccc333", "Task C", "open")
	})

	t.Run("it emits trees as an empty list when no task has dependencies", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, unconnectedTasks(now))

		doc := runDepTreeJSON(t, dir)

		trees, ok := doc["trees"].([]any)
		if !ok {
			t.Fatalf("trees should be array (not null), got %T: %v", doc["trees"], doc["trees"])
		}
		if len(trees) != 0 {
			t.Errorf("trees should be empty, got %d items", len(trees))
		}
		for key, want := range map[string]float64{"chains": 0, "longest": 0, "blocked": 0} {
			if got := doc[key]; got != want {
				t.Errorf("%s = %v, want %v", key, got, want)
			}
		}
	})

	t.Run("it carries a cycle's participants under trees", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, cycleTasks(now))

		doc := runDepTreeJSON(t, dir)

		tree := jsonDepTreeOnlyTree(t, doc)
		assertJSONDepTreeTask(t, tree, "tick-aaa111", "Task A", "open")
		child := jsonDepTreeOnlyChild(t, tree)
		assertJSONDepTreeTask(t, child, "tick-bbb222", "Task B", "open")
		assertJSONDepTreeTask(t, jsonDepTreeOnlyChild(t, child), "tick-aaa111", "Task A", "open")

		for key, want := range map[string]float64{"chains": 1, "longest": 2, "blocked": 2} {
			if got := doc[key]; got != want {
				t.Errorf("%s = %v, want %v", key, got, want)
			}
		}
	})

	t.Run("it carries a bare id for a blocker no task carries in JSON", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, danglingBlockerTasks(now))

		doc := runDepTreeJSON(t, dir)

		tree := jsonDepTreeOnlyTree(t, doc)
		assertJSONDepTreeTask(t, tree, "tick-ghost1", "", "")
		assertJSONDepTreeTask(t, jsonDepTreeOnlyChild(t, tree), "tick-aaa111", "Task A", "open")
	})

	t.Run("it renders a cycle in the terminal", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, cycleTasks(now))

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		want := "" +
			"tick-aaa111  Task A (open)\n" +
			"└── tick-bbb222  Task B (open)\n" +
			"    └── tick-aaa111  Task A (open)\n" +
			"\n" +
			"1 chain, longest: 2, 2 blocked\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it renders a dangling blocker in the terminal", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, danglingBlockerTasks(now))

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		want := "" +
			"tick-ghost1   ()\n" +
			"└── tick-aaa111  Task A (open)\n" +
			"\n" +
			"1 chain, longest: 1, 1 blocked\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it leaves rooted terminal output unchanged", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, chainTasks(now))

		stdout, stderr, exitCode := runDepTree(t, dir)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		want := "" +
			"tick-aaa111  Task A (open)\n" +
			"└── tick-bbb222  Task B (open)\n" +
			"    └── tick-ccc333  Task C (open)\n" +
			"\n" +
			"1 chain, longest: 2, 2 blocked\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("it still renders the populated graph", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, chainTasks(now))

		doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))

		assertToonEdgeRows(t, doc, "dep_tree", []toonEdgeRow{
			{From: "tick-aaa111", To: "tick-bbb222"},
			{From: "tick-bbb222", To: "tick-ccc333"},
		})
		assertToonFields(t, doc, map[string]any{
			"chains":  float64(1),
			"longest": float64(2),
			"blocked": float64(2),
		})
	})

	t.Run("it outputs no dependencies for isolated task in focused mode", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir, "tick-aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := stdout
		// Should show the task itself: ID, title, and status
		if !strings.Contains(output, "tick-aaa111") {
			t.Error("output should contain task ID")
		}
		if !strings.Contains(output, "Task A") {
			t.Error("output should contain task title")
		}
		if !strings.Contains(output, "open") {
			t.Error("output should contain task status")
		}
		// Should show "No dependencies." message
		if !strings.Contains(output, "No dependencies.") {
			t.Errorf("output should contain 'No dependencies.', got %q", output)
		}
	})

	t.Run("it returns error for nonexistent task ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runDepTree(t, dir, "tick-zzz999")
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for nonexistent task ID")
		}
		if !strings.Contains(stderr, "not found") {
			t.Errorf("stderr should contain 'not found', got %q", stderr)
		}
	})

	t.Run("it resolves partial task ID", func(t *testing.T) {
		// Isolated task — focused mode with no deps shows task info directly
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		// Use partial ID "aaa111" (6 hex chars without tick- prefix) — should resolve to tick-aaa111
		stdout, stderr, exitCode := runDepTree(t, dir, "aaa111")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		output := stdout
		// Isolated task outputs task ID + title + status directly via handler
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("partial ID should resolve to tick-aaa111; output = %q", output)
		}
		if !strings.Contains(output, "Task A") {
			t.Errorf("output should contain task title; output = %q", output)
		}
	})

	t.Run("it suppresses output in quiet mode", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		stdout, stderr, exitCode := runDepTree(t, dir, "--quiet")
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		if stdout != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout)
		}
	})

	t.Run("it emits both directions on a focused document with no dependencies", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		parsed := runDepTreeJSON(t, dir, "tick-aaa111")

		if parsed["mode"] != "focused" {
			t.Errorf("mode = %v, want %q", parsed["mode"], "focused")
		}
		if _, exists := parsed["message"]; exists {
			t.Errorf("message key should be absent, got %v", parsed["message"])
		}

		target, ok := parsed["target"].(map[string]any)
		if !ok {
			t.Fatalf("target should be object, got %T: %v", parsed["target"], parsed["target"])
		}
		if target["id"] != "tick-aaa111" {
			t.Errorf("target.id = %v, want %q", target["id"], "tick-aaa111")
		}
		if target["title"] != "Task A" {
			t.Errorf("target.title = %v, want %q", target["title"], "Task A")
		}
		if target["status"] != "open" {
			t.Errorf("target.status = %v, want %q", target["status"], "open")
		}

		for _, key := range []string{"blocked_by", "blocks"} {
			nodes, ok := parsed[key].([]any)
			if !ok {
				t.Fatalf("%s should be array (not null), got %T: %v", key, parsed[key], parsed[key])
			}
			if len(nodes) != 0 {
				t.Errorf("%s should be empty, got %d items", key, len(nodes))
			}
		}
	})

	t.Run("it emits the emptied full document instead of a message", func(t *testing.T) {
		dir, _ := setupTickProjectWithTasks(t, unconnectedTasks(now))

		parsed := runDepTreeJSON(t, dir)

		if parsed["mode"] != "full" {
			t.Errorf("mode = %v, want %q", parsed["mode"], "full")
		}
		if _, exists := parsed["message"]; exists {
			t.Errorf("message key should be absent, got %v", parsed["message"])
		}

		trees, ok := parsed["trees"].([]any)
		if !ok {
			t.Fatalf("trees should be array (not null), got %T: %v", parsed["trees"], parsed["trees"])
		}
		if len(trees) != 0 {
			t.Errorf("trees should be empty, got %d items", len(trees))
		}

		for _, key := range []string{"chains", "longest", "blocked"} {
			if parsed[key] != float64(0) {
				t.Errorf("%s = %v, want 0", key, parsed[key])
			}
		}
	})

	t.Run("it returns error for ambiguous partial ID", func(t *testing.T) {
		// Two tasks with IDs that share a prefix: tick-aaa111 and tick-aaa222
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A1", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-aaa222", Title: "Task A2", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		_, stderr, exitCode := runDepTree(t, dir, "aaa")
		if exitCode == 0 {
			t.Fatal("expected non-zero exit code for ambiguous partial ID")
		}
		if !strings.Contains(stderr, "ambiguous") {
			t.Errorf("stderr should contain 'ambiguous', got %q", stderr)
		}
	})

	t.Run("it handles focused view via full App.Run dispatch", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--pretty", "dep", "tree", "tick-aaa111"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}
	})
}
