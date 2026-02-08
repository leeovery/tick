package cli

import (
	"database/sql"
	"fmt"
)

// readyConditionsFor returns the WHERE clause conditions that define a "ready" task
// parameterized by table alias:
// 1. All blocked_by tasks are closed (done or cancelled)
// 2. No children with status 'open' or 'in_progress'
// Shared by readyQuery, blockedQuery, and list combined filters so ready logic
// is defined once.
func readyConditionsFor(alias string) string {
	return `
  NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON d.blocked_by = blocker.id
    WHERE d.task_id = ` + alias + `.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = ` + alias + `.id
      AND child.status IN ('open', 'in_progress')
  )`
}

// readyQuery is the SQL query that returns tasks matching all ready conditions:
// 1. Status is 'open'
// 2. All blocked_by tasks are closed (done or cancelled)
// 3. No children with status 'open' or 'in_progress'
// Order: priority ASC, created ASC (deterministic)
var readyQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND` + readyConditionsFor("t") + `
ORDER BY t.priority ASC, t.created ASC
`

// runReady implements the `tick ready` command.
// It is an alias for `tick list --ready`, returning tasks that are workable.
func (a *App) runReady(args []string) error {
	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var rows []listRow

	err = s.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(readyQuery)
		if err != nil {
			return fmt.Errorf("failed to query ready tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.ID, &r.Status, &r.Priority, &r.Title); err != nil {
				return fmt.Errorf("failed to scan task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	return a.Formatter.FormatTaskList(a.Stdout, rows, a.Quiet)
}
