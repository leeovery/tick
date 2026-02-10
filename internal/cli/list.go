package cli

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
)

// RunList executes the list command: queries all tasks from SQLite and outputs them
// in aligned columns ordered by priority ASC, then created ASC.
func RunList(dir string, quiet bool, stdout io.Writer) error {
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
		sqlRows, err := db.Query(
			`SELECT id, status, priority, title FROM tasks ORDER BY priority ASC, created ASC`,
		)
		if err != nil {
			return fmt.Errorf("failed to query tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
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
		fmt.Fprintln(stdout, "No tasks found.")
		return nil
	}

	if quiet {
		for _, r := range rows {
			fmt.Fprintln(stdout, r.id)
		}
		return nil
	}

	// Print header
	fmt.Fprintf(stdout, "%-12s%-13s%-5s%s\n", "ID", "STATUS", "PRI", "TITLE")

	// Print rows
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-12s%-13s%-5d%s\n", r.id, r.status, r.priority, r.title)
	}

	return nil
}
