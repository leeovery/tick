package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// ReadySQL is the SQL query for ready tasks: open, all blockers closed, no open children.
// Exported so it can be reused by other queries (e.g., blocked query, list filters).
const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )
ORDER BY priority ASC, created ASC`

// BlockedSQL is the SQL query for blocked tasks: open tasks that are NOT ready.
// A task is blocked if it is open AND has unclosed blockers OR has open children.
// This is the inverse of ReadySQL — reuses the same subqueries.
const BlockedSQL = `SELECT id, status, priority, title FROM tasks
WHERE status = 'open'
  AND (
    id IN (
      SELECT d.task_id FROM dependencies d
      JOIN tasks t ON d.blocked_by = t.id
      WHERE t.status NOT IN ('done', 'cancelled')
    )
    OR id IN (
      SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
    )
  )
ORDER BY priority ASC, created ASC`

// listAllSQL is the default list query: all tasks ordered by priority and created.
const listAllSQL = `SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`

// listFlags holds parsed list-specific flags.
type listFlags struct {
	ready   bool
	blocked bool
}

// parseListFlags parses list-specific flags from args.
func parseListFlags(args []string) listFlags {
	var flags listFlags
	for _, arg := range args {
		switch arg {
		case "--ready":
			flags.ready = true
		case "--blocked":
			flags.blocked = true
		}
	}
	return flags
}

// runList implements the `tick list` command.
// It queries tasks ordered by priority ASC then created ASC and displays them in aligned columns.
// When --ready is set (or invoked via `tick ready`), it filters to ready tasks only.
func (a *App) runList(args []string) error {
	flags := parseListFlags(args)

	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	type listRow struct {
		ID       string
		Status   string
		Priority int
		Title    string
	}

	querySQL := listAllSQL
	if flags.ready {
		querySQL = ReadySQL
	} else if flags.blocked {
		querySQL = BlockedSQL
	}

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(querySQL)
		if err != nil {
			return fmt.Errorf("failed to query tasks: %w", err)
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

	if len(rows) == 0 {
		fmt.Fprintln(a.stdout, "No tasks found.")
		return nil
	}

	if a.config.Quiet {
		for _, r := range rows {
			fmt.Fprintln(a.stdout, r.ID)
		}
		return nil
	}

	// Column widths: ID (12), STATUS (12), PRI (4), TITLE (remainder)
	fmt.Fprintf(a.stdout, "%-12s%-12s%-4s%s\n", "ID", "STATUS", "PRI", "TITLE")
	for _, r := range rows {
		fmt.Fprintf(a.stdout, "%-12s%-12s%-4d%s\n", r.ID, r.Status, r.Priority, r.Title)
	}

	return nil
}
