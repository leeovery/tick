package cli

import (
	"database/sql"
	"fmt"
)

// readyQuery is the SQL for tasks that are open, unblocked, and have no open children.
const readyQuery = `
SELECT t.id, t.title, t.status, t.priority
FROM tasks t
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
ORDER BY t.priority ASC, t.created ASC
`

func (a *App) cmdReady(workDir string, args []string) error {
	return a.cmdListFiltered(workDir, readyQuery)
}

// cmdListFiltered runs a filtered list query and outputs results.
func (a *App) cmdListFiltered(workDir string, query string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := a.openStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	type taskRow struct {
		ID       string
		Title    string
		Status   string
		Priority int
	}

	var tasks []taskRow

	err = store.Query(func(db *sql.DB) error {
		rows, err := db.Query(query)
		if err != nil {
			return fmt.Errorf("querying tasks: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var t taskRow
			if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
				return fmt.Errorf("scanning task: %w", err)
			}
			tasks = append(tasks, t)
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}

	if a.opts.Quiet {
		if len(tasks) == 0 {
			return nil
		}
		for _, t := range tasks {
			fmt.Fprintln(a.stdout, t.ID)
		}
		return nil
	}

	items := make([]TaskListItem, len(tasks))
	for i, t := range tasks {
		items[i] = TaskListItem{ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority}
	}
	return a.fmtr.FormatTaskList(a.stdout, items)
}
