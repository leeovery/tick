package cli

import (
	"database/sql"
	"fmt"
)

// runStats implements the `tick stats` command.
// It queries SQLite for counts grouped by status, priority, and workflow state
// (ready/blocked), then formats the result via the Formatter interface.
func (a *App) runStats(args []string) error {
	if a.Quiet {
		return nil
	}

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var stats StatsData

	err = s.Query(func(db *sql.DB) error {
		// Count by status
		if err := queryStatusCounts(db, &stats); err != nil {
			return err
		}

		// Count by priority (all 5 levels)
		if err := queryPriorityCounts(db, &stats); err != nil {
			return err
		}

		// Count ready and blocked (reusing ready query logic)
		if err := queryWorkflowCounts(db, &stats); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return a.Formatter.FormatStats(a.Stdout, stats)
}

// queryStatusCounts populates the status breakdown fields in StatsData.
func queryStatusCounts(db *sql.DB, stats *StatsData) error {
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
		stats.Total += count
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
	return rows.Err()
}

// queryPriorityCounts populates the ByPriority array in StatsData.
// Always produces 5 entries (P0-P4), even if some have zero count.
func queryPriorityCounts(db *sql.DB, stats *StatsData) error {
	rows, err := db.Query("SELECT priority, COUNT(*) FROM tasks GROUP BY priority")
	if err != nil {
		return fmt.Errorf("failed to query priority counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var priority, count int
		if err := rows.Scan(&priority, &count); err != nil {
			return fmt.Errorf("failed to scan priority count: %w", err)
		}
		if priority >= 0 && priority <= 4 {
			stats.ByPriority[priority] = count
		}
	}
	return rows.Err()
}

// queryWorkflowCounts populates Ready and Blocked counts in StatsData.
// Ready = open tasks that pass all ready conditions (unblocked, no open children).
// Blocked = open tasks that do NOT pass ready conditions.
// This reuses readyConditionsFor() from ready.go so the semantics stay in sync.
func queryWorkflowCounts(db *sql.DB, stats *StatsData) error {
	readyCountQuery := `SELECT COUNT(*) FROM tasks t WHERE t.status = 'open' AND` + readyConditionsFor("t")

	var readyCount int
	if err := db.QueryRow(readyCountQuery).Scan(&readyCount); err != nil {
		return fmt.Errorf("failed to query ready count: %w", err)
	}
	stats.Ready = readyCount
	stats.Blocked = stats.Open - readyCount

	return nil
}
