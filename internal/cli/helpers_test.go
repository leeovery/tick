package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestParseCommaSeparatedIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"single ID",
			"tick-aaa111",
			[]string{"tick-aaa111"},
		},
		{
			"multiple IDs",
			"tick-aaa111,tick-bbb222,tick-ccc333",
			[]string{"tick-aaa111", "tick-bbb222", "tick-ccc333"},
		},
		{
			"whitespace around IDs",
			"tick-aaa111 , tick-bbb222 , tick-ccc333",
			[]string{"tick-aaa111", "tick-bbb222", "tick-ccc333"},
		},
		{
			"empty string",
			"",
			nil,
		},
		{
			"only commas and whitespace",
			" , , ",
			nil,
		},
		{
			"normalizes to lowercase",
			"TICK-AAA111,Tick-BBB222",
			[]string{"tick-aaa111", "tick-bbb222"},
		},
		{
			"filters empty segments from trailing comma",
			"tick-aaa111,",
			[]string{"tick-aaa111"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommaSeparatedIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseCommaSeparatedIDs(%q) returned %d items %v, want %d items %v",
					tt.input, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCommaSeparatedIDs(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestApplyBlocks(t *testing.T) {
	t.Run("it appends sourceID to matching tasks BlockedBy", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 || tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001]", tasks[0].BlockedBy)
		}
		// Unmatched task should not be modified
		if len(tasks[1].BlockedBy) != 0 {
			t.Errorf("tasks[1].BlockedBy = %v, want empty", tasks[1].BlockedBy)
		}
	})

	t.Run("it sets Updated timestamp on modified tasks", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if !tasks[0].Updated.Equal(applyNow) {
			t.Errorf("tasks[0].Updated = %v, want %v", tasks[0].Updated, applyNow)
		}
		// Unmatched task should retain original Updated
		if !tasks[1].Updated.Equal(now) {
			t.Errorf("tasks[1].Updated = %v, want %v (unchanged)", tasks[1].Updated, now)
		}
	})

	t.Run("it is a no-op with non-existent blockIDs", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-nonexist"}, applyNow)

		if len(tasks[0].BlockedBy) != 0 {
			t.Errorf("tasks[0].BlockedBy = %v, want empty", tasks[0].BlockedBy)
		}
		if !tasks[0].Updated.Equal(now) {
			t.Errorf("tasks[0].Updated = %v, want %v (unchanged)", tasks[0].Updated, now)
		}
	})

	t.Run("it skips duplicate when sourceID already in BlockedBy", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2,
				BlockedBy: []string{"tick-src001"}, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001] (no duplicate)", tasks[0].BlockedBy)
		}
		if tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy[0] = %q, want %q", tasks[0].BlockedBy[0], "tick-src001")
		}
		// Updated should NOT be changed since no new dependency was added
		if !tasks[0].Updated.Equal(now) {
			t.Errorf("tasks[0].Updated = %v, want %v (unchanged)", tasks[0].Updated, now)
		}
	})

	t.Run("it matches blockIDs case-insensitively", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"TICK-AAA111"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 || tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001]", tasks[0].BlockedBy)
		}
	})

	t.Run("it detects existing dep case-insensitively", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2,
				BlockedBy: []string{"TICK-SRC001"}, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 {
			t.Errorf("tasks[0].BlockedBy = %v, want [TICK-SRC001] (no duplicate)", tasks[0].BlockedBy)
		}
	})

	t.Run("it handles multiple blockIDs", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111", "tick-ccc333"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 || tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001]", tasks[0].BlockedBy)
		}
		if len(tasks[1].BlockedBy) != 0 {
			t.Errorf("tasks[1].BlockedBy = %v, want empty", tasks[1].BlockedBy)
		}
		if len(tasks[2].BlockedBy) != 1 || tasks[2].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[2].BlockedBy = %v, want [tick-src001]", tasks[2].BlockedBy)
		}
	})
}

func TestOutputMutationResult(t *testing.T) {
	t.Run("it outputs only the ID in quiet mode", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore error: %v", err)
		}
		defer store.Close()

		var buf strings.Builder
		fc := FormatConfig{Quiet: true}
		fmtr := &PrettyFormatter{}

		err = outputMutationResult(store, "tick-aaa111", fc, fmtr, &buf, nil)
		if err != nil {
			t.Fatalf("outputMutationResult error: %v", err)
		}

		expected := "tick-aaa111\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it outputs full task detail in non-quiet mode", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Test task", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir, _ := setupTickProjectWithTasks(t, tasks)

		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore error: %v", err)
		}
		defer store.Close()

		var buf strings.Builder
		fc := FormatConfig{Quiet: false}
		fmtr := &PrettyFormatter{}

		err = outputMutationResult(store, "tick-aaa111", fc, fmtr, &buf, nil)
		if err != nil {
			t.Fatalf("outputMutationResult error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "tick-aaa111") {
			t.Errorf("output should contain task ID, got %q", output)
		}
		if !strings.Contains(output, "Test task") {
			t.Errorf("output should contain task title, got %q", output)
		}
		if !strings.Contains(output, "ID:") {
			t.Errorf("output should contain 'ID:' field, got %q", output)
		}
		if !strings.Contains(output, "Status:") {
			t.Errorf("output should contain 'Status:' field, got %q", output)
		}
	})

	t.Run("it returns error for non-existent task ID", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore error: %v", err)
		}
		defer store.Close()

		var buf strings.Builder
		fc := FormatConfig{Quiet: false}
		fmtr := &PrettyFormatter{}

		err = outputMutationResult(store, "tick-nonexist", fc, fmtr, &buf, nil)
		if err == nil {
			t.Fatal("expected error for non-existent task ID")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want to contain 'not found'", err.Error())
		}
	})
}

