package cli

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// listRow holds a single row of list output data.
type listRow struct {
	id       string
	status   string
	priority int
	title    string
}

// listFilters holds the parsed filter flags for the list command.
type listFilters struct {
	ready    bool
	blocked  bool
	status   string
	priority int
	hasPri   bool // true when --priority was explicitly set
	parent   string
}

// validStatuses lists the accepted values for --status.
var validStatuses = []string{"open", "in_progress", "done", "cancelled"}

// parseListFlags parses list subcommand flags from ctx.Args. Returns an error
// for invalid or mutually exclusive flags.
func parseListFlags(args []string) (listFilters, error) {
	var f listFilters
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ready":
			f.ready = true
		case "--blocked":
			f.blocked = true
		case "--status":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--status requires a value (open, in_progress, done, cancelled)")
			}
			i++
			f.status = args[i]
			if !isValidStatus(f.status) {
				return f, fmt.Errorf("invalid status %q - valid values: open, in_progress, done, cancelled", f.status)
			}
		case "--priority":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--priority requires a value (0-4)")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return f, fmt.Errorf("invalid priority %q - must be a number 0-4", args[i])
			}
			if p < 0 || p > 4 {
				return f, fmt.Errorf("invalid priority %d - valid values: 0, 1, 2, 3, 4", p)
			}
			f.priority = p
			f.hasPri = true
		case "--parent":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--parent requires a task ID")
			}
			i++
			f.parent = task.NormalizeID(args[i])
		default:
			return f, fmt.Errorf("unknown list flag %q", args[i])
		}
	}

	if f.ready && f.blocked {
		return f, fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	return f, nil
}

// isValidStatus checks whether s is a recognized task status value.
func isValidStatus(s string) bool {
	for _, v := range validStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// buildListQuery returns the SQL query and parameters for the list command
// based on the provided filters. When descendantIDs is non-nil, results are
// restricted to tasks whose ID is in that set (the --parent pre-filter).
func buildListQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	if f.ready {
		return buildReadyFilterQuery(f, descendantIDs)
	}
	if f.blocked {
		return buildBlockedFilterQuery(f, descendantIDs)
	}
	return buildSimpleFilterQuery(f, descendantIDs)
}

// buildReadyFilterQuery wraps ReadyQuery with additional AND filters for
// status and priority.
func buildReadyFilterQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	return buildWrappedFilterQuery(ReadyQuery, "ready", f, descendantIDs)
}

// buildBlockedFilterQuery wraps BlockedQuery with additional AND filters.
func buildBlockedFilterQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	return buildWrappedFilterQuery(BlockedQuery, "blocked", f, descendantIDs)
}

// buildWrappedFilterQuery wraps an inner query (which provides ordering) with
// an outer SELECT and optional status, priority, and descendant filters.
func buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM (` + innerQuery + `) AS ` + alias + ` WHERE 1=1`
	var params []interface{}

	q, params = appendDescendantFilter(q, params, descendantIDs)

	if f.status != "" {
		q += ` AND status = ?`
		params = append(params, f.status)
	}
	if f.hasPri {
		q += ` AND priority = ?`
		params = append(params, f.priority)
	}

	return q, params
}

// buildSimpleFilterQuery builds a query for all tasks with optional status
// and priority filters, ordered by priority ASC then created ASC.
func buildSimpleFilterQuery(f listFilters, descendantIDs []string) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM tasks WHERE 1=1`
	var params []interface{}

	q, params = appendDescendantFilter(q, params, descendantIDs)

	if f.status != "" {
		q += ` AND status = ?`
		params = append(params, f.status)
	}
	if f.hasPri {
		q += ` AND priority = ?`
		params = append(params, f.priority)
	}

	q += ` ORDER BY priority ASC, created ASC`

	return q, params
}

// appendDescendantFilter adds an AND id IN (...) clause to restrict results to
// the given set of descendant IDs. If descendantIDs is nil, the query and params
// are returned unchanged.
func appendDescendantFilter(q string, params []interface{}, descendantIDs []string) (string, []interface{}) {
	if descendantIDs == nil {
		return q, params
	}
	placeholders := make([]string, len(descendantIDs))
	for i, id := range descendantIDs {
		placeholders[i] = "?"
		params = append(params, id)
	}
	q += ` AND id IN (` + strings.Join(placeholders, ",") + `)`
	return q, params
}

// queryDescendantIDs executes a recursive CTE to collect all descendant task IDs
// of the given parent. The parent itself is excluded from the result.
func queryDescendantIDs(db *sql.DB, parentID string) ([]string, error) {
	query := `
WITH RECURSIVE descendants(id) AS (
  SELECT id FROM tasks WHERE parent = ?
  UNION ALL
  SELECT t.id FROM tasks t
  JOIN descendants d ON t.parent = d.id
)
SELECT id FROM descendants`

	rows, err := db.Query(query, parentID)
	if err != nil {
		return nil, fmt.Errorf("querying descendants: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning descendant ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// parentTaskExists checks whether a task with the given ID exists in the database.
func parentTaskExists(db *sql.DB, id string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = ?`, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking parent task: %w", err)
	}
	return count > 0, nil
}

// runList implements the "tick list" command. It parses filter flags (--ready,
// --blocked, --status, --priority, --parent), builds the appropriate SQL query,
// and displays results in aligned columns.
func runList(ctx *Context) error {
	filters, err := parseListFlags(ctx.Args)
	if err != nil {
		return err
	}

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
	if err != nil {
		return err
	}
	defer store.Close()

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		// Resolve --parent pre-filter if set.
		var descendantIDs []string
		if filters.parent != "" {
			exists, err := parentTaskExists(db, filters.parent)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("Task '%s' not found", filters.parent)
			}
			descendantIDs, err = queryDescendantIDs(db, filters.parent)
			if err != nil {
				return err
			}
			// If no descendants, return early with empty result (not an error).
			if len(descendantIDs) == 0 {
				return nil
			}
		}

		query, params := buildListQuery(filters, descendantIDs)
		sqlRows, err := db.Query(query, params...)
		if err != nil {
			return fmt.Errorf("querying tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
				return fmt.Errorf("scanning task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	if ctx.Quiet {
		for _, r := range rows {
			fmt.Fprintln(ctx.Stdout, r.id)
		}
		return nil
	}

	taskRows := make([]TaskRow, len(rows))
	for i, r := range rows {
		taskRows[i] = TaskRow{
			ID:       r.id,
			Title:    r.title,
			Status:   r.status,
			Priority: r.priority,
		}
	}

	return ctx.Fmt.FormatTaskList(ctx.Stdout, taskRows)
}
