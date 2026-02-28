package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestResolveID(t *testing.T) {
	created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)

	// Standard fixture: two tasks that share a 3-char prefix "a3f" but differ at char 4+.
	baseTasks := []task.Task{
		{ID: "tick-a3f1b2", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		{ID: "tick-a3f1b3", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		{ID: "tick-b12345", Title: "Task C", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
	}

	t.Run("it resolves a unique 3-char prefix to the full ID", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("b12")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"b12\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it strips tick- prefix before matching", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("tick-b12")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"tick-b12\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it normalizes input to lowercase", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("B12")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"B12\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it strips tick- prefix case-insensitively (TICK-A3F)", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("TICK-B12")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"TICK-B12\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it returns exact full ID immediately without ambiguity check", func(t *testing.T) {
		// tick-a3f1b2 and tick-a3f1b3 both share prefix "a3f1b". If we pass the
		// exact 6-char hex "a3f1b2", it should return immediately without ambiguity error.
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("tick-a3f1b2")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-a3f1b2" {
			t.Errorf("ResolveID(\"tick-a3f1b2\") = %q, want %q", got, "tick-a3f1b2")
		}
	})

	t.Run("it errors when prefix is shorter than 3 hex chars", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("ab")
		if err == nil {
			t.Fatal("expected error for 2-char prefix, got nil")
		}
		expected := "partial ID must be at least 3 hex characters"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it errors when prefix is 1 hex char", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("a")
		if err == nil {
			t.Fatal("expected error for 1-char prefix, got nil")
		}
		expected := "partial ID must be at least 3 hex characters"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it errors with ambiguous prefix listing all matching IDs", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("a3f")
		if err == nil {
			t.Fatal("expected ambiguity error, got nil")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "tick-a3f1b2") {
			t.Errorf("error should list tick-a3f1b2, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "tick-a3f1b3") {
			t.Errorf("error should list tick-a3f1b3, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "ambiguous") {
			t.Errorf("error should contain 'ambiguous', got %q", errMsg)
		}
	})

	t.Run("it errors with not found for non-matching prefix", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("zzz")
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
		expected := "task 'zzz' not found"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it resolves a 4-char prefix uniquely", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("b123")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"b123\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it resolves a 5-char prefix uniquely", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		got, err := store.ResolveID("b1234")
		if err != nil {
			t.Fatalf("ResolveID returned error: %v", err)
		}
		if got != "tick-b12345" {
			t.Errorf("ResolveID(\"b1234\") = %q, want %q", got, "tick-b12345")
		}
	})

	t.Run("it falls back to prefix search when 6-char input has no exact match", func(t *testing.T) {
		// IDs are exactly tick-{6 hex}. A 6-char input with no exact match falls through
		// to prefix search, which is equivalent to exact match at 6 chars. So the fallback
		// produces not-found when no task has that exact ID.
		singleTasks := []task.Task{
			{ID: "tick-abcde1", Title: "Task 1", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
			{ID: "tick-abcde2", Title: "Task 2", Status: task.StatusOpen, Priority: 2, Created: created, Updated: created},
		}
		tickDir := setupTickDirWithTasks(t, singleTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		// "abcdef" is 6 hex chars, no exact match exists, prefix search also finds nothing.
		_, err = store.ResolveID("abcdef")
		if err == nil {
			t.Fatal("expected not-found error for 6-char non-match fallback, got nil")
		}
		expected := "task 'abcdef' not found"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it errors when prefix is empty string", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("")
		if err == nil {
			t.Fatal("expected error for empty input, got nil")
		}
		expected := "partial ID must be at least 3 hex characters"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it errors when tick- prefix leaves fewer than 3 hex chars", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("tick-ab")
		if err == nil {
			t.Fatal("expected error for tick-ab (2 hex chars), got nil")
		}
		expected := "partial ID must be at least 3 hex characters"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it preserves original input in not-found error message", func(t *testing.T) {
		tickDir := setupTickDirWithTasks(t, baseTasks)
		store, err := NewStore(tickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()

		_, err = store.ResolveID("TICK-ZZZ")
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
		expected := "task 'TICK-ZZZ' not found"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})
}
