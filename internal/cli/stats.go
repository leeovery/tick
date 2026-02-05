package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
)

// queryStats queries SQLite for aggregate counts by status, priority, and workflow state.
// Ready/blocked counts reuse ReadyCondition and BlockedCondition from Phase 3.
// All 5 priority levels (P0-P4) are always returned, even with zero counts.
func queryStats(db *sql.DB) (*StatsData, error) {
	data := &StatsData{}

	// Count total tasks
	err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&data.Total)
	if err != nil {
		return nil, err
	}

	// Count by status
	statusQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0)
		FROM tasks
	`
	err = db.QueryRow(statusQuery).Scan(&data.Open, &data.InProgress, &data.Done, &data.Cancelled)
	if err != nil {
		return nil, err
	}

	// Count ready tasks (reuse ReadyCondition)
	readyQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + ReadyCondition
	err = db.QueryRow(readyQuery).Scan(&data.Ready)
	if err != nil {
		return nil, err
	}

	// Count blocked tasks (reuse BlockedCondition)
	blockedQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + BlockedCondition
	err = db.QueryRow(blockedQuery).Scan(&data.Blocked)
	if err != nil {
		return nil, err
	}

	// Count by priority (always 5 entries, P0-P4)
	data.ByPriority = make([]PriorityCount, 5)
	for i := 0; i <= 4; i++ {
		data.ByPriority[i] = PriorityCount{Priority: i, Count: 0}
	}

	priRows, err := db.Query("SELECT priority, COUNT(*) FROM tasks GROUP BY priority")
	if err != nil {
		return nil, err
	}
	defer priRows.Close()

	for priRows.Next() {
		var priority, count int
		if err := priRows.Scan(&priority, &count); err != nil {
			return nil, err
		}
		if priority >= 0 && priority <= 4 {
			data.ByPriority[priority].Count = count
		}
	}
	if err := priRows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// runStats executes the stats subcommand.
// Displays aggregate counts by status, priority, and workflow state (ready/blocked).
func (a *App) runStats() int {
	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Open store
	a.WriteVerbose("store open %s", tickDir)
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	var stats *StatsData

	// Query stats from SQLite
	a.WriteVerbose("lock acquire shared")
	a.WriteVerbose("cache freshness check")
	err = store.Query(func(db *sql.DB) error {
		var queryErr error
		stats, queryErr = queryStats(db)
		return queryErr
	})
	a.WriteVerbose("lock release")

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output
	if !a.formatConfig.Quiet {
		formatter := a.formatConfig.Formatter()
		fmt.Fprint(a.Stdout, formatter.FormatStats(stats))
	}

	return 0
}
