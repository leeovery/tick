package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// ListFilter holds parsed filter flags for the list command.
type ListFilter struct {
	Ready    bool
	Blocked  bool
	Status   string
	Priority int
	// HasPriority indicates whether --priority was explicitly set.
	HasPriority bool
	// Parent restricts results to descendants of the specified task ID.
	Parent string
}

// parseListFlags parses list-specific flags from subArgs.
// Returns the parsed filter and an error if validation fails.
func parseListFlags(args []string) (ListFilter, error) {
	var f ListFilter
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ready":
			f.Ready = true
		case "--blocked":
			f.Blocked = true
		case "--status":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--status requires a value")
			}
			i++
			f.Status = args[i]
		case "--priority":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return f, fmt.Errorf("invalid priority '%s': must be 0-4", args[i])
			}
			f.Priority = p
			f.HasPriority = true
		case "--parent":
			if i+1 >= len(args) {
				return f, fmt.Errorf("--parent requires a value")
			}
			i++
			f.Parent = task.NormalizeID(args[i])
		}
	}

	if f.Ready && f.Blocked {
		return f, fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	if f.Status != "" {
		valid := map[string]bool{
			string(task.StatusOpen):       true,
			string(task.StatusInProgress): true,
			string(task.StatusDone):       true,
			string(task.StatusCancelled):  true,
		}
		if !valid[f.Status] {
			return f, fmt.Errorf("invalid status '%s': must be one of open, in_progress, done, cancelled", f.Status)
		}
	}

	if f.HasPriority {
		if f.Priority < 0 || f.Priority > 4 {
			return f, fmt.Errorf("invalid priority '%d': must be 0-4", f.Priority)
		}
	}

	return f, nil
}

// RunList executes the list command: queries tasks from SQLite with optional filters
// and outputs them via the Formatter, ordered by priority ASC, then created ASC.
func RunList(dir string, fc FormatConfig, fmtr Formatter, filter ListFilter, stdout io.Writer) error {
	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir, storeOpts(fc)...)
	if err != nil {
		return err
	}
	defer store.Close()

	type listRow struct {
		id       string
		status   string
		priority int
		title    string
	}

	var rows []listRow

	err = store.Query(func(db *sql.DB) error {
		var descendantIDs []string

		if filter.Parent != "" {
			// Validate parent exists.
			var exists int
			err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", filter.Parent).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check parent task: %w", err)
			}
			if exists == 0 {
				return fmt.Errorf("task '%s' not found", filter.Parent)
			}

			// Collect descendant IDs via recursive CTE.
			descendantIDs, err = queryDescendantIDs(db, filter.Parent)
			if err != nil {
				return err
			}
		}

		query, queryArgs := buildListQuery(filter, descendantIDs)

		sqlRows, err := db.Query(query, queryArgs...)
		if err != nil {
			return fmt.Errorf("failed to query tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.id, &r.status, &r.priority, &r.title); err != nil {
				return fmt.Errorf("failed to scan task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	// Convert rows to task.Task slice for the formatter.
	tasks := make([]task.Task, len(rows))
	for i, r := range rows {
		tasks[i] = task.Task{
			ID:       r.id,
			Title:    r.title,
			Status:   task.Status(r.status),
			Priority: r.priority,
		}
	}

	if fc.Quiet {
		for _, t := range tasks {
			fmt.Fprintln(stdout, t.ID)
		}
		return nil
	}

	fmt.Fprintln(stdout, fmtr.FormatTaskList(tasks))
	return nil
}

// queryDescendantIDs executes a recursive CTE to collect all descendant task IDs
// of the given parent ID. The parent itself is excluded from results.
func queryDescendantIDs(db *sql.DB, parentID string) ([]string, error) {
	const descendantCTE = `
		WITH RECURSIVE descendants(id) AS (
			SELECT id FROM tasks WHERE parent = ?
			UNION ALL
			SELECT t.id FROM tasks t
			JOIN descendants d ON t.parent = d.id
		)
		SELECT id FROM descendants`

	rows, err := db.Query(descendantCTE, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query descendants: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan descendant ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// buildListQuery composes a SQL query string and args based on the filter.
// When descendantIDs is non-empty, results are restricted to those IDs.
func buildListQuery(f ListFilter, descendantIDs []string) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if f.Ready {
		// Reuse the ready conditions: open, no unclosed blockers, no open children.
		conditions = append(conditions,
			`t.status = 'open'`,
			`NOT EXISTS (
				SELECT 1 FROM dependencies d
				JOIN tasks blocker ON blocker.id = d.blocked_by
				WHERE d.task_id = t.id
				  AND blocker.status NOT IN ('done', 'cancelled')
			)`,
			`NOT EXISTS (
				SELECT 1 FROM tasks child
				WHERE child.parent = t.id
				  AND child.status IN ('open', 'in_progress')
			)`,
		)
	}

	if f.Blocked {
		// Reuse the blocked conditions: open AND (unclosed blocker OR open children).
		conditions = append(conditions,
			`t.status = 'open'`,
			`(
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
			)`,
		)
	}

	if f.Status != "" {
		conditions = append(conditions, `t.status = ?`)
		args = append(args, f.Status)
	}

	if f.HasPriority {
		conditions = append(conditions, `t.priority = ?`)
		args = append(args, f.Priority)
	}

	if len(descendantIDs) > 0 {
		placeholders := make([]string, len(descendantIDs))
		for i, id := range descendantIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, `t.id IN (`+strings.Join(placeholders, ",")+`)`)
	} else if f.Parent != "" {
		// Parent exists but has no descendants: use impossible condition.
		conditions = append(conditions, `1 = 0`)
	}

	query := `SELECT t.id, t.status, t.priority, t.title FROM tasks t`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY t.priority ASC, t.created ASC"

	return query, args
}
