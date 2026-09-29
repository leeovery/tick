package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// nearCeilingGap is smaller than any transition record or blocked_by entry,
// so one recorded change takes a padded line to the old ceiling or past it.
const nearCeilingGap = 8

// nearCeiling pads tk's description so its stored line sits nearCeilingGap
// bytes under the old ceiling.
func nearCeiling(t *testing.T, tk task.Task) task.Task {
	t.Helper()
	tk.Description = "d"
	data, err := storage.MarshalJSONL([]task.Task{tk})
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	lineLen := len(data) - len("\n")
	tk.Description = strings.Repeat("d", len(tk.Description)+oldLineCeiling-nearCeilingGap-lineLen)
	return tk
}

func setupNearCeilingProject(t *testing.T, grows string, tasks ...task.Task) string {
	t.Helper()
	dir, _ := setupTickProjectWithTasks(t, tasks)
	if got := storedLineLen(t, dir, grows); got >= oldLineCeiling {
		t.Fatalf("seed line for %s is %d bytes, want under %d", grows, got, oldLineCeiling)
	}
	return dir
}

func assertShownStatus(t *testing.T, dir, id string, want task.Status) {
	t.Helper()
	doc := decodeToonDoc(t, runToonCommand(t, dir, "show", id))
	assertToonFields(t, doc, map[string]any{"id": id, "status": string(want)})
}

func assertChangedRows(t *testing.T, doc map[string]any, want ...map[string]any) {
	t.Helper()
	rows := toonRows(t, doc, "changed")
	if len(rows) != len(want) {
		t.Fatalf("changed has %d rows, want %d", len(rows), len(want))
	}
	for i, row := range rows {
		assertToonFields(t, row, want[i])
	}
}

func TestStatusAndDependencyGrowthCrossingOldCeiling(t *testing.T) {
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	closed := now.Add(time.Hour)
	plain := func(id, title string, status task.Status) task.Task {
		tk := task.Task{ID: id, Title: title, Status: status, Priority: 2, Created: now, Updated: now}
		if status == task.StatusDone || status == task.StatusCancelled {
			tk.Closed = &closed
		}
		return tk
	}

	t.Run("it starts a child whose cascade takes an unnamed parent past the old ceiling", func(t *testing.T) {
		parent := nearCeiling(t, plain("tick-aaa111", "Parent", task.StatusOpen))
		child := plain("tick-bbb222", "Child", task.StatusOpen)
		child.Parent = parent.ID
		dir := setupNearCeilingProject(t, parent.ID, parent, child)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "start", child.ID))

		assertChangedRows(t, doc,
			map[string]any{"id": child.ID, "from": "open", "to": "in_progress", "auto": false},
			map[string]any{"id": parent.ID, "from": "open", "to": "in_progress", "auto": true},
		)
		assertLineAtLeastOldCeiling(t, dir, parent.ID)
		assertListedIDs(t, dir, parent.ID, child.ID)
		assertShownStatus(t, dir, parent.ID, task.StatusInProgress)
	})

	t.Run("it adds a dependency that takes the task past the old ceiling", func(t *testing.T) {
		blocked := nearCeiling(t, plain("tick-aaa111", "Blocked", task.StatusOpen))
		blocker := plain("tick-bbb222", "Blocker", task.StatusOpen)
		dir := setupNearCeilingProject(t, blocked.ID, blocked, blocker)

		got := runToonCommand(t, dir, "dep", "add", blocked.ID, blocker.ID)

		if want := "Dependency added: tick-aaa111 blocked by tick-bbb222\n"; got != want {
			t.Errorf("dep add output = %q, want %q", got, want)
		}
		assertLineAtLeastOldCeiling(t, dir, blocked.ID)
		assertListedIDs(t, dir, blocked.ID, blocker.ID)
	})

	statusChanges := []struct {
		command string
		from    task.Status
		to      task.Status
	}{
		{"start", task.StatusOpen, task.StatusInProgress},
		{"done", task.StatusInProgress, task.StatusDone},
		{"cancel", task.StatusOpen, task.StatusCancelled},
		{"reopen", task.StatusDone, task.StatusOpen},
	}
	for _, sc := range statusChanges {
		t.Run("it reports "+sc.command+" whose transition takes the task past the old ceiling", func(t *testing.T) {
			grows := nearCeiling(t, plain("tick-aaa111", "Grows", sc.from))
			neighbour := plain("tick-bbb222", "Neighbour", task.StatusOpen)
			dir := setupNearCeilingProject(t, grows.ID, grows, neighbour)

			doc := decodeToonDoc(t, runToonCommand(t, dir, sc.command, grows.ID))

			assertChangedRows(t, doc, map[string]any{
				"id": grows.ID, "title": "Grows", "from": string(sc.from), "to": string(sc.to), "auto": false,
			})
			assertLineAtLeastOldCeiling(t, dir, grows.ID)
			assertListedIDs(t, dir, grows.ID, neighbour.ID)
			assertShownStatus(t, dir, grows.ID, sc.to)
		})
	}

	t.Run("it creates a child whose done-parent reopen takes the parent past the old ceiling", func(t *testing.T) {
		parent := nearCeiling(t, plain("tick-aaa111", "Parent", task.StatusDone))
		dir := setupNearCeilingProject(t, parent.ID, parent)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "create", "Child", "--parent", parent.ID))

		childID, _ := doc["id"].(string)
		assertToonFields(t, doc, map[string]any{"title": "Child", "parent": parent.ID})
		assertChangedRows(t, doc, map[string]any{"id": parent.ID, "from": "done", "to": "open", "auto": true})
		assertLineAtLeastOldCeiling(t, dir, parent.ID)
		assertListedIDs(t, dir, parent.ID, childID)
		assertShownStatus(t, dir, parent.ID, task.StatusOpen)
	})

	t.Run("it creates a task whose --blocks link takes the blocked task past the old ceiling", func(t *testing.T) {
		target := nearCeiling(t, plain("tick-aaa111", "Target", task.StatusOpen))
		dir := setupNearCeilingProject(t, target.ID, target)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "create", "Blocker", "--blocks", target.ID))

		blockerID, _ := doc["id"].(string)
		assertToonFields(t, doc, map[string]any{"title": "Blocker"})
		assertLineAtLeastOldCeiling(t, dir, target.ID)
		assertListedIDs(t, dir, target.ID, blockerID)
		shown := decodeToonDoc(t, runToonCommand(t, dir, "show", target.ID))
		rows := toonRows(t, shown, "blocked_by")
		if len(rows) != 1 || rows[0]["id"] != blockerID {
			t.Errorf("blocked_by rows = %v, want one row for %s", rows, blockerID)
		}
	})
}
