package cli

import (
	"database/sql"
	"fmt"
)

// runStats implements the `tick stats` command.
// It queries the database for counts by status, priority, and workflow state
// (ready/blocked) and passes the result to the formatter.
func (a *App) runStats() error {
	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	store, err := a.newStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var stats StatsData

	err = store.Query(func(db *sql.DB) error {
		// Count by status
		rows, err := db.Query(`SELECT status, COUNT(*) FROM tasks GROUP BY status`)
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
			return err
		}

		stats.Total = stats.Open + stats.InProgress + stats.Done + stats.Cancelled

		// Count by priority
		priRows, err := db.Query(`SELECT priority, COUNT(*) FROM tasks GROUP BY priority`)
		if err != nil {
			return fmt.Errorf("failed to query priority counts: %w", err)
		}
		defer priRows.Close()

		for priRows.Next() {
			var pri, count int
			if err := priRows.Scan(&pri, &count); err != nil {
				return fmt.Errorf("failed to scan priority count: %w", err)
			}
			if pri >= 0 && pri <= 4 {
				stats.ByPriority[pri] = count
			}
		}
		if err := priRows.Err(); err != nil {
			return err
		}

		// Count ready tasks (reuses readyWhere from list.go)
		var readyCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + readyWhere).Scan(&readyCount)
		if err != nil {
			return fmt.Errorf("failed to query ready count: %w", err)
		}
		stats.Ready = readyCount

		// Count blocked tasks (reuses blockedWhere from list.go)
		var blockedCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE ` + blockedWhere).Scan(&blockedCount)
		if err != nil {
			return fmt.Errorf("failed to query blocked count: %w", err)
		}
		stats.Blocked = blockedCount

		return nil
	})
	if err != nil {
		return err
	}

	if a.config.Quiet {
		return nil
	}

	return a.formatter.FormatStats(a.stdout, &stats)
}
