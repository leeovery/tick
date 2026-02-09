package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/engine"
)

// ReadyQuery is the SQL query that returns tasks matching all three ready
// conditions: status is open, all blockers are closed (done/cancelled), and
// no children have status open or in_progress. Results are ordered by
// priority ASC then created ASC for deterministic output.
const ReadyQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )
ORDER BY t.priority ASC, t.created ASC
`

// runReady implements the "tick ready" command, which is an alias for
// "tick list --ready". It queries for tasks that are workable: open,
// unblocked, and have no open children.
func runReady(ctx *Context) error {
	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(ReadyQuery)
		if err != nil {
			return fmt.Errorf("querying ready tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
				return fmt.Errorf("scanning ready task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Fprintln(ctx.Stdout, "No tasks found.")
		return nil
	}

	if ctx.Quiet {
		for _, r := range rows {
			fmt.Fprintln(ctx.Stdout, r.id)
		}
		return nil
	}

	printListTable(ctx.Stdout, rows)
	return nil
}
