package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

func (a *App) cmdList(workDir string, args []string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
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
		rows, err := db.Query("SELECT id, title, status, priority FROM tasks ORDER BY priority ASC, created ASC")
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

	if len(tasks) == 0 {
		fmt.Fprintln(a.stdout, "No tasks found.")
		return nil
	}

	if a.opts.Quiet {
		for _, t := range tasks {
			fmt.Fprintln(a.stdout, t.ID)
		}
		return nil
	}

	// Aligned column output
	fmt.Fprintf(a.stdout, "%-12s %-12s %-4s %s\n", "ID", "STATUS", "PRI", "TITLE")
	for _, t := range tasks {
		fmt.Fprintf(a.stdout, "%-12s %-12s %-4d %s\n", t.ID, t.Status, t.Priority, t.Title)
	}

	_ = args
	return nil
}

func (a *App) cmdShow(workDir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick show <id>")
	}

	taskID := task.NormalizeID(args[0])

	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	type depInfo struct {
		ID     string
		Title  string
		Status string
	}

	type taskDetail struct {
		ID          string
		Title       string
		Status      string
		Priority    int
		Description string
		Parent      string
		Created     string
		Updated     string
		Closed      string
	}

	var td taskDetail
	var found bool
	var blockers []depInfo
	var children []depInfo
	var parentInfo *depInfo

	err = store.Query(func(db *sql.DB) error {
		// Query main task
		var description, parent, closed sql.NullString
		var created, updated string
		err := db.QueryRow(
			"SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id=?",
			taskID,
		).Scan(&td.ID, &td.Title, &td.Status, &td.Priority, &description, &parent, &created, &updated, &closed)

		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("querying task: %w", err)
		}
		found = true
		td.Created = created
		td.Updated = updated
		if description.Valid {
			td.Description = description.String
		}
		if parent.Valid {
			td.Parent = parent.String
		}
		if closed.Valid {
			td.Closed = closed.String
		}

		// Query blocked_by with context
		rows, err := db.Query(
			"SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id=?",
			taskID,
		)
		if err != nil {
			return fmt.Errorf("querying dependencies: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var d depInfo
			if err := rows.Scan(&d.ID, &d.Title, &d.Status); err != nil {
				return fmt.Errorf("scanning dependency: %w", err)
			}
			blockers = append(blockers, d)
		}

		// Query children
		childRows, err := db.Query(
			"SELECT id, title, status FROM tasks WHERE parent=?",
			taskID,
		)
		if err != nil {
			return fmt.Errorf("querying children: %w", err)
		}
		defer childRows.Close()
		for childRows.Next() {
			var c depInfo
			if err := childRows.Scan(&c.ID, &c.Title, &c.Status); err != nil {
				return fmt.Errorf("scanning child: %w", err)
			}
			children = append(children, c)
		}

		// Query parent context
		if td.Parent != "" {
			var p depInfo
			err := db.QueryRow("SELECT id, title, status FROM tasks WHERE id=?", td.Parent).
				Scan(&p.ID, &p.Title, &p.Status)
			if err == nil {
				parentInfo = &p
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("Task '%s' not found", taskID)
	}

	if a.opts.Quiet {
		fmt.Fprintln(a.stdout, td.ID)
		return nil
	}

	// Basic output format
	fmt.Fprintf(a.stdout, "ID:       %s\n", td.ID)
	fmt.Fprintf(a.stdout, "Title:    %s\n", td.Title)
	fmt.Fprintf(a.stdout, "Status:   %s\n", td.Status)
	fmt.Fprintf(a.stdout, "Priority: %d\n", td.Priority)
	if parentInfo != nil {
		fmt.Fprintf(a.stdout, "Parent:   %s  %s (%s)\n", parentInfo.ID, parentInfo.Title, parentInfo.Status)
	}
	fmt.Fprintf(a.stdout, "Created:  %s\n", td.Created)
	fmt.Fprintf(a.stdout, "Updated:  %s\n", td.Updated)
	if td.Closed != "" {
		fmt.Fprintf(a.stdout, "Closed:   %s\n", td.Closed)
	}

	if len(blockers) > 0 {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Blocked by:")
		for _, b := range blockers {
			fmt.Fprintf(a.stdout, "  %s  %s (%s)\n", b.ID, b.Title, b.Status)
		}
	}

	if len(children) > 0 {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Children:")
		for _, c := range children {
			fmt.Fprintf(a.stdout, "  %s  %s (%s)\n", c.ID, c.Title, c.Status)
		}
	}

	if td.Description != "" {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Description:")
		for _, line := range strings.Split(td.Description, "\n") {
			fmt.Fprintf(a.stdout, "  %s\n", line)
		}
	}

	return nil
}
