package beads

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/migrate"
	"github.com/leeovery/tick/internal/task"
)

// helper creates a temp dir with optional .beads/issues.jsonl content.
func setupBeadsDir(t *testing.T, content string) string {
	t.Helper()
	baseDir := t.TempDir()
	beadsDir := filepath.Join(baseDir, ".beads")
	if err := os.Mkdir(beadsDir, 0o755); err != nil {
		t.Fatalf("failed to create .beads dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "issues.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write issues.jsonl: %v", err)
	}
	return baseDir
}

func TestBeadsProvider(t *testing.T) {
	t.Run("Name returns beads", func(t *testing.T) {
		p := NewBeadsProvider("/tmp/claude/fake")
		if p.Name() != "beads" {
			t.Errorf("Name() = %q, want %q", p.Name(), "beads")
		}
	})

	t.Run("BeadsProvider implements Provider interface", func(t *testing.T) {
		var _ migrate.Provider = NewBeadsProvider("/tmp/claude/fake")
	})

	t.Run("Tasks reads valid JSONL and returns MigratedTasks", func(t *testing.T) {
		content := `{"id":"b-001","title":"Implement login","description":"Login flow","status":"pending","priority":2,"issue_type":"task","created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-12T14:00:00Z","closed_at":"","close_reason":"","created_by":"alice","dependencies":[]}
{"id":"b-002","title":"Fix database","description":"DB connection fix","status":"closed","priority":1,"issue_type":"epic","created_at":"2026-01-05T08:00:00Z","updated_at":"2026-01-06T12:00:00Z","closed_at":"2026-01-07T10:00:00Z","close_reason":"completed","created_by":"bob","dependencies":["b-001"]}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0].Title != "Implement login" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Implement login")
		}
		if tasks[1].Title != "Fix database" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Fix database")
		}
	})

	t.Run("Tasks maps beads pending status to tick open", func(t *testing.T) {
		content := `{"id":"b-001","title":"Pending task","status":"pending","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusOpen {
			t.Errorf("Status = %q, want %q", tasks[0].Status, task.StatusOpen)
		}
	})

	t.Run("Tasks maps beads in_progress status to tick in_progress", func(t *testing.T) {
		content := `{"id":"b-001","title":"Active task","status":"in_progress","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusInProgress {
			t.Errorf("Status = %q, want %q", tasks[0].Status, task.StatusInProgress)
		}
	})

	t.Run("Tasks maps beads closed status to tick done", func(t *testing.T) {
		content := `{"id":"b-001","title":"Done task","status":"closed","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != task.StatusDone {
			t.Errorf("Status = %q, want %q", tasks[0].Status, task.StatusDone)
		}
	})

	t.Run("Tasks maps unknown status to empty string", func(t *testing.T) {
		content := `{"id":"b-001","title":"Unknown status task","status":"wontfix","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Status != "" {
			t.Errorf("Status = %q, want empty", tasks[0].Status)
		}
	})

	t.Run("Tasks maps beads priority values directly 0-3", func(t *testing.T) {
		tests := []struct {
			name     string
			priority int
		}{
			{"priority 0", 0},
			{"priority 1", 1},
			{"priority 2", 2},
			{"priority 3", 3},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				content := `{"id":"b-001","title":"Priority task","status":"pending","priority":` +
					string(rune('0'+tt.priority)) + `}`
				baseDir := setupBeadsDir(t, content)
				p := NewBeadsProvider(baseDir)

				tasks, err := p.Tasks()
				if err != nil {
					t.Fatalf("Tasks() returned error: %v", err)
				}
				if len(tasks) != 1 {
					t.Fatalf("expected 1 task, got %d", len(tasks))
				}
				if tasks[0].Priority == nil {
					t.Fatal("expected Priority to be non-nil")
				}
				if *tasks[0].Priority != tt.priority {
					t.Errorf("Priority = %d, want %d", *tasks[0].Priority, tt.priority)
				}
			})
		}
	})

	t.Run("Tasks produces nil Priority when JSON line omits priority field", func(t *testing.T) {
		content := `{"id":"b-001","title":"No priority task","status":"pending"}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Priority != nil {
			t.Errorf("Priority = %v, want nil (absent priority should not default to 0)", *tasks[0].Priority)
		}
	})

	t.Run("Tasks produces non-nil Priority pointing to 0 when JSON has priority 0", func(t *testing.T) {
		content := `{"id":"b-001","title":"Zero priority task","status":"pending","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Priority == nil {
			t.Fatal("expected Priority to be non-nil for explicit priority 0")
		}
		if *tasks[0].Priority != 0 {
			t.Errorf("Priority = %d, want 0", *tasks[0].Priority)
		}
	})

	t.Run("Tasks parses ISO 8601 timestamps into time.Time", func(t *testing.T) {
		content := `{"id":"b-001","title":"Timestamped task","status":"pending","priority":0,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-12T14:00:00Z"}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		wantCreated := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
		wantUpdated := time.Date(2026, 1, 12, 14, 0, 0, 0, time.UTC)
		if !tasks[0].Created.Equal(wantCreated) {
			t.Errorf("Created = %v, want %v", tasks[0].Created, wantCreated)
		}
		if !tasks[0].Updated.Equal(wantUpdated) {
			t.Errorf("Updated = %v, want %v", tasks[0].Updated, wantUpdated)
		}
	})

	t.Run("Tasks returns error when .beads directory is missing", func(t *testing.T) {
		baseDir := t.TempDir() // no .beads dir
		p := NewBeadsProvider(baseDir)

		_, err := p.Tasks()
		if err == nil {
			t.Fatal("expected error for missing .beads directory, got nil")
		}
		want := ".beads directory not found in " + baseDir
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Tasks returns error when issues.jsonl is missing", func(t *testing.T) {
		baseDir := t.TempDir()
		beadsDir := filepath.Join(baseDir, ".beads")
		if err := os.Mkdir(beadsDir, 0o755); err != nil {
			t.Fatalf("failed to create .beads dir: %v", err)
		}
		p := NewBeadsProvider(baseDir)

		_, err := p.Tasks()
		if err == nil {
			t.Fatal("expected error for missing issues.jsonl, got nil")
		}
		want := "issues.jsonl not found in " + beadsDir
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Tasks returns empty slice and nil error for empty file", func(t *testing.T) {
		baseDir := setupBeadsDir(t, "")
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("Tasks returns empty slice and nil error for file with only blank lines", func(t *testing.T) {
		baseDir := setupBeadsDir(t, "\n\n  \n\t\n")
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("Tasks returns malformed JSON lines as sentinel MigratedTask entries", func(t *testing.T) {
		content := `{"id":"b-001","title":"Good task","status":"pending","priority":0}
not valid json at all
{"id":"b-002","title":"Another good task","status":"closed","priority":1}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks (including malformed entry), got %d", len(tasks))
		}
		if tasks[0].Title != "Good task" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Good task")
		}
		if tasks[1].Title != "(malformed entry)" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "(malformed entry)")
		}
		if tasks[2].Title != "Another good task" {
			t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "Another good task")
		}
	})

	t.Run("Tasks returns entries with empty title for engine validation", func(t *testing.T) {
		content := `{"id":"b-001","title":"","status":"pending","priority":0}
{"id":"b-002","title":"Valid task","status":"pending","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (including empty title), got %d", len(tasks))
		}
		if tasks[0].Title != "" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "")
		}
		if tasks[1].Title != "Valid task" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Valid task")
		}
	})

	t.Run("Tasks returns entries with whitespace-only title for engine validation", func(t *testing.T) {
		content := `{"id":"b-001","title":"   \t  ","status":"pending","priority":0}
{"id":"b-002","title":"Valid task","status":"pending","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks (including whitespace title), got %d", len(tasks))
		}
		if tasks[0].Title != "   \t  " {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "   \t  ")
		}
		if tasks[1].Title != "Valid task" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Valid task")
		}
	})

	t.Run("Tasks returns entries with invalid priority for engine validation", func(t *testing.T) {
		content := `{"id":"b-001","title":"Good task","status":"pending","priority":2}
{"id":"b-002","title":"Bad priority","status":"pending","priority":99}
{"id":"b-003","title":"Also good","status":"pending","priority":1}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks (including invalid priority), got %d", len(tasks))
		}
		if tasks[0].Title != "Good task" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Good task")
		}
		if tasks[1].Title != "Bad priority" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Bad priority")
		}
		if tasks[1].Priority == nil || *tasks[1].Priority != 99 {
			t.Errorf("tasks[1].Priority = %v, want 99", tasks[1].Priority)
		}
		if tasks[2].Title != "Also good" {
			t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "Also good")
		}
	})

	t.Run("Tasks returns all entries from mixed valid invalid and malformed JSONL", func(t *testing.T) {
		content := `{"id":"b-001","title":"Valid task","status":"pending","priority":2}
{"id":"b-002","title":"","status":"pending","priority":0}
not valid json
{"id":"b-003","title":"Bad priority","status":"pending","priority":99}
{"id":"b-004","title":"Another valid","status":"closed","priority":1}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 5 {
			t.Fatalf("expected 5 tasks (all entries), got %d", len(tasks))
		}
		if tasks[0].Title != "Valid task" {
			t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Valid task")
		}
		if tasks[1].Title != "" {
			t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "")
		}
		if tasks[2].Title != "(malformed entry)" {
			t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "(malformed entry)")
		}
		if tasks[3].Title != "Bad priority" {
			t.Errorf("tasks[3].Title = %q, want %q", tasks[3].Title, "Bad priority")
		}
		if tasks[4].Title != "Another valid" {
			t.Errorf("tasks[4].Title = %q, want %q", tasks[4].Title, "Another valid")
		}
	})

	t.Run("Tasks discards id issue_type close_reason created_by dependencies fields", func(t *testing.T) {
		content := `{"id":"b-001","title":"Preserved task","description":"kept","status":"pending","priority":1,"issue_type":"epic","close_reason":"completed","created_by":"alice","dependencies":["b-002"]}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		// MigratedTask has no fields for id, issue_type, close_reason, created_by, or dependencies.
		// We verify the kept fields are correct, which implicitly proves discarded fields didn't interfere.
		tk := tasks[0]
		if tk.Title != "Preserved task" {
			t.Errorf("Title = %q, want %q", tk.Title, "Preserved task")
		}
		if tk.Description != "kept" {
			t.Errorf("Description = %q, want %q", tk.Description, "kept")
		}
		if tk.Status != task.StatusOpen {
			t.Errorf("Status = %q, want %q", tk.Status, task.StatusOpen)
		}
		if tk.Priority == nil || *tk.Priority != 1 {
			t.Errorf("Priority = %v, want 1", tk.Priority)
		}
	})

	t.Run("Tasks maps description field to MigratedTask Description", func(t *testing.T) {
		content := `{"id":"b-001","title":"With description","description":"A detailed description of the task","status":"pending","priority":0}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Description != "A detailed description of the task" {
			t.Errorf("Description = %q, want %q", tasks[0].Description, "A detailed description of the task")
		}
	})

	t.Run("Tasks handles closed_at timestamp for closed tasks", func(t *testing.T) {
		content := `{"id":"b-001","title":"Closed task","status":"closed","priority":0,"created_at":"2026-01-05T08:00:00Z","updated_at":"2026-01-06T12:00:00Z","closed_at":"2026-01-07T10:00:00Z"}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		wantClosed := time.Date(2026, 1, 7, 10, 0, 0, 0, time.UTC)
		if !tasks[0].Closed.Equal(wantClosed) {
			t.Errorf("Closed = %v, want %v", tasks[0].Closed, wantClosed)
		}
	})

	t.Run("Tasks leaves timestamp fields as zero when parsing fails", func(t *testing.T) {
		content := `{"id":"b-001","title":"Bad timestamps","status":"pending","priority":0,"created_at":"not-a-date","updated_at":"also-bad","closed_at":"nope"}`
		baseDir := setupBeadsDir(t, content)
		p := NewBeadsProvider(baseDir)

		tasks, err := p.Tasks()
		if err != nil {
			t.Fatalf("Tasks() returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if !tasks[0].Created.IsZero() {
			t.Errorf("Created should be zero, got %v", tasks[0].Created)
		}
		if !tasks[0].Updated.IsZero() {
			t.Errorf("Updated should be zero, got %v", tasks[0].Updated)
		}
		if !tasks[0].Closed.IsZero() {
			t.Errorf("Closed should be zero, got %v", tasks[0].Closed)
		}
	})
}

