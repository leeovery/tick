package cli

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/store"
)

// listRow represents a single task row in list/ready output.
type listRow struct {
	ID       string
	Status   string
	Priority int
	Title    string
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

// runList implements the `tick list` command.
// It queries all tasks from SQLite, ordered by priority ASC then created ASC,
// and displays them as aligned columns.
func (a *App) runList(args []string) error {
	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var rows []listRow

	err = s.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(
			"SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC",
		)
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
