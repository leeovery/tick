package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/migrate"
	"github.com/leeovery/tick/internal/migrate/beads"
	"github.com/leeovery/tick/internal/task"
)

func TestNewMigrateProvider(t *testing.T) {
	t.Run("registry returns BeadsProvider for name beads", func(t *testing.T) {
		provider, err := newMigrateProvider("beads", "/tmp/claude/fake")
		if err != nil {
			t.Fatalf("newMigrateProvider(beads) returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.Name() != "beads" {
			t.Errorf("provider.Name() = %q, want %q", provider.Name(), "beads")
		}
		if _, ok := provider.(*beads.BeadsProvider); !ok {
			t.Errorf("expected *beads.BeadsProvider, got %T", provider)
		}
	})

	t.Run("NewProvider returns UnknownProviderError for unrecognized name", func(t *testing.T) {
		_, err := newMigrateProvider("jira", "/tmp/claude/fake")
		if err == nil {
			t.Fatal("expected error for unknown provider, got nil")
		}
		var upe *migrate.UnknownProviderError
		if !errors.As(err, &upe) {
			t.Fatalf("expected *migrate.UnknownProviderError, got %T: %v", err, err)
		}
		if upe.Name != "jira" {
			t.Errorf("UnknownProviderError.Name = %q, want %q", upe.Name, "jira")
		}
		if len(upe.Available) == 0 {
			t.Fatal("UnknownProviderError.Available should not be empty")
		}
	})

	t.Run("NewProvider still returns BeadsProvider for name beads (regression)", func(t *testing.T) {
		provider, err := newMigrateProvider("beads", "/tmp/claude/fake")
		if err != nil {
			t.Fatalf("newMigrateProvider(beads) returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.Name() != "beads" {
			t.Errorf("provider.Name() = %q, want %q", provider.Name(), "beads")
		}
	})

	t.Run("AvailableProviders returns sorted list of registered provider names", func(t *testing.T) {
		providers := availableProviders()
		if len(providers) == 0 {
			t.Fatal("availableProviders() returned empty list")
		}
		// Must contain "beads"
		found := false
		for _, p := range providers {
			if p == "beads" {
				found = true
			}
		}
		if !found {
			t.Errorf("availableProviders() = %v, want to contain %q", providers, "beads")
		}
		// Must be sorted
		for i := 1; i < len(providers); i++ {
			if providers[i] < providers[i-1] {
				t.Errorf("availableProviders() not sorted: %v", providers)
				break
			}
		}
	})
}

// runMigrate runs tick migrate with the given args and returns stdout, stderr, exit code.
func runMigrate(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
		IsTTY:  true,
	}
	fullArgs := append([]string{"tick", "migrate"}, args...)
	code := app.Run(fullArgs)
	return stdoutBuf.String(), stderrBuf.String(), code
}

func TestMigrateCommand(t *testing.T) {
	t.Run("migrate command requires --from flag", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runMigrate(t, dir)

		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "--from") {
			t.Errorf("stderr = %q, want to contain --from", stderr)
		}
	})

	t.Run("migrate command with empty --from value returns error", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runMigrate(t, dir, "--from", "")

		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, "--from") {
			t.Errorf("stderr = %q, want to contain --from", stderr)
		}
	})

	t.Run("CLI prints Error Unknown provider followed by available providers to stderr", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		_, stderr, exitCode := runMigrate(t, dir, "--from", "xyz")

		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		wantFirst := "Error: Unknown provider \"xyz\""
		if !strings.Contains(stderr, wantFirst) {
			t.Errorf("stderr missing first line, got:\n%s", stderr)
		}
		wantHeader := "Available providers:"
		if !strings.Contains(stderr, wantHeader) {
			t.Errorf("stderr missing Available providers header, got:\n%s", stderr)
		}
		wantProvider := "  - beads"
		if !strings.Contains(stderr, wantProvider) {
			t.Errorf("stderr missing provider listing, got:\n%s", stderr)
		}
	})

	t.Run("migrate command with --from beads resolves beads provider", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		// Create .beads/issues.jsonl with valid data
		setupBeadsFixture(t, dir, `{"id":"b-001","title":"Test task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Importing from beads...") {
			t.Errorf("stdout missing header, got %q", stdout)
		}
		if !strings.Contains(stdout, "Test task") {
			t.Errorf("stdout missing task title, got %q", stdout)
		}
	})

	t.Run("migrate command exits 0 when some tasks fail validation but others succeed", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		// One valid task, one with invalid status (will fail engine validation)
		content := `{"id":"b-001","title":"Good task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`
		setupBeadsFixture(t, dir, content)

		_, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
	})

	t.Run("migrate output omits Failures section when all tasks succeed", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		content := `{"id":"b-001","title":"Task A","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Task B","status":"pending","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Should NOT contain Failures: section
		if strings.Contains(stdout, "Failures:") {
			t.Errorf("stdout should not contain Failures: section when all tasks succeed, got:\n%s", stdout)
		}
	})

	t.Run("migrate command exits 1 when provider cannot be read", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		// No .beads directory → provider.Tasks() will fail

		_, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(stderr, ".beads") {
			t.Errorf("stderr = %q, want to contain .beads error", stderr)
		}
	})

	t.Run("end-to-end: migrate --from beads reads tasks, inserts, and prints output", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		content := `{"id":"b-001","title":"Implement login flow","description":"Login desc","status":"pending","priority":2,"issue_type":"task","created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-12T14:00:00Z","closed_at":"","close_reason":"","created_by":"alice","dependencies":[]}
{"id":"b-002","title":"Fix database connection","description":"DB fix","status":"closed","priority":1,"issue_type":"epic","created_at":"2026-01-05T08:00:00Z","updated_at":"2026-01-06T12:00:00Z","closed_at":"2026-01-07T10:00:00Z","close_reason":"completed","created_by":"bob","dependencies":["b-001"]}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}

		// Verify output format
		if !strings.Contains(stdout, "Importing from beads...") {
			t.Errorf("stdout missing header, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Implement login flow") {
			t.Errorf("stdout missing first task, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Fix database connection") {
			t.Errorf("stdout missing second task, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Done: 2 imported, 0 failed") {
			t.Errorf("stdout missing summary, got:\n%s", stdout)
		}

		// Verify tasks were persisted to tick store
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 persisted tasks, got %d", len(tasks))
		}

		// Find the task by title to verify fields
		var loginTask, dbTask task.Task
		for _, tk := range tasks {
			switch tk.Title {
			case "Implement login flow":
				loginTask = tk
			case "Fix database connection":
				dbTask = tk
			}
		}

		if loginTask.Title == "" {
			t.Fatal("login task not found in persisted tasks")
		}
		if loginTask.Status != task.StatusOpen {
			t.Errorf("login task status = %q, want %q", loginTask.Status, task.StatusOpen)
		}
		if loginTask.Priority != 2 {
			t.Errorf("login task priority = %d, want 2", loginTask.Priority)
		}
		if loginTask.Description != "Login desc" {
			t.Errorf("login task description = %q, want %q", loginTask.Description, "Login desc")
		}

		if dbTask.Title == "" {
			t.Fatal("db task not found in persisted tasks")
		}
		if dbTask.Status != task.StatusDone {
			t.Errorf("db task status = %q, want %q", dbTask.Status, task.StatusDone)
		}
		if dbTask.Priority != 1 {
			t.Errorf("db task priority = %d, want 1", dbTask.Priority)
		}
		if dbTask.Closed == nil {
			t.Error("db task should have closed timestamp")
		}
	})

	t.Run("migrate command prints header and summary for zero tasks", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		// Empty issues.jsonl
		setupBeadsFixture(t, dir, "")

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Importing from beads...") {
			t.Errorf("stdout missing header, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Done: 0 imported, 0 failed") {
			t.Errorf("stdout missing summary, got:\n%s", stdout)
		}
	})

	t.Run("migrate command exits 1 when tick not initialized", func(t *testing.T) {
		// Directory without .tick/
		dir := t.TempDir()
		setupBeadsFixture(t, dir, `{"id":"b-001","title":"Test","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`)

		_, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if stderr == "" {
			t.Error("expected error message on stderr")
		}
	})

	t.Run("migrate shows failures for invalid entries from beads provider", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		content := `{"id":"b-001","title":"Good task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"","status":"pending","priority":0}
not valid json at all
{"id":"b-003","title":"Bad priority task","status":"pending","priority":99}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Should show 1 imported, 3 failed
		if !strings.Contains(stdout, "Done: 1 imported, 3 failed") {
			t.Errorf("expected 1 imported, 3 failed in summary, got:\n%s", stdout)
		}
		// Should contain Failures: section
		if !strings.Contains(stdout, "Failures:") {
			t.Errorf("expected Failures: section in output, got:\n%s", stdout)
		}
		// Should contain title validation error for empty title
		if !strings.Contains(stdout, "title is required") {
			t.Errorf("expected title validation error in output, got:\n%s", stdout)
		}
		// Should contain priority validation error
		if !strings.Contains(stdout, "priority must be") {
			t.Errorf("expected priority validation error in output, got:\n%s", stdout)
		}
		// Should contain the malformed entry sentinel
		if !strings.Contains(stdout, "(malformed entry)") {
			t.Errorf("expected (malformed entry) in output, got:\n%s", stdout)
		}
	})
}

func TestMigrateDryRun(t *testing.T) {
	t.Run("--dry-run flag defaults to false", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		setupBeadsFixture(t, dir, `{"id":"b-001","title":"Persisted task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Without --dry-run, header should NOT contain [dry-run]
		if strings.Contains(stdout, "[dry-run]") {
			t.Errorf("stdout should not contain [dry-run] by default, got:\n%s", stdout)
		}
		// Tasks should be persisted
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 persisted task, got %d", len(tasks))
		}
	})

	t.Run("non-dry-run execution still uses StoreTaskCreator", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		setupBeadsFixture(t, dir, `{"id":"b-001","title":"Stored task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`)

		_, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 persisted task, got %d", len(tasks))
		}
		if tasks[0].Title != "Stored task" {
			t.Errorf("task title = %q, want %q", tasks[0].Title, "Stored task")
		}
	})

	t.Run("dry-run with zero tasks prints header with [dry-run] and summary with zero counts", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		setupBeadsFixture(t, dir, "")

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads", "--dry-run")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Importing from beads... [dry-run]") {
			t.Errorf("stdout missing dry-run header, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Done: 0 imported, 0 failed") {
			t.Errorf("stdout missing zero-count summary, got:\n%s", stdout)
		}
	})

	t.Run("dry-run with multiple tasks shows all as successful", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		content := `{"id":"b-001","title":"Task Alpha","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Task Beta","status":"pending","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-003","title":"Task Gamma","status":"closed","priority":3,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z","closed_at":"2026-01-11T10:00:00Z","close_reason":"completed"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads", "--dry-run")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Importing from beads... [dry-run]") {
			t.Errorf("stdout missing dry-run header, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "\u2713 Task: Task Alpha") {
			t.Errorf("stdout missing Task Alpha checkmark, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "\u2713 Task: Task Beta") {
			t.Errorf("stdout missing Task Beta checkmark, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "\u2713 Task: Task Gamma") {
			t.Errorf("stdout missing Task Gamma checkmark, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Done: 3 imported, 0 failed") {
			t.Errorf("stdout missing summary, got:\n%s", stdout)
		}

		// Verify NO tasks were persisted
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 persisted tasks in dry-run, got %d", len(tasks))
		}
	})

	t.Run("dry-run summary shows correct imported count matching number of valid tasks", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		content := `{"id":"b-001","title":"Valid task A","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Valid task B","status":"pending","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-003","title":"Valid task C","status":"closed","priority":3,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z","closed_at":"2026-01-11T10:00:00Z"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads", "--dry-run")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Done: 3 imported, 0 failed") {
			t.Errorf("expected 3 imported, 0 failed in dry-run summary, got:\n%s", stdout)
		}
	})
}

