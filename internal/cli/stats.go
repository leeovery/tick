package cli

import (
	"database/sql"
	"fmt"
	"io"
)

// RunStats executes the stats command: queries aggregate counts by status, priority,
// and workflow state (ready/blocked), then outputs via the Formatter interface.
func RunStats(dir string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
	if fc.Quiet {
		return nil
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	var stats Stats

	err = store.Query(func(db *sql.DB) error {
		// Total count.
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&stats.Total); err != nil {
			return fmt.Errorf("failed to query total count: %w", err)
		}

		// Counts by status.
		rows, err := db.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
		if err != nil {
			return fmt.Errorf("failed to query status counts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err != nil {
				return fmt.Errorf("failed to scan status count: %w", err)
			}
			switch status {
			case "open":
				stats.Open = count
			case "in_progress":
				stats.InProgress = count
			case "done":
				stats.Done = count
			case "cancelled":
				stats.Cancelled = count
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to iterate status counts: %w", err)
		}

		// Counts by priority.
		priRows, err := db.Query("SELECT priority, COUNT(*) FROM tasks GROUP BY priority")
		if err != nil {
			return fmt.Errorf("failed to query priority counts: %w", err)
		}
		defer priRows.Close()

		for priRows.Next() {
			var priority, count int
			if err := priRows.Scan(&priority, &count); err != nil {
				return fmt.Errorf("failed to scan priority count: %w", err)
			}
			if priority >= 0 && priority <= 4 {
				stats.ByPriority[priority] = count
			}
		}
		if err := priRows.Err(); err != nil {
			return fmt.Errorf("failed to iterate priority counts: %w", err)
		}

		// Ready count: open, no unclosed blockers, no open children.
		readyQuery := "\n\t\t\tSELECT COUNT(*) FROM tasks t\n\t\t\tWHERE " + ReadyWhereClause()
		if err := db.QueryRow(readyQuery).Scan(&stats.Ready); err != nil {
			return fmt.Errorf("failed to query ready count: %w", err)
		}

		// Blocked count: open AND NOT ready (derived from ready).
		stats.Blocked = stats.Open - stats.Ready

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, fmtr.FormatStats(stats))
	return nil
}
