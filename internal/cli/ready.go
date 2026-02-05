package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// ReadyCondition returns SQL WHERE clause conditions for ready tasks.
// A task is ready when:
// 1. Status is 'open'
// 2. All blockers are closed (done or cancelled)
// 3. No children with status open or in_progress
//
// The condition assumes the main table is aliased as 't'.
// Reusable by tick ready, tick list --ready, and tick blocked.
const ReadyCondition = `
	t.status = 'open'
	AND NOT EXISTS (
		SELECT 1 FROM dependencies d
		JOIN tasks blocker ON d.blocked_by = blocker.id
		WHERE d.task_id = t.id
		  AND blocker.status IN ('open', 'in_progress')
	)
	AND NOT EXISTS (
		SELECT 1 FROM tasks child
		WHERE child.parent = t.id
		  AND child.status IN ('open', 'in_progress')
	)
`

// taskRow represents a task row for list-style output.
type taskRow struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// scanTaskRows scans sql.Rows into a slice of taskRow.
// Expects rows with columns: id, title, status, priority.
func scanTaskRows(rows *sql.Rows) ([]taskRow, error) {
	var tasks []taskRow
	for rows.Next() {
		var t taskRow
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// renderTaskList converts taskRows to formatted output via the configured formatter.
// Handles --quiet mode (ID-only output) and normal formatted output.
func (a *App) renderTaskList(tasks []taskRow) {
	if a.formatConfig.Quiet {
		for _, t := range tasks {
			fmt.Fprintln(a.Stdout, t.ID)
		}
		return
	}

	data := &TaskListData{
		Tasks: make([]TaskRowData, len(tasks)),
	}
	for i, t := range tasks {
		data.Tasks[i] = TaskRowData{
			ID:       t.ID,
			Title:    t.Title,
			Status:   t.Status,
			Priority: t.Priority,
		}
	}
	formatter := a.formatConfig.Formatter()
	fmt.Fprint(a.Stdout, formatter.FormatTaskList(data))
}

// queryReadyTasks queries tasks matching the ready criteria.
// Returns slice of taskRow ordered by priority ASC, created ASC.
func queryReadyTasks(db *sql.DB) ([]taskRow, error) {
	query := `
		SELECT t.id, t.title, t.status, t.priority
		FROM tasks t
		WHERE ` + ReadyCondition + `
		ORDER BY t.priority ASC, t.created ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTaskRows(rows)
}

// runReady executes the ready subcommand.
// Returns tasks that are open, have no open/in_progress blockers, and have no open/in_progress children.
func (a *App) runReady() int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Open store
	a.WriteVerbose("store open %s", tickDir)
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var tasks []taskRow

	// Query ready tasks from SQLite
	a.WriteVerbose("lock acquire shared")
	a.WriteVerbose("cache freshness check")
	err = store.Query(func(db *sql.DB) error {
		var queryErr error
		tasks, queryErr = queryReadyTasks(db)
		return queryErr
	})
	a.WriteVerbose("lock release")

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	a.renderTaskList(tasks)
	return 0
}
