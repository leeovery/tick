package cli

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// beadsIssueLine renders one beads issue at priority 2; an empty createdAt
// leaves the issue without a creation time.
func beadsIssueLine(title, createdAt string) string {
	return `{"id":"b-` + title + `","title":"` + title + `","status":"open","priority":2,"issue_type":"task","created_at":"` +
		createdAt + `","updated_at":"","closed_at":"","dependencies":[]}`
}

func setupBeadsIssues(t *testing.T, dir string, lines ...string) {
	t.Helper()
	setupBeadsFixture(t, dir, strings.Join(lines, "\n")+"\n")
}

func migrateBeads(t *testing.T, dir string) {
	t.Helper()
	if _, stderr, code := runMigrate(t, dir, "--from", "beads"); code != 0 {
		t.Fatalf("migrate exit code = %d, want 0; stderr = %q", code, stderr)
	}
}

// importedIDs returns the IDs of the tasks titled titles, in the order of titles.
func importedIDs(t *testing.T, tickDir string, titles []string) []string {
	t.Helper()
	byTitle := map[string]string{}
	for _, tk := range readPersistedTasks(t, tickDir) {
		byTitle[tk.Title] = tk.ID
	}
	ids := make([]string, 0, len(titles))
	for _, title := range titles {
		id, ok := byTitle[title]
		if !ok {
			t.Fatalf("imported task %q not found in tasks.jsonl", title)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestMigrateAssignsSequence(t *testing.T) {
	importOrder := []string{"one", "two", "three", "four", "five"}

	t.Run("it lists a same-second import without creation times in import order", func(t *testing.T) {
		// IDs are random and each import stamps the wall clock: retry until the
		// batch shares one second and its IDs are not already ascending.
		const maxAttempts = 20
		for range maxAttempts {
			dir, tickDir := setupTickProject(t)
			var lines []string
			for _, title := range importOrder {
				lines = append(lines, beadsIssueLine(title, ""))
			}
			setupBeadsIssues(t, dir, lines...)
			migrateBeads(t, dir)

			ids := importedIDs(t, tickDir, importOrder)
			if !sameCreatedSecond(readPersistedTasks(t, tickDir)) || slices.IsSorted(ids) {
				continue
			}

			if got := rawSeqs(t, tickDir); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
				t.Errorf("stored seqs = %v, want [1 2 3 4 5]", got)
			}
			assertIDOrder(t, listedIDs(t, dir, "list"), ids)
			return
		}
		t.Fatalf("no import of %d issues tied on one second with non-ascending IDs in %d attempts", len(importOrder), maxAttempts)
	})

	t.Run("it numbers an import above the project's highest sequence in import order", func(t *testing.T) {
		dir, tickDir := setupRawProject(t,
			seqLine("tick-aaa111", "existing-a", "", 4),
			seqLine("tick-bbb222", "existing-b", "", 9),
		)
		setupBeadsIssues(t, dir,
			beadsIssueLine("one", ""),
			beadsIssueLine("two", ""),
			beadsIssueLine("three", ""),
		)
		migrateBeads(t, dir)

		if got := rawSeqs(t, tickDir); !slices.Equal(got, []int{4, 9, 10, 11, 12}) {
			t.Errorf("stored seqs = %v, want [4 9 10 11 12]", got)
		}
	})

	t.Run("it lists an import by its creation times across seconds, not import order", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		setupBeadsIssues(t, dir,
			beadsIssueLine("one", "2026-01-10T09:00:04Z"),
			beadsIssueLine("two", "2026-01-10T09:00:03Z"),
			beadsIssueLine("three", "2026-01-10T09:00:02Z"),
			beadsIssueLine("four", "2026-01-10T09:00:01Z"),
			beadsIssueLine("five", "2026-01-10T09:00:00Z"),
		)
		migrateBeads(t, dir)

		byCreated := importedIDs(t, tickDir, importOrder)
		slices.Reverse(byCreated)
		assertIDOrder(t, listedIDs(t, dir, "list"), byCreated)
	})

	t.Run("it lists an import sharing one second in import order whatever the source fractions", func(t *testing.T) {
		// IDs are random: retry until they are not already ascending in import order.
		const maxAttempts = 20
		for range maxAttempts {
			dir, tickDir := setupTickProject(t)
			setupBeadsIssues(t, dir,
				beadsIssueLine("one", "2026-01-10T09:00:00.9Z"),
				beadsIssueLine("two", "2026-01-10T09:00:00.7Z"),
				beadsIssueLine("three", "2026-01-10T09:00:00.5Z"),
				beadsIssueLine("four", "2026-01-10T09:00:00.3Z"),
				beadsIssueLine("five", "2026-01-10T09:00:00.1Z"),
			)
			migrateBeads(t, dir)

			ids := importedIDs(t, tickDir, importOrder)
			if slices.IsSorted(ids) {
				continue
			}

			wantCreated := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
			for _, tk := range readPersistedTasks(t, tickDir) {
				if !tk.Created.Equal(wantCreated) {
					t.Errorf("%s created = %s, want %s", tk.Title, tk.Created.Format(time.RFC3339Nano), task.FormatTimestamp(wantCreated))
				}
			}
			assertIDOrder(t, listedIDs(t, dir, "list"), ids)
			return
		}
		t.Fatalf("no import of %d issues with non-ascending IDs in %d attempts", len(importOrder), maxAttempts)
	})
}
