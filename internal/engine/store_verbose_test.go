package engine

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestStoreVerbose(t *testing.T) {
	t.Run("it logs lock/cache/hash/write operations when verbose is on", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr, true)

		s, err := NewStore(tickDir, WithVerbose(vl))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// Mutate triggers: lock acquire, freshness check, cache rebuild, atomic write, lock release
		err = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			return existing, nil
		})
		if err != nil {
			t.Fatalf("Mutate: %v", err)
		}

		output := stderr.String()

		// Check all expected verbose lines are present
		expectedPhrases := []string{
			"verbose: lock acquired (exclusive)",
			"verbose: cache freshness check",
			"verbose: cache rebuild",
			"verbose: atomic write",
			"verbose: lock released",
		}
		for _, phrase := range expectedPhrases {
			if !strings.Contains(output, phrase) {
				t.Errorf("expected verbose output to contain %q, got:\n%s", phrase, output)
			}
		}
	})

	t.Run("it logs lock/cache/hash operations during query when verbose is on", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr, true)

		s, err := NewStore(tickDir, WithVerbose(vl))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		err = s.Query(func(db *sql.DB) error {
			return nil
		})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}

		output := stderr.String()

		expectedPhrases := []string{
			"verbose: lock acquired (shared)",
			"verbose: cache freshness check",
			"verbose: lock released",
		}
		for _, phrase := range expectedPhrases {
			if !strings.Contains(output, phrase) {
				t.Errorf("expected verbose output to contain %q, got:\n%s", phrase, output)
			}
		}
	})

	t.Run("it writes nothing when verbose is off", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr, false)

		s, err := NewStore(tickDir, WithVerbose(vl))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		err = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			return existing, nil
		})
		if err != nil {
			t.Fatalf("Mutate: %v", err)
		}

		if stderr.Len() != 0 {
			t.Errorf("expected no verbose output when off, got %q", stderr.String())
		}
	})

	t.Run("it logs hash comparison on freshness check", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr, true)

		s, err := NewStore(tickDir, WithVerbose(vl))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		// First query triggers rebuild
		_ = s.Query(func(db *sql.DB) error { return nil })

		stderr.Reset()

		// Second query should show hash match (cache is fresh)
		_ = s.Query(func(db *sql.DB) error { return nil })

		output := stderr.String()
		if !strings.Contains(output, "verbose: cache freshness check") {
			t.Errorf("expected freshness check log, got:\n%s", output)
		}
		if !strings.Contains(output, "fresh") {
			t.Errorf("expected 'fresh' in hash comparison output, got:\n%s", output)
		}
	})

	t.Run("it prefixes all lines with verbose:", func(t *testing.T) {
		tasks := sampleTasks()
		tickDir := setupTickDirWithTasks(t, tasks)

		var stderr bytes.Buffer
		vl := NewVerboseLogger(&stderr, true)

		s, err := NewStore(tickDir, WithVerbose(vl))
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer s.Close()

		_ = s.Mutate(func(existing []task.Task) ([]task.Task, error) {
			return existing, nil
		})

		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "verbose: ") {
				t.Errorf("expected all lines to start with 'verbose: ', got %q", line)
			}
		}
	})
}
