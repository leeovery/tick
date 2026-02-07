package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestDepAdd_AddsDependencyBetweenTwoExistingTasks(t *testing.T) {
	t.Run("it adds a dependency between two existing tasks", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if len(taskA.BlockedBy) != 1 || taskA.BlockedBy[0] != "tick-bbb222" {
			t.Errorf("expected blocked_by [tick-bbb222], got %v", taskA.BlockedBy)
		}
	})
}

func TestDepRm_RemovesExistingDependency(t *testing.T) {
	t.Run("it removes an existing dependency", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if len(taskA.BlockedBy) != 0 {
			t.Errorf("expected empty blocked_by after removal, got %v", taskA.BlockedBy)
		}
	})
}

func TestDepAdd_OutputsConfirmation(t *testing.T) {
	t.Run("it outputs confirmation on success (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		expected := "Dependency added: tick-aaa111 blocked by tick-bbb222"
		if output != expected {
			t.Errorf("expected output %q, got %q", expected, output)
		}
	})
}

func TestDepRm_OutputsConfirmation(t *testing.T) {
	t.Run("it outputs confirmation on success (rm)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		output := strings.TrimSpace(stdout.String())
		expected := "Dependency removed: tick-aaa111 no longer blocked by tick-bbb222"
		if output != expected {
			t.Errorf("expected output %q, got %q", expected, output)
		}
	})
}

func TestDepAdd_UpdatesTimestamp(t *testing.T) {
	t.Run("it updates task's updated timestamp on dep add", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if !taskA.Updated.After(now) {
			t.Errorf("expected updated timestamp to be refreshed, got %v (original: %v)", taskA.Updated, now)
		}
	})
}

func TestDepRm_UpdatesTimestamp(t *testing.T) {
	t.Run("it updates task's updated timestamp on dep rm", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if !taskA.Updated.After(now) {
			t.Errorf("expected updated timestamp to be refreshed, got %v (original: %v)", taskA.Updated, now)
		}
	})
}

func TestDepAdd_ErrorTaskIDNotFound(t *testing.T) {
	t.Run("it errors when task_id not found (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-nonexist", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-nonexist") {
			t.Errorf("expected error to contain task ID, got %q", errMsg)
		}
	})
}

func TestDepRm_ErrorTaskIDNotFound(t *testing.T) {
	t.Run("it errors when task_id not found (rm)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm", "tick-nonexist", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-nonexist") {
			t.Errorf("expected error to contain task ID, got %q", errMsg)
		}
	})
}

func TestDepAdd_ErrorBlockedByIDNotFound(t *testing.T) {
	t.Run("it errors when blocked_by_id not found (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-nonexist"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-nonexist") {
			t.Errorf("expected error to contain blocked_by ID, got %q", errMsg)
		}
	})
}

func TestDepAdd_ErrorDuplicateDependency(t *testing.T) {
	t.Run("it errors on duplicate dependency (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}

		// Verify no mutation occurred
		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if len(taskA.BlockedBy) != 1 {
			t.Errorf("expected blocked_by to remain unchanged with 1 entry, got %v", taskA.BlockedBy)
		}
	})
}

func TestDepRm_ErrorDependencyNotFound(t *testing.T) {
	t.Run("it errors when dependency not found (rm)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestDepAdd_ErrorSelfReference(t *testing.T) {
	t.Run("it errors on self-reference (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-aaa111"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestDepAdd_ErrorCycle(t *testing.T) {
	t.Run("it errors when add creates cycle", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		// tick-bbb222 is blocked by tick-aaa111
		// Adding tick-aaa111 blocked by tick-bbb222 creates cycle
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-aaa111"}, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "cycle") {
			t.Errorf("expected cycle error message, got %q", errMsg)
		}
	})
}

func TestDepAdd_ErrorChildBlockedByParent(t *testing.T) {
	t.Run("it errors when add creates child-blocked-by-parent", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-parent1", Title: "Parent", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-child01", Title: "Child", Status: task.StatusOpen, Priority: 2, Parent: "tick-parent1", Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-child01", "tick-parent1"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "parent") {
			t.Errorf("expected parent-related error message, got %q", errMsg)
		}
	})
}

