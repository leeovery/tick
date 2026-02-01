package cli

import (
	"database/sql"
	"fmt"
)

func (a *App) cmdStats(workDir string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var data StatsData

	err = store.Query(func(db *sql.DB) error {
		// Count by status.
		rows, err := db.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
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
		if err := rows.Err(); err != nil {
			return err
		}

		data.Total = data.Open + data.InProgress + data.Done + data.Cancelled

		// Count by priority (P0-P4).
		pRows, err := db.Query("SELECT priority, COUNT(*) FROM tasks GROUP BY priority")
		if err != nil {
			return fmt.Errorf("querying priority counts: %w", err)
		}
		defer pRows.Close()
		for pRows.Next() {
			var priority, count int
			if err := pRows.Scan(&priority, &count); err != nil {
				return fmt.Errorf("scanning priority count: %w", err)
			}
			if priority >= 0 && priority <= 4 {
				data.ByPriority[priority] = count
			}
		}
		if err := pRows.Err(); err != nil {
			return err
		}

		// Ready count — reuse ready query logic.
		err = db.QueryRow(`
			SELECT COUNT(*) FROM tasks t
			WHERE t.status = 'open'
			  AND NOT EXISTS (
			    SELECT 1 FROM dependencies d
			    JOIN tasks blocker ON d.blocked_by = blocker.id
			    WHERE d.task_id = t.id
			      AND blocker.status NOT IN ('done', 'cancelled')
			  )
			  AND NOT EXISTS (
			    SELECT 1 FROM tasks child
			    WHERE child.parent = t.id
			      AND child.status IN ('open', 'in_progress')
			  )
		`).Scan(&data.Ready)
		if err != nil {
			return fmt.Errorf("querying ready count: %w", err)
		}

		// Blocked = open tasks that are not ready.
		data.Blocked = data.Open - data.Ready

		return nil
	})
	if err != nil {
		return err
	}

	if a.opts.Quiet {
		return nil
	}

	return a.fmtr.FormatStats(a.stdout, data)
}