func TestMigratePendingOnly(t *testing.T) {
	t.Run("--pending-only flag defaults to false", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		// Include a closed task (beads "closed" maps to tick "done")
		content := `{"id":"b-001","title":"Open task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Done task","status":"closed","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z","closed_at":"2026-01-11T10:00:00Z","close_reason":"completed"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Without --pending-only, both tasks should be imported
		if !strings.Contains(stdout, "Done: 2 imported, 0 failed") {
			t.Errorf("expected 2 imported without --pending-only, got:\n%s", stdout)
		}
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 persisted tasks, got %d", len(tasks))
		}
	})

	t.Run("--pending-only flag is accepted by the migrate command", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		content := `{"id":"b-001","title":"Open task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Done task","status":"closed","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z","closed_at":"2026-01-11T10:00:00Z","close_reason":"completed"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads", "--pending-only")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Only the open task should be imported (done task filtered out)
		if !strings.Contains(stdout, "Done: 1 imported, 0 failed") {
			t.Errorf("expected 1 imported with --pending-only, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Open task") {
			t.Errorf("expected Open task in output, got:\n%s", stdout)
		}
	})

	t.Run("--pending-only combined with --dry-run filters then previews without writing", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		content := `{"id":"b-001","title":"Active task","status":"pending","priority":2,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}
{"id":"b-002","title":"Closed task","status":"closed","priority":1,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z","closed_at":"2026-01-11T10:00:00Z","close_reason":"completed"}
{"id":"b-003","title":"WIP task","status":"in_progress","priority":3,"created_at":"2026-01-10T09:00:00Z","updated_at":"2026-01-10T09:00:00Z"}`
		setupBeadsFixture(t, dir, content)

		stdout, stderr, exitCode := runMigrate(t, dir, "--from", "beads", "--pending-only", "--dry-run")

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr)
		}
		// Should show dry-run header
		if !strings.Contains(stdout, "[dry-run]") {
			t.Errorf("expected [dry-run] in output, got:\n%s", stdout)
		}
		// Should only show 2 tasks (pending ones), not the closed one
		if !strings.Contains(stdout, "Done: 2 imported, 0 failed") {
			t.Errorf("expected 2 imported with --pending-only --dry-run, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Active task") {
			t.Errorf("expected Active task in output, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "WIP task") {
			t.Errorf("expected WIP task in output, got:\n%s", stdout)
		}
		// Verify NO tasks persisted (dry-run)
		tasks := readPersistedTasks(t, tickDir)
		if len(tasks) != 0 {
			t.Errorf("expected 0 persisted tasks in dry-run, got %d", len(tasks))
		}
	})
}