func intPtr(v int) *int { return &v }

func TestMapToMigratedTask(t *testing.T) {
	t.Run("mapToMigratedTask produces valid MigratedTask from fully populated beadsIssue", func(t *testing.T) {
		issue := beadsIssue{
			ID:           "b-001",
			Title:        "Implement login flow",
			Description:  "Full markdown description",
			Status:       "closed",
			Priority:     intPtr(3),
			IssueType:    "epic",
			CreatedAt:    "2026-01-10T09:00:00Z",
			UpdatedAt:    "2026-01-12T14:00:00Z",
			ClosedAt:     "2026-01-15T10:00:00Z",
			CloseReason:  "completed",
			CreatedBy:    "alice",
			Dependencies: []interface{}{"b-002", "b-003"},
		}

		tk := mapToMigratedTask(issue)

		if tk.Title != "Implement login flow" {
			t.Errorf("Title = %q, want %q", tk.Title, "Implement login flow")
		}
		if tk.Description != "Full markdown description" {
			t.Errorf("Description = %q, want %q", tk.Description, "Full markdown description")
		}
		if tk.Status != task.StatusDone {
			t.Errorf("Status = %q, want %q", tk.Status, task.StatusDone)
		}
		if tk.Priority == nil || *tk.Priority != 3 {
			t.Errorf("Priority = %v, want 3", tk.Priority)
		}

		wantCreated := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
		if !tk.Created.Equal(wantCreated) {
			t.Errorf("Created = %v, want %v", tk.Created, wantCreated)
		}
		wantUpdated := time.Date(2026, 1, 12, 14, 0, 0, 0, time.UTC)
		if !tk.Updated.Equal(wantUpdated) {
			t.Errorf("Updated = %v, want %v", tk.Updated, wantUpdated)
		}
		wantClosed := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		if !tk.Closed.Equal(wantClosed) {
			t.Errorf("Closed = %v, want %v", tk.Closed, wantClosed)
		}

		// Verify it passes validation.
		if err := tk.Validate(); err != nil {
			t.Errorf("expected valid MigratedTask, got validation error: %v", err)
		}
	})

	t.Run("mapToMigratedTask returns MigratedTask with empty title without error", func(t *testing.T) {
		issue := beadsIssue{
			ID:       "b-001",
			Title:    "",
			Status:   "pending",
			Priority: intPtr(0),
		}

		tk := mapToMigratedTask(issue)

		if tk.Title != "" {
			t.Errorf("Title = %q, want empty string", tk.Title)
		}
		if tk.Status != task.StatusOpen {
			t.Errorf("Status = %q, want %q", tk.Status, task.StatusOpen)
		}
	})
}
