package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strconv"

	"github.com/leeovery/tick/internal/engine"
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
// based on the provided filters. When no filters are set, returns a query
// for all tasks.
func buildListQuery(f listFilters) (string, []interface{}) {
	if f.ready {
		return buildReadyFilterQuery(f)
	}
	if f.blocked {
		return buildBlockedFilterQuery(f)
	}
	return buildSimpleFilterQuery(f)
}

// buildReadyFilterQuery wraps ReadyQuery with additional AND filters for
// status and priority. Ordering is provided by the inner ReadyQuery (priority
// ASC, created ASC) and preserved by the outer select.
func buildReadyFilterQuery(f listFilters) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM (` + ReadyQuery + `) AS ready WHERE 1=1`
	var params []interface{}

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

// buildBlockedFilterQuery wraps BlockedQuery with additional AND filters.
// Ordering is provided by the inner BlockedQuery (priority ASC, created ASC)
// and preserved by the outer select.
func buildBlockedFilterQuery(f listFilters) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM (` + BlockedQuery + `) AS blocked WHERE 1=1`
	var params []interface{}

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
func buildSimpleFilterQuery(f listFilters) (string, []interface{}) {
	q := `SELECT id, status, priority, title FROM tasks WHERE 1=1`
	var params []interface{}

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

// runList implements the "tick list" command. It parses filter flags (--ready,
// --blocked, --status, --priority), builds the appropriate SQL query, and
// displays results in aligned columns.
func runList(ctx *Context) error {
	filters, err := parseListFlags(ctx.Args)
	if err != nil {
		return err
	}

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	query, params := buildListQuery(filters)
	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
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

// printListTable prints tasks in aligned columns: ID (12), STATUS (12), PRI (4), TITLE.
func printListTable(w io.Writer, rows []listRow) {
	fmt.Fprintf(w, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
	for _, r := range rows {
		fmt.Fprintf(w, "%-12s %-12s %-4d %s\n", r.id, r.status, r.priority, r.title)
	}
}
