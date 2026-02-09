package cli

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/engine"
)

// listRow holds a single row of list output data.
type listRow struct {
	id       string
	status   string
	priority int
	title    string
}

// runList implements the "tick list" command. It queries all tasks from the
// SQLite cache ordered by priority ASC then created ASC, and displays them
// in aligned columns.
func runList(ctx *Context) error {
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
		sqlRows, err := db.Query(
			`SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`,
		)
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
