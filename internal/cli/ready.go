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
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var tasks []taskRow

	// Query ready tasks from SQLite
	err = store.Query(func(db *sql.DB) error {
		var queryErr error
		tasks, queryErr = queryReadyTasks(db)
		return queryErr
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Handle empty result
	if len(tasks) == 0 {
		fmt.Fprintln(a.Stdout, "No tasks found.")
		return 0
	}

	// Output
	if a.flags.Quiet {
		// --quiet: output only task IDs, one per line
		for _, t := range tasks {
			fmt.Fprintln(a.Stdout, t.ID)
		}
	} else {
		// Aligned columns: ID (12), STATUS (12), PRI (4), TITLE (remainder)
		fmt.Fprintf(a.Stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
		for _, t := range tasks {
			fmt.Fprintf(a.Stdout, "%-12s %-12s %-4d %s\n", t.ID, t.Status, t.Priority, t.Title)
		}
	}

	return 0
}