func TestDep_NormalizesIDsToLowercase(t *testing.T) {
	t.Run("it normalizes IDs to lowercase (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		// Use uppercase IDs
		code := app.Run([]string{"tick", "dep", "add", "TICK-AAA111", "TICK-BBB222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if len(taskA.BlockedBy) != 1 || taskA.BlockedBy[0] != "tick-bbb222" {
			t.Errorf("expected blocked_by [tick-bbb222] (lowercase), got %v", taskA.BlockedBy)
		}

		// Output should use lowercase IDs
		output := strings.TrimSpace(stdout.String())
		expected := "Dependency added: tick-aaa111 blocked by tick-bbb222"
		if output != expected {
			t.Errorf("expected output %q, got %q", expected, output)
		}
	})

	t.Run("it normalizes IDs to lowercase (rm)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		// Use uppercase IDs
		code := app.Run([]string{"tick", "dep", "rm", "TICK-AAA111", "TICK-BBB222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		tasks := readTasksFromDir(t, dir)
		var taskA *task.Task
		for i := range tasks {
			if tasks[i].ID == "tick-aaa111" {
				taskA = &tasks[i]
				break
			}
		}
		if taskA == nil {
			t.Fatal("could not find task tick-aaa111")
		}
		if len(taskA.BlockedBy) != 0 {
			t.Errorf("expected empty blocked_by after removal, got %v", taskA.BlockedBy)
		}
	})
}

func TestDep_QuietSuppressesOutput(t *testing.T) {
	t.Run("it suppresses output with --quiet (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it suppresses output with --quiet (rm)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, BlockedBy: []string{"tick-bbb222"}, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "--quiet", "dep", "rm", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		if stdout.String() != "" {
			t.Errorf("expected no stdout with --quiet, got %q", stdout.String())
		}
	})
}

func TestDep_ErrorFewerThanTwoIDs(t *testing.T) {
	t.Run("it errors when fewer than two IDs provided (add, zero args)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Usage:") {
			t.Errorf("expected usage hint in error, got %q", errMsg)
		}
	})

	t.Run("it errors when fewer than two IDs provided (add, one arg)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Usage:") {
			t.Errorf("expected usage hint in error, got %q", errMsg)
		}
	})

	t.Run("it errors when fewer than two IDs provided (rm, zero args)", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "rm"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "Usage:") {
			t.Errorf("expected usage hint in error, got %q", errMsg)
		}
	})

	t.Run("it errors when no subcommand provided to dep", func(t *testing.T) {
		dir := setupInitializedDir(t)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		errMsg := stderr.String()
		if !strings.Contains(errMsg, "Error:") {
			t.Errorf("expected error on stderr, got %q", errMsg)
		}
	})
}

func TestDep_PersistsViaAtomicWrite(t *testing.T) {
	t.Run("it persists via atomic write (add)", func(t *testing.T) {
		now := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		existing := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}
		dir := setupInitializedDirWithTasks(t, existing)
		var stdout, stderr bytes.Buffer

		app := &App{Stdout: &stdout, Stderr: &stderr, Dir: dir}
		code := app.Run([]string{"tick", "dep", "add", "tick-aaa111", "tick-bbb222"})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
		}

		// Read raw file and verify it's valid JSONL
		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		data, err := os.ReadFile(jsonlPath)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines in JSONL, got %d", len(lines))
		}

		// Parse first line and verify blocked_by was added
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
			t.Fatalf("failed to parse JSONL line as JSON: %v", err)
		}
		if parsed["id"] == "tick-aaa111" {
			blockedBy, ok := parsed["blocked_by"].([]interface{})
			if !ok || len(blockedBy) != 1 {
				t.Errorf("expected blocked_by with 1 entry in persisted JSONL, got %v", parsed["blocked_by"])
			}
		}

		// Verify cache.db was updated
		cacheDB := filepath.Join(dir, ".tick", "cache.db")
		if _, err := os.Stat(cacheDB); os.IsNotExist(err) {
			t.Error("expected cache.db to exist after dep add")
		}
	})
}
