package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// BlockedCondition returns SQL WHERE clause conditions for blocked tasks.
// A task is blocked when:
// 1. Status is 'open'
// 2. AND either:
//   - Has at least one blocker with status 'open' or 'in_progress', OR
//   - Has at least one child with status 'open' or 'in_progress'
//
// In other words: blocked = open tasks that are not ready.
// The condition assumes the main table is aliased as 't'.
const BlockedCondition = `
	t.status = 'open'
	AND (
		EXISTS (
			SELECT 1 FROM dependencies d
			JOIN tasks blocker ON d.blocked_by = blocker.id
			WHERE d.task_id = t.id
			  AND blocker.status IN ('open', 'in_progress')
		)
		OR EXISTS (
			SELECT 1 FROM tasks child
			WHERE child.parent = t.id
			  AND child.status IN ('open', 'in_progress')
		)
	)
`

// queryBlockedTasks queries tasks matching the blocked criteria.
// Returns slice of taskRow ordered by priority ASC, created ASC.
func queryBlockedTasks(db *sql.DB) ([]taskRow, error) {
	query := `
		SELECT t.id, t.title, t.status, t.priority
		FROM tasks t
		WHERE ` + BlockedCondition + `
		ORDER BY t.priority ASC, t.created ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTaskRows(rows)
}

// runBlocked executes the blocked subcommand.
// Returns tasks that are open and have either unclosed blockers or open children.
func (a *App) runBlocked() int {
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

	// Query blocked tasks from SQLite
	a.WriteVerbose("lock acquire shared")
	a.WriteVerbose("cache freshness check")
	err = store.Query(func(db *sql.DB) error {
		var queryErr error
		tasks, queryErr = queryBlockedTasks(db)
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
