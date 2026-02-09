package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/engine"
)

// StatsQuery counts tasks grouped by status. It returns one row per status
// value with the count for that status.
const StatsQuery = `
SELECT status, COUNT(*) as cnt
FROM tasks
GROUP BY status
`

// StatsPriorityQuery counts tasks grouped by priority. It returns one row
// per priority value with the count for that priority.
const StatsPriorityQuery = `
SELECT priority, COUNT(*) as cnt
FROM tasks
GROUP BY priority
`

// StatsReadyCountQuery counts open tasks matching the ready criteria:
// status is open, all blockers closed, no open children.
const StatsReadyCountQuery = `
SELECT COUNT(*) FROM tasks t
WHERE t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )
`

// StatsBlockedCountQuery counts open tasks that are NOT ready: they have an
// unclosed blocker or open/in_progress children.
const StatsBlockedCountQuery = `
SELECT COUNT(*) FROM tasks t
WHERE t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON blocker.id = d.blocked_by
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )
`

// runStats implements the "tick stats" command. It queries SQLite for counts
// grouped by status, priority, and workflow state, then formats via the
// Formatter interface.
func runStats(ctx *Context) error {
	if ctx.Quiet {
		return nil
	}

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
	if err != nil {
		return err
	}
	defer store.Close()

	var data StatsData

	err = store.Query(func(db *sql.DB) error {
		// Count by status.
		if err := queryStatusCounts(db, &data); err != nil {
			return err
		}

		// Count by priority.
		if err := queryPriorityCounts(db, &data); err != nil {
			return err
		}

		// Count ready tasks.
		if err := db.QueryRow(StatsReadyCountQuery).Scan(&data.Ready); err != nil {
			return fmt.Errorf("counting ready tasks: %w", err)
		}

		// Count blocked tasks.
		if err := db.QueryRow(StatsBlockedCountQuery).Scan(&data.Blocked); err != nil {
			return fmt.Errorf("counting blocked tasks: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.Fmt.FormatStats(ctx.Stdout, &data)
}

// queryStatusCounts populates the StatsData status fields and Total from
// the database.
func queryStatusCounts(db *sql.DB, data *StatsData) error {
	rows, err := db.Query(StatsQuery)
	if err != nil {
		return fmt.Errorf("querying status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return fmt.Errorf("scanning status count: %w", err)
		}
		data.Total += count
		switch status {
		case "open":
			data.Open = count
		case "in_progress":
			data.InProgress = count
		case "done":
			data.Done = count
		case "cancelled":
			data.Cancelled = count
		}
	}
	return rows.Err()
}

// queryPriorityCounts populates the StatsData ByPriority array from
// the database. Priority values outside 0-4 are ignored.
func queryPriorityCounts(db *sql.DB, data *StatsData) error {
	rows, err := db.Query(StatsPriorityQuery)
	if err != nil {
		return fmt.Errorf("querying priority counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var priority, count int
		if err := rows.Scan(&priority, &count); err != nil {
			return fmt.Errorf("scanning priority count: %w", err)
		}
		if priority >= 0 && priority <= 4 {
			data.ByPriority[priority] = count
		}
	}
	return rows.Err()
}
