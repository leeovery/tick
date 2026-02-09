package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/engine"
)

// BlockedQuery is the SQL query that returns open tasks failing ready
// conditions: they have at least one unclosed blocker OR have open/in_progress
// children. This is the inverse of ReadyQuery -- blocked = open AND NOT ready.
// Results are ordered by priority ASC then created ASC for deterministic output.
const BlockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON blocker.id = d.blocked_by
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )
ORDER BY t.priority ASC, t.created ASC
`

// runBlocked implements the "tick blocked" command, which is an alias for
// "tick list --blocked". It queries for tasks that are open but not workable:
// they have unclosed blockers or open children.
func runBlocked(ctx *Context) error {
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
		sqlRows, err := db.Query(BlockedQuery)
		if err != nil {
			return fmt.Errorf("querying blocked tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
				return fmt.Errorf("scanning blocked task row: %w", err)
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
