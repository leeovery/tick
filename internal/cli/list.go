package cli

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/leeovery/tick/internal/storage"
)

// readyWhere is the WHERE clause fragment for ready tasks: open, all blockers closed, no open children.
const readyWhere = `status = 'open'
  AND id NOT IN (
    SELECT d.task_id FROM dependencies d
    JOIN tasks t ON d.blocked_by = t.id
    WHERE t.status NOT IN ('done', 'cancelled')
  )
  AND id NOT IN (
    SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
  )`

// blockedWhere is the WHERE clause fragment for blocked tasks: open tasks that are NOT ready.
const blockedWhere = `status = 'open'
  AND (
    id IN (
      SELECT d.task_id FROM dependencies d
      JOIN tasks t ON d.blocked_by = t.id
      WHERE t.status NOT IN ('done', 'cancelled')
    )
    OR id IN (
      SELECT parent FROM tasks WHERE parent IS NOT NULL AND status IN ('open', 'in_progress')
    )
  )`

// ReadySQL is the SQL query for ready tasks: open, all blockers closed, no open children.
// Exported so it can be reused by other queries (e.g., blocked query, list filters).
const ReadySQL = `SELECT id, status, priority, title FROM tasks
WHERE ` + readyWhere + `
ORDER BY priority ASC, created ASC`

// BlockedSQL is the SQL query for blocked tasks: open tasks that are NOT ready.
// A task is blocked if it is open AND has unclosed blockers OR has open children.
// This is the inverse of ReadySQL — reuses the same subqueries.
const BlockedSQL = `SELECT id, status, priority, title FROM tasks
WHERE ` + blockedWhere + `
ORDER BY priority ASC, created ASC`

// listAllSQL is the default list query: all tasks ordered by priority and created.
const listAllSQL = `SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`

// validStatuses lists all valid status values for the --status flag.
var validStatuses = map[string]bool{
	"open":        true,
	"in_progress": true,
	"done":        true,
	"cancelled":   true,
}

// listFlags holds parsed list-specific flags.
type listFlags struct {
	ready    bool
	blocked  bool
	status   string
	priority int
	hasPri   bool // true when --priority was explicitly set
}

// parseListFlags parses list-specific flags from args and validates them.
func parseListFlags(args []string) (listFlags, error) {
	var flags listFlags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ready":
			flags.ready = true
		case "--blocked":
			flags.blocked = true
		case "--status":
			if i+1 >= len(args) {
				return flags, fmt.Errorf("--status requires a value (open, in_progress, done, cancelled)")
			}
			i++
			flags.status = args[i]
		case "--priority":
			if i+1 >= len(args) {
				return flags, fmt.Errorf("--priority requires a value (0-4)")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return flags, fmt.Errorf("--priority must be a number (0-4)")
			}
			flags.priority = p
			flags.hasPri = true
		}
	}

	// Validate mutual exclusion
	if flags.ready && flags.blocked {
		return flags, fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	// Validate status value
	if flags.status != "" && !validStatuses[flags.status] {
		return flags, fmt.Errorf("invalid status '%s' — valid values: open, in_progress, done, cancelled", flags.status)
	}

	// Validate priority range
	if flags.hasPri && (flags.priority < 0 || flags.priority > 4) {
		return flags, fmt.Errorf("invalid priority %d — valid range: 0-4", flags.priority)
	}

	return flags, nil
}

// buildListQuery constructs the SQL query from the parsed flags.
// It reuses readyWhere/blockedWhere fragments and adds additional WHERE clauses for --status and --priority.
func buildListQuery(flags listFlags) (string, []interface{}) {
	var where []string
	var queryArgs []interface{}

	// Base filter: --ready or --blocked
	if flags.ready {
		where = append(where, "("+readyWhere+")")
	} else if flags.blocked {
		where = append(where, "("+blockedWhere+")")
	}

	// Additional filter: --status
	if flags.status != "" {
		where = append(where, "status = ?")
		queryArgs = append(queryArgs, flags.status)
	}

	// Additional filter: --priority
	if flags.hasPri {
		where = append(where, "priority = ?")
		queryArgs = append(queryArgs, flags.priority)
	}

	q := "SELECT id, status, priority, title FROM tasks"
	if len(where) > 0 {
		q += "\nWHERE "
		for i, w := range where {
			if i > 0 {
				q += "\n  AND "
			}
			q += w
		}
	}
	q += "\nORDER BY priority ASC, created ASC"

	return q, queryArgs
}

// runList implements the `tick list` command.
// It queries tasks ordered by priority ASC then created ASC and displays them in aligned columns.
// Supports --ready, --blocked, --status, and --priority flags, combined with AND logic.
func (a *App) runList(args []string) error {
	flags, err := parseListFlags(args)
	if err != nil {
		return err
	}

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

	querySQL, queryArgs := buildListQuery(flags)

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(querySQL, queryArgs...)
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
