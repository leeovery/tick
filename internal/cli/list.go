package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/store"
)

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

	type listRow struct {
		ID       string
		Status   string
		Priority int
		Title    string
	}

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

	// Empty result
	if len(rows) == 0 {
		fmt.Fprintln(a.Stdout, "No tasks found.")
		return nil
	}

	// Quiet mode: only IDs
	if a.Quiet {
		for _, r := range rows {
			fmt.Fprintln(a.Stdout, r.ID)
		}
		return nil
	}

	// Aligned columns: ID (12), STATUS (12), PRI (4), TITLE (remainder)
	fmt.Fprintf(a.Stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
	for _, r := range rows {
		fmt.Fprintf(a.Stdout, "%-12s %-12s %-4d %s\n", r.ID, r.Status, r.Priority, r.Title)
	}

	return nil
}
