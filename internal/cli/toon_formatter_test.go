package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestToonFormatter(t *testing.T) {
	// Compile-time interface verification.
	var _ Formatter = (*ToonFormatter)(nil)

	t.Run("it formats list with correct header count and schema", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup Sanctum", Status: task.StatusDone, Priority: 1, Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %q", len(lines), result)
		}
		expectedHeader := "tasks[2]{id,title,status,priority,type}:"
		if lines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", lines[0], expectedHeader)
		}
		expectedRow1 := `  tick-a1b2,Setup Sanctum,done,1,""`
		if lines[1] != expectedRow1 {
			t.Errorf("row 1 = %q, want %q", lines[1], expectedRow1)
		}
		expectedRow2 := `  tick-c3d4,Login endpoint,open,1,""`
		if lines[2] != expectedRow2 {
			t.Errorf("row 2 = %q, want %q", lines[2], expectedRow2)
		}
	})

	t.Run("it formats zero tasks as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskList([]task.Task{})
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats zero tasks from nil slice as empty section", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTaskList(nil)
		expected := "tasks[0]{id,title,status,priority,type}:"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats show with all sections", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		updated := time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "Setup Sanctum",
				Status:      task.StatusInProgress,
				Priority:    1,
				Description: "Full task description here.\nCan be multiple lines.",
				Parent:      "tick-e5f6",
				Created:     now,
				Updated:     updated,
			},
			BlockedBy: []RelatedTask{
				{ID: "tick-c3d4", Title: "Database migrations", Status: "done"},
				{ID: "tick-g7h8", Title: "Config setup", Status: "in_progress"},
			},
			Children:    []RelatedTask{},
			ParentTitle: "Auth System",
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 4 {
			t.Fatalf("expected 4 sections, got %d: %q", len(sections), result)
		}
		// Section 1: task
		taskLines := strings.Split(sections[0], "\n")
		expectedTaskHeader := "task{id,title,status,priority,parent,created,updated}:"
		if taskLines[0] != expectedTaskHeader {
			t.Errorf("task header = %q, want %q", taskLines[0], expectedTaskHeader)
		}
		expectedTaskRow := `  tick-a1b2,Setup Sanctum,in_progress,1,tick-e5f6,"2026-01-19T10:00:00Z","2026-01-19T14:30:00Z"`
		if taskLines[1] != expectedTaskRow {
			t.Errorf("task row = %q, want %q", taskLines[1], expectedTaskRow)
		}
		// Section 2: blocked_by
		blockedLines := strings.Split(sections[1], "\n")
		expectedBlockedHeader := "blocked_by[2]{id,title,status}:"
		if blockedLines[0] != expectedBlockedHeader {
			t.Errorf("blocked_by header = %q, want %q", blockedLines[0], expectedBlockedHeader)
		}
		// Section 3: children
		expectedChildren := "children[0]{id,title,status}:"
		if sections[2] != expectedChildren {
			t.Errorf("children section = %q, want %q", sections[2], expectedChildren)
		}
		// Section 4: description
		descLines := strings.Split(sections[3], "\n")
		if descLines[0] != "description:" {
			t.Errorf("description header = %q, want %q", descLines[0], "description:")
		}
		if descLines[1] != "  Full task description here." {
			t.Errorf("description line 1 = %q, want %q", descLines[1], "  Full task description here.")
		}
		if descLines[2] != "  Can be multiple lines." {
			t.Errorf("description line 2 = %q, want %q", descLines[2], "  Can be multiple lines.")
		}
	})

	t.Run("it omits parent and closed from schema when null", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Simple task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		taskLines := strings.Split(sections[0], "\n")
		expectedHeader := "task{id,title,status,priority,created,updated}:"
		if taskLines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", taskLines[0], expectedHeader)
		}
		expectedRow := `  tick-a1b2,Simple task,open,2,"2026-01-19T10:00:00Z","2026-01-19T10:00:00Z"`
		if taskLines[1] != expectedRow {
			t.Errorf("row = %q, want %q", taskLines[1], expectedRow)
		}
	})

	t.Run("it renders blocked_by and children with count 0 when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "No deps",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if !strings.Contains(result, "blocked_by[0]{id,title,status}:") {
			t.Errorf("missing blocked_by[0] section in: %q", result)
		}
		if !strings.Contains(result, "children[0]{id,title,status}:") {
			t.Errorf("missing children[0] section in: %q", result)
		}
	})

	t.Run("it omits description section when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Simple task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		if strings.Contains(result, "description:") {
			t.Errorf("description section should be omitted when empty, got: %q", result)
		}
		// Should have exactly 3 sections (task, blocked_by, children)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 3 {
			t.Errorf("expected 3 sections (no description), got %d: %q", len(sections), result)
		}
	})

	t.Run("it renders multiline description as indented lines", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:          "tick-a1b2",
				Title:       "With description",
				Status:      task.StatusOpen,
				Priority:    2,
				Description: "Line one.\nLine two.\nLine three.",
				Created:     now,
				Updated:     now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		descSection := sections[len(sections)-1]
		descLines := strings.Split(descSection, "\n")
		if descLines[0] != "description:" {
			t.Errorf("description header = %q, want %q", descLines[0], "description:")
		}
		if len(descLines) != 4 {
			t.Fatalf("expected 4 description lines (header + 3), got %d: %q", len(descLines), descSection)
		}
		if descLines[1] != "  Line one." {
			t.Errorf("line 1 = %q, want %q", descLines[1], "  Line one.")
		}
		if descLines[2] != "  Line two." {
			t.Errorf("line 2 = %q, want %q", descLines[2], "  Line two.")
		}
		if descLines[3] != "  Line three." {
			t.Errorf("line 3 = %q, want %q", descLines[3], "  Line three.")
		}
	})

	t.Run("it escapes commas in titles", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Setup, Deploy", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		expectedRow := `  tick-a1b2,"Setup, Deploy",open,1,""`
		if lines[1] != expectedRow {
			t.Errorf("row = %q, want %q", lines[1], expectedRow)
		}
	})

	t.Run("it formats stats with all counts", func(t *testing.T) {
		f := &ToonFormatter{}
		stats := Stats{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		result := f.FormatStats(stats)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 2 {
			t.Fatalf("expected 2 sections, got %d: %q", len(sections), result)
		}
		// Section 1: stats summary
		summaryLines := strings.Split(sections[0], "\n")
		expectedHeader := "stats{total,open,in_progress,done,cancelled,ready,blocked}:"
		if summaryLines[0] != expectedHeader {
			t.Errorf("stats header = %q, want %q", summaryLines[0], expectedHeader)
		}
		expectedRow := "  47,12,3,28,4,8,4"
		if summaryLines[1] != expectedRow {
			t.Errorf("stats row = %q, want %q", summaryLines[1], expectedRow)
		}
	})

	t.Run("it formats by_priority with 5 rows including zeros", func(t *testing.T) {
		f := &ToonFormatter{}
		stats := Stats{
			Total:      10,
			Open:       10,
			ByPriority: [5]int{0, 5, 3, 0, 2},
		}
		result := f.FormatStats(stats)
		sections := strings.Split(result, "\n\n")
		if len(sections) != 2 {
			t.Fatalf("expected 2 sections, got %d: %q", len(sections), result)
		}
		priorityLines := strings.Split(sections[1], "\n")
		expectedPriorityHeader := "by_priority[5]{priority,count}:"
		if priorityLines[0] != expectedPriorityHeader {
			t.Errorf("by_priority header = %q, want %q", priorityLines[0], expectedPriorityHeader)
		}
		if len(priorityLines) != 6 {
			t.Fatalf("expected 6 lines (header + 5 rows), got %d: %q", len(priorityLines), sections[1])
		}
		expectedRows := []string{
			"  0,0",
			"  1,5",
			"  2,3",
			"  3,0",
			"  4,2",
		}
		for i, expected := range expectedRows {
			if priorityLines[i+1] != expected {
				t.Errorf("priority row %d = %q, want %q", i, priorityLines[i+1], expected)
			}
		}
	})

	t.Run("it formats transition as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatTransition("tick-a3f2b7", "open", "in_progress")
		expected := "tick-a3f2b7: open \u2192 in_progress"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats dep change as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
		resultAdd := f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")
		expectedAdd := "Dependency added: tick-c3d4 blocked by tick-a1b2"
		if resultAdd != expectedAdd {
			t.Errorf("add result = %q, want %q", resultAdd, expectedAdd)
		}
		resultRm := f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")
		expectedRm := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2"
		if resultRm != expectedRm {
			t.Errorf("rm result = %q, want %q", resultRm, expectedRm)
		}
	})

	t.Run("it formats message as plain text", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatMessage("Tick initialized in /path/to/project")
		expected := "Tick initialized in /path/to/project"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it includes closed in show schema when present", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Done task",
				Status:   task.StatusDone,
				Priority: 1,
				Created:  now,
				Updated:  now,
				Closed:   &closed,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		taskLines := strings.Split(sections[0], "\n")
		expectedHeader := "task{id,title,status,priority,created,updated,closed}:"
		if taskLines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", taskLines[0], expectedHeader)
		}
		expectedRow := `  tick-a1b2,Done task,done,1,"2026-01-19T10:00:00Z","2026-01-19T10:00:00Z","2026-01-19T16:00:00Z"`
		if taskLines[1] != expectedRow {
			t.Errorf("row = %q, want %q", taskLines[1], expectedRow)
		}
	})

	t.Run("it formats single task removal via baseFormatter", func(t *testing.T) {
		f := &ToonFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
		})
		expected := `Removed tick-a1b2 "My task"`
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it includes type in toon list rows", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-a1b2", Title: "Fix login bug", Status: task.StatusOpen, Priority: 1, Type: "bug", Created: now, Updated: now},
			{ID: "tick-c3d4", Title: "Add search", Status: task.StatusDone, Priority: 2, Type: "feature", Created: now, Updated: now},
		}
		result := f.FormatTaskList(tasks)
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %q", len(lines), result)
		}
		// Header should include type in schema
		expectedHeader := "tasks[2]{id,title,status,priority,type}:"
		if lines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", lines[0], expectedHeader)
		}
		// Rows should include type value
		expectedRow1 := "  tick-a1b2,Fix login bug,open,1,bug"
		if lines[1] != expectedRow1 {
			t.Errorf("row 1 = %q, want %q", lines[1], expectedRow1)
		}
		expectedRow2 := "  tick-c3d4,Add search,done,2,feature"
		if lines[2] != expectedRow2 {
			t.Errorf("row 2 = %q, want %q", lines[2], expectedRow2)
		}
	})

	t.Run("it includes type in toon show when set", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Fix login bug",
				Status:   task.StatusOpen,
				Priority: 1,
				Type:     "bug",
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		taskLines := strings.Split(sections[0], "\n")
		// Header should include type in schema
		expectedHeader := "task{id,title,status,priority,type,created,updated}:"
		if taskLines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", taskLines[0], expectedHeader)
		}
		// Row should include type value
		expectedRow := `  tick-a1b2,Fix login bug,open,1,bug,"2026-01-19T10:00:00Z","2026-01-19T10:00:00Z"`
		if taskLines[1] != expectedRow {
			t.Errorf("row = %q, want %q", taskLines[1], expectedRow)
		}
	})

	t.Run("it omits type from toon show when empty", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Simple task",
				Status:   task.StatusOpen,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
			BlockedBy: []RelatedTask{},
			Children:  []RelatedTask{},
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		taskLines := strings.Split(sections[0], "\n")
		// Header should NOT include type when empty
		expectedHeader := "task{id,title,status,priority,created,updated}:"
		if taskLines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", taskLines[0], expectedHeader)
		}
		// Row should not contain type field
		if strings.Contains(taskLines[1], ",type") || strings.Count(taskLines[1], ",") > 5 {
			t.Errorf("row should not contain type when empty: %q", taskLines[1])
		}
	})

	t.Run("it includes both parent and closed in show schema when both present", func(t *testing.T) {
		f := &ToonFormatter{}
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		closed := time.Date(2026, 1, 19, 16, 0, 0, 0, time.UTC)
		detail := TaskDetail{
			Task: task.Task{
				ID:       "tick-a1b2",
				Title:    "Done child",
				Status:   task.StatusDone,
				Priority: 1,
				Parent:   "tick-e5f6",
				Created:  now,
				Updated:  now,
				Closed:   &closed,
			},
			BlockedBy:   []RelatedTask{},
			Children:    []RelatedTask{},
			ParentTitle: "Parent task",
		}
		result := f.FormatTaskDetail(detail)
		sections := strings.Split(result, "\n\n")
		taskLines := strings.Split(sections[0], "\n")
		expectedHeader := "task{id,title,status,priority,parent,created,updated,closed}:"
		if taskLines[0] != expectedHeader {
			t.Errorf("header = %q, want %q", taskLines[0], expectedHeader)
		}
	})
}
