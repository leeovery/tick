package cli

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
)

// readySQL selects tasks that are ready: open, all blockers closed, no open children.
// A task is ready when:
//  1. Status is 'open'
//  2. All blocked_by tasks are done or cancelled (or no blockers at all)
//  3. No children with status 'open' or 'in_progress'
//
// Ordered by priority ASC, created ASC for deterministic results.
const readySQL = `
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

// RunReady executes the ready command: queries ready tasks from SQLite and outputs them
// in aligned columns. This is an alias for "list --ready".
func RunReady(dir string, quiet bool, stdout io.Writer) error {
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	type listRow struct {
		id       string
		status   string
		priority int
		title    string
	}

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(readySQL)
		if err != nil {
			return fmt.Errorf("failed to query ready tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
				return fmt.Errorf("failed to scan ready task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Fprintln(stdout, "No tasks found.")
		return nil
	}

	if quiet {
		for _, r := range rows {
			fmt.Fprintln(stdout, r.id)
		}
		return nil
	}

	// Print header (same format as list)
	fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")

	// Print rows
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
	}

	return nil
}
