package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// runList implements the `tick list` command.
// It queries all tasks ordered by priority ASC then created ASC and displays them in aligned columns.
func (a *App) runList() error {
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

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
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
