package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// runList executes the list subcommand.
func (a *App) runList(args []string) int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	type taskRow struct {
		ID       string
		Title    string
		Status   string
		Priority int
	}

	var tasks []taskRow

	// Query tasks from SQLite, ordered by priority ASC (highest first), then created ASC (oldest first)
	err = store.Query(func(db *sql.DB) error {
		rows, err := db.Query(`SELECT id, title, status, priority FROM tasks ORDER BY priority ASC, created ASC`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var t taskRow
			if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
				return err
			}
			tasks = append(tasks, t)
		}
		return rows.Err()
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Handle empty result
	if len(tasks) == 0 {
		fmt.Fprintln(a.Stdout, "No tasks found.")
		return 0
	}

	// Output
	if a.flags.Quiet {
		// --quiet: output only task IDs, one per line
		for _, t := range tasks {
			fmt.Fprintln(a.Stdout, t.ID)
		}
	} else {
		// Aligned columns: ID (12), STATUS (12), PRI (4), TITLE (remainder)
		fmt.Fprintf(a.Stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
		for _, t := range tasks {
			fmt.Fprintf(a.Stdout, "%-12s %-12s %-4d %s\n", t.ID, t.Status, t.Priority, t.Title)
		}
	}

	return 0
}