// stubProvider is a test Provider that returns preconfigured tasks.
type stubProvider struct {
	name  string
	tasks []migrate.MigratedTask
	err   error
}

func (s *stubProvider) Name() string                           { return s.name }
func (s *stubProvider) Tasks() ([]migrate.MigratedTask, error) { return s.tasks, s.err }

func TestRunMigrateFailureDetail(t *testing.T) {
	t.Run("output includes Failures detail section when tasks fail validation", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		provider := &stubProvider{
			name: "test",
			tasks: []migrate.MigratedTask{
				{Title: "Good task", Status: task.StatusOpen},
				{Title: "", Status: task.StatusOpen},                                   // empty title — will fail validation
				{Title: "Bad priority", Status: task.StatusOpen, Priority: intPtr(99)}, // invalid priority
			},
		}

		var buf bytes.Buffer
		err := RunMigrate(dir, provider, true, false, &buf)

		if err != nil {
			t.Fatalf("RunMigrate returned error: %v", err)
		}

		stdout := buf.String()

		// Should contain cross-mark lines for failed tasks
		if !strings.Contains(stdout, "\u2717") {
			t.Errorf("stdout missing cross-mark for failed task, got:\n%s", stdout)
		}
		// Should contain Failures: detail section
		if !strings.Contains(stdout, "Failures:\n") {
			t.Errorf("stdout missing Failures: detail section, got:\n%s", stdout)
		}
		// Should contain per-task error messages in the failures section
		if !strings.Contains(stdout, "title is required") {
			t.Errorf("stdout missing title validation error in failures section, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "priority must be") {
			t.Errorf("stdout missing priority validation error in failures section, got:\n%s", stdout)
		}
		// Should show correct summary counts
		if !strings.Contains(stdout, "Done: 1 imported, 2 failed") {
			t.Errorf("stdout missing correct summary, got:\n%s", stdout)
		}
	})

	t.Run("output omits Failures section when all tasks succeed", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		provider := &stubProvider{
			name: "test",
			tasks: []migrate.MigratedTask{
				{Title: "Task A", Status: task.StatusOpen},
				{Title: "Task B", Status: task.StatusOpen},
			},
		}

		var buf bytes.Buffer
		err := RunMigrate(dir, provider, true, false, &buf)

		if err != nil {
			t.Fatalf("RunMigrate returned error: %v", err)
		}

		stdout := buf.String()

		// Should NOT contain Failures: section
		if strings.Contains(stdout, "Failures:") {
			t.Errorf("stdout should not contain Failures: section when all tasks succeed, got:\n%s", stdout)
		}
		// Should show correct summary
		if !strings.Contains(stdout, "Done: 2 imported, 0 failed") {
			t.Errorf("stdout missing correct summary, got:\n%s", stdout)
		}
	})
}

func intPtr(v int) *int { return &v }

// setupBeadsFixture creates a .beads/issues.jsonl file in the given directory.
func setupBeadsFixture(t *testing.T, dir string, content string) {
	t.Helper()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("failed to create .beads dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "issues.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write issues.jsonl: %v", err)
	}
}
