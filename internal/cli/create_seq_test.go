package cli

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

// createTaskID runs tick create with title and returns the new task's ID.
func createTaskID(t *testing.T, dir, tickDir, title string) string {
	t.Helper()
	runToonCommand(t, dir, "create", title)
	for _, tk := range readPersistedTasks(t, tickDir) {
		if tk.Title == title {
			return tk.ID
		}
	}
	t.Fatalf("created task %q not found in tasks.jsonl", title)
	return ""
}

// persistedSeqs maps each task ID to the seq its tasks.jsonl record carries,
// 0 where it carries none.
func persistedSeqs(t *testing.T, tickDir string) map[string]int {
	t.Helper()
	seqs := map[string]int{}
	for _, record := range rawRecords(t, tickDir) {
		seqs[record.ID] = record.Seq
	}
	return seqs
}

func cachedSeq(t *testing.T, tickDir, id string) int {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tickDir, "cache.db"))
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()
	var seq int
	if err := db.QueryRow(`SELECT seq FROM tasks WHERE id = ?`, id).Scan(&seq); err != nil {
		t.Fatalf("failed to read seq for %s: %v", id, err)
	}
	return seq
}

func assertUniqueSeqs(t *testing.T, records []storedRecord) {
	t.Helper()
	seen := map[int]string{}
	for _, record := range records {
		if other, ok := seen[record.Seq]; ok {
			t.Errorf("tasks %s and %s share sequence %d", other, record.ID, record.Seq)
		}
		seen[record.Seq] = record.ID
	}
}

func TestCreateAssignsSequence(t *testing.T) {
	t.Run("it numbers successive creates 1, 2, 3 in an empty project", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		for _, title := range []string{"first", "second", "third"} {
			createTaskID(t, dir, tickDir, title)
		}

		var titles []string
		for _, tk := range readPersistedTasks(t, tickDir) {
			titles = append(titles, tk.Title)
		}
		if !slices.Equal(titles, []string{"first", "second", "third"}) {
			t.Fatalf("titles = %v, want creation order", titles)
		}
		if seqs := rawSeqs(t, tickDir); !slices.Equal(seqs, []int{1, 2, 3}) {
			t.Errorf("seqs = %v, want [1 2 3]", seqs)
		}
	})

	t.Run("it numbers one above the highest sequence, not the last line or the count", func(t *testing.T) {
		dir, tickDir := setupRawProject(t,
			seqLine("tick-aaa111", "three", "", 3),
			seqLine("tick-bbb222", "seven", "", 7),
			seqLine("tick-ccc333", "five", "", 5),
		)

		id := createTaskID(t, dir, tickDir, "new")

		if got := persistedSeqs(t, tickDir)[id]; got != 8 {
			t.Errorf("new task seq = %d, want 8", got)
		}
	})

	t.Run("it keeps a created task's sequence through update, remove, create and rebuild", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		keptID := createTaskID(t, dir, tickDir, "kept")
		otherID := createTaskID(t, dir, tickDir, "other")
		const wantSeq = 1

		assertKept := func(step string) {
			t.Helper()
			if got := persistedSeqs(t, tickDir)[keptID]; got != wantSeq {
				t.Errorf("after %s: tasks.jsonl seq = %d, want %d", step, got, wantSeq)
			}
			if got := cachedSeq(t, tickDir, keptID); got != wantSeq {
				t.Errorf("after %s: cache seq = %d, want %d", step, got, wantSeq)
			}
		}

		assertKept("create")
		runToonCommand(t, dir, "update", keptID, "--title", "kept renamed")
		assertKept("update")
		runToonCommand(t, dir, "remove", otherID, "--force")
		assertKept("remove")
		createTaskID(t, dir, tickDir, "later-1")
		createTaskID(t, dir, tickDir, "later-2")
		assertKept("further creates")
		runToonCommand(t, dir, "rebuild")
		assertKept("rebuild")
	})

	t.Run("it reuses a removed highest sequence without duplicating one", func(t *testing.T) {
		dir, tickDir := setupTickProject(t)
		idA := createTaskID(t, dir, tickDir, "A")
		idB := createTaskID(t, dir, tickDir, "B")
		idC := createTaskID(t, dir, tickDir, "C")

		seqs := persistedSeqs(t, tickDir)
		if got := []int{seqs[idA], seqs[idB], seqs[idC]}; !slices.Equal(got, []int{1, 2, 3}) {
			t.Fatalf("A, B, C seqs = %v, want [1 2 3]", got)
		}

		runToonCommand(t, dir, "remove", idC, "--force")
		idD := createTaskID(t, dir, tickDir, "D")

		if got := persistedSeqs(t, tickDir)[idD]; got != 3 {
			t.Errorf("seq after removing C = %d, want 3", got)
		}
		assertUniqueSeqs(t, rawRecords(t, tickDir))
	})

	t.Run("it lists same-second same-priority creates in creation order", func(t *testing.T) {
		// IDs are random and creation reads the wall clock: retry until the
		// batch shares one second and its IDs are not already ascending.
		const batchSize = 5
		const maxAttempts = 20
		for range maxAttempts {
			dir, tickDir := setupTickProject(t)
			var created []string
			for i := range batchSize {
				created = append(created, createTaskID(t, dir, tickDir, "step-"+strconv.Itoa(i+1)))
			}
			if !sameCreatedSecond(readPersistedTasks(t, tickDir)) || slices.IsSorted(created) {
				continue
			}

			assertIDOrder(t, listedIDs(t, dir, "list"), created)
			return
		}
		t.Fatalf("no batch of %d creates tied on one second with non-ascending IDs in %d attempts", batchSize, maxAttempts)
	})
}

func sameCreatedSecond(tasks []task.Task) bool {
	for _, tk := range tasks {
		if !tk.Created.Equal(tasks[0].Created) {
			return false
		}
	}
	return true
}
