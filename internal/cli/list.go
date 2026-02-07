package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strconv"

	"github.com/leeovery/tick/internal/store"
)

// listRow represents a single task row in list/ready output.
type listRow struct {
	ID       string
	Status   string
	Priority int
	Title    string
}

// listFlags holds parsed flags for the list command.
type listFlags struct {
	ready    bool
	blocked  bool
	status   string
	priority int
	hasPri   bool // distinguishes "not set" from "set to 0"
}

// validStatuses lists the allowed values for the --status flag.
var validStatuses = []string{"open", "in_progress", "done", "cancelled"}

// parseListFlags extracts list-specific flags from args.
// Returns the parsed flags or an error for invalid/conflicting values.
func parseListFlags(args []string) (listFlags, error) {
	var f listFlags
	f.priority = -1 // sentinel: not set

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
			valid := false
			for _, s := range validStatuses {
				if f.status == s {
					valid = true
					break
				}
			}
			if !valid {
				return f, fmt.Errorf("invalid status %q; valid values: open, in_progress, done, cancelled", f.status)
			}
		case "--priority":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--priority requires a value (0-4)")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil || p < 0 || p > 4 {
				return f, fmt.Errorf("invalid priority %q; valid values: 0, 1, 2, 3, 4", args[i])
			}
			f.priority = p
			f.hasPri = true
		default:
			return f, fmt.Errorf("unknown flag %q for list command", args[i])
		}
	}

	if f.ready && f.blocked {
		return f, fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	return f, nil
}

// renderListOutput writes task rows to stdout as aligned columns.
// In quiet mode it outputs only task IDs. When no rows exist it prints "No tasks found.".
func renderListOutput(rows []listRow, stdout io.Writer, quiet bool) error {
	if len(rows) == 0 {
		fmt.Fprintln(stdout, "No tasks found.")
		return nil
	}

	if quiet {
		for _, r := range rows {
			fmt.Fprintln(stdout, r.ID)
		}
		return nil
	}

	// Aligned columns: ID (12), STATUS (12), PRI (4), TITLE (remainder)
	fmt.Fprintf(stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-12s %-12s %-4d %s\n", r.ID, r.Status, r.Priority, r.Title)
	}

	return nil
}

// buildListQuery constructs the SQL query and parameters for the list command
// based on the provided flags. It reuses readyQuery and blockedQuery for
// --ready and --blocked flags respectively.
func buildListQuery(f listFlags) (string, []interface{}) {
	if f.ready && !f.hasPri && f.status == "" {
		return readyQuery, nil
	}
	if f.blocked && !f.hasPri && f.status == "" {
		return blockedQuery, nil
	}

	// Build dynamic query with conditions
	query := "SELECT t.id, t.status, t.priority, t.title FROM tasks t WHERE 1=1"
	var params []interface{}

	if f.ready {
		query += " AND t.status = 'open' AND" + readyConditionsFor("t")
	}
	if f.blocked {
		query += " AND t.status = 'open' AND t.id NOT IN (SELECT t2.id FROM tasks t2 WHERE t2.status = 'open' AND" +
			readyConditionsFor("t2") + ")"
	}
	if f.status != "" {
		query += " AND t.status = ?"
		params = append(params, f.status)
	}
	if f.hasPri {
		query += " AND t.priority = ?"
		params = append(params, f.priority)
	}

	query += " ORDER BY t.priority ASC, t.created ASC"
	return query, params
}

// runList implements the `tick list` command.
// It queries tasks from SQLite with optional filters (--ready, --blocked,
// --status, --priority), ordered by priority ASC then created ASC,
// and displays them as aligned columns.
func (a *App) runList(args []string) error {
	flags, err := parseListFlags(args)
	if err != nil {
		return err
	}

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	query, params := buildListQuery(flags)

	var rows []listRow

	err = s.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(query, params...)
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

	return renderListOutput(rows, a.Stdout, a.Quiet)
}
