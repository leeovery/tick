package cli

import (
	"slices"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// Three records numbered in authoring order; IDs descend, and the appended
// ID sorts ahead of all three.
const (
	numberedFirstID  = "tick-d00004"
	numberedSecondID = "tick-c00003"
	numberedThirdID  = "tick-b00002"
	unnumberedID     = "tick-000001"
)

func numberedLines() []string {
	return []string{
		seqLine(numberedFirstID, "first", "", 1),
		seqLine(numberedSecondID, "second", "", 2),
		seqLine(numberedThirdID, "third", "", 3),
	}
}

func unnumberedTask(t *testing.T, id string) task.Task {
	t.Helper()
	created, err := time.Parse(time.RFC3339, sameSecond)
	if err != nil {
		t.Fatalf("failed to parse %q: %v", sameSecond, err)
	}
	return task.Task{ID: id, Title: id, Status: task.StatusOpen, Priority: 2, Created: created, Updated: created}
}

func mutateStore(t *testing.T, tickDir string, fn func(tasks []task.Task) ([]task.Task, error)) {
	t.Helper()
	store, err := storage.NewStore(tickDir)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	defer store.Close()
	if err := store.Mutate(fn); err != nil {
		t.Fatalf("Mutate returned error: %v", err)
	}
}

func appendUnnumbered(t *testing.T, tickDir string) {
	t.Helper()
	mutateStore(t, tickDir, func(tasks []task.Task) ([]task.Task, error) {
		return append(tasks, unnumberedTask(t, unnumberedID)), nil
	})
}

func readStoreSeqs(t *testing.T, tickDir string) map[string]int {
	t.Helper()
	store, err := storage.NewStore(tickDir)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	defer store.Close()
	tasks, err := store.ReadTasks()
	if err != nil {
		t.Fatalf("ReadTasks returned error: %v", err)
	}
	seqs := map[string]int{}
	for _, tk := range tasks {
		seqs[tk.ID] = tk.Seq
	}
	return seqs
}

func TestMutateNumbersUnnumberedTasks(t *testing.T) {
	wantSeqs := map[string]int{numberedFirstID: 1, numberedSecondID: 2, numberedThirdID: 3, unnumberedID: 4}

	t.Run("it writes and caches an appended unnumbered task above the highest sequence", func(t *testing.T) {
		_, tickDir := setupRawProject(t, numberedLines()...)

		appendUnnumbered(t, tickDir)

		if got := rawSeqs(t, tickDir); !slices.Equal(got, []int{1, 2, 3, 4}) {
			t.Errorf("tasks.jsonl seqs = %v, want [1 2 3 4]", got)
		}
		assertCachedSeqs(t, tickDir, wantSeqs)
	})

	t.Run("it lists the appended task last and readies the first-authored task first straight after the write", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, numberedLines()...)

		appendUnnumbered(t, tickDir)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{numberedFirstID, numberedSecondID, numberedThirdID, unnumberedID})
		assertIDOrder(t, listedIDs(t, dir, "ready", "--count", "1"), []string{numberedFirstID})
	})

	t.Run("it caches what rebuild and ReadTasks derive from the written bytes", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, numberedLines()...)

		appendUnnumbered(t, tickDir)
		assertCachedSeqs(t, tickDir, wantSeqs)

		runToonCommand(t, dir, "rebuild")
		assertCachedSeqs(t, tickDir, wantSeqs)

		for id, got := range readStoreSeqs(t, tickDir) {
			if want := wantSeqs[id]; got != want {
				t.Errorf("ReadTasks seq for %s = %d, want %d", id, got, want)
			}
		}
	})

	t.Run("it numbers a mutation's unnumbered records above every carried sequence in record order", func(t *testing.T) {
		_, tickDir := setupTickProject(t)
		numbered := func(id string, seq int) task.Task {
			tk := unnumberedTask(t, id)
			tk.Seq = seq
			return tk
		}

		mutateStore(t, tickDir, func([]task.Task) ([]task.Task, error) {
			return []task.Task{
				numbered(mergeA, 1),
				numbered(mergeB, 2),
				numbered(mergeC, 3),
				unnumberedTask(t, mergeX),
				unnumberedTask(t, mergeY),
				numbered(mergeD, 4),
				numbered(mergeE, 5),
			}, nil
		})

		if got := rawSeqs(t, tickDir); !slices.Equal(got, []int{1, 2, 3, 6, 7, 4, 5}) {
			t.Errorf("tasks.jsonl seqs = %v, want [1 2 3 6 7 4 5]", got)
		}
		assertCachedSeqs(t, tickDir, mergeShapeSeqs)
	})
}