func TestOutputStatusChanges(t *testing.T) {
	now := time.Now()

	t.Run("it writes the toon changed table with a trailing newline", func(t *testing.T) {
		var buf strings.Builder

		outputStatusChanges(&buf, &ToonFormatter{}, CascadeResult{
			Changed: []StatusChange{
				{ID: "tick-abc123", Title: "My Task", From: "open", To: "in_progress"},
			},
		})

		expected := "changed[1]{id,title,from,to,auto}:\n  tick-abc123,My Task,open,in_progress,false\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("it writes the pretty cascade tree", func(t *testing.T) {
		var buf strings.Builder

		parent := task.Task{ID: "tick-parent1", Title: "Parent", Status: task.StatusCancelled, Created: now, Updated: now}
		child := task.Task{ID: "tick-child1", Title: "Child", Status: task.StatusCancelled, Parent: "tick-parent1", Created: now, Updated: now}
		tasks := []task.Task{parent, child}
		result := task.TransitionResult{OldStatus: task.StatusInProgress, NewStatus: task.StatusCancelled}
		cascades := []task.CascadeChange{
			{Task: &tasks[1], Action: "cancel", OldStatus: task.StatusOpen, NewStatus: task.StatusCancelled},
		}

		outputStatusChanges(&buf, &PrettyFormatter{}, buildCascadeResult("tick-parent1", "Parent", result, cascades, tasks, false))

		expected := "tick-parent1: in_progress \u2192 cancelled\n\nCascaded:\n\u2514\u2500 tick-child1 \"Child\": open \u2192 cancelled\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}

func TestValidateAndReopenParent(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	t.Run("it returns no-op when parent is open", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		_, _, reopened, err := validateAndReopenParent(tasks, "tick-parent1", &sm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reopened {
			t.Error("reopened should be false for open parent")
		}
	})

	t.Run("it returns no-op when parent is in_progress", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusInProgress, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		_, _, reopened, err := validateAndReopenParent(tasks, "tick-parent1", &sm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reopened {
			t.Error("reopened should be false for in_progress parent")
		}
	})

	t.Run("it returns error when parent is cancelled (Rule 7)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusCancelled, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		_, _, _, err := validateAndReopenParent(tasks, "tick-parent1", &sm)
		if err == nil {
			t.Fatal("expected error for cancelled parent")
		}
		if !strings.Contains(err.Error(), "cancelled") {
			t.Errorf("error = %q, want it to mention cancelled", err.Error())
		}
	})

	t.Run("it reopens done parent (Rule 6)", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusDone, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		result, _, reopened, err := validateAndReopenParent(tasks, "tick-parent1", &sm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reopened {
			t.Error("reopened should be true for done parent")
		}
		if result.OldStatus != task.StatusDone {
			t.Errorf("result.OldStatus = %v, want done", result.OldStatus)
		}
		if result.NewStatus != task.StatusOpen {
			t.Errorf("result.NewStatus = %v, want open", result.NewStatus)
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("tasks[0].Status = %v, want open after reopen", tasks[0].Status)
		}
	})

	t.Run("it finds parent by normalized ID", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusDone, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		_, _, reopened, err := validateAndReopenParent(tasks, "TICK-PARENT1", &sm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reopened {
			t.Error("reopened should be true")
		}
	})

	t.Run("it returns no-op when parent ID not found in tasks", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusDone, Priority: 2, Created: now, Updated: now},
		}
		var sm task.StateMachine

		_, _, reopened, err := validateAndReopenParent(tasks, "tick-nonexist", &sm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reopened {
			t.Error("reopened should be false when parent not found")
		}
	})
}

func TestOpenStore(t *testing.T) {
	t.Run("it returns a valid store for a valid tick directory", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		store, err := openStore(dir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore returned error: %v", err)
		}
		defer store.Close()

		if store == nil {
			t.Fatal("openStore returned nil store")
		}
	})

	t.Run("it returns error when no tick directory exists", func(t *testing.T) {
		dir := t.TempDir()

		store, err := openStore(dir, FormatConfig{})
		if err == nil {
			defer store.Close()
			t.Fatal("openStore should return error for missing .tick directory")
		}

		if !strings.Contains(err.Error(), "no .tick directory found") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "no .tick directory found")
		}
	})

	t.Run("it discovers tick directory from subdirectory", func(t *testing.T) {
		dir, _ := setupTickProject(t)

		// Create a subdirectory
		subDir := filepath.Join(dir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		store, err := openStore(subDir, FormatConfig{})
		if err != nil {
			t.Fatalf("openStore returned error: %v", err)
		}
		defer store.Close()

		if store == nil {
			t.Fatal("openStore returned nil store")
		}
	})
}
