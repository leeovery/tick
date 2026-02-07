package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/store"
	"github.com/leeovery/tick/internal/task"
)

// runShow implements the `tick show <id>` command.
// It displays full details of a single task including dependencies and children.
func (a *App) runShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick show <id>")
	}

	id := task.NormalizeID(strings.TrimSpace(args[0]))

	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

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

	type relatedTask struct {
		ID     string
		Title  string
		Status string
	}

	var detail taskDetail
	var blockedBy []relatedTask
	var children []relatedTask
	var parentInfo *relatedTask

	err = s.Query(func(db *sql.DB) error {
		// Query the task itself
		var desc, parent, closed sql.NullString
		err := db.QueryRow(
			"SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?",
			id,
		).Scan(&detail.ID, &detail.Title, &detail.Status, &detail.Priority, &desc, &parent, &detail.Created, &detail.Updated, &closed)
		if err == sql.ErrNoRows {
			return fmt.Errorf("Task '%s' not found", id)
		}
		if err != nil {
			return fmt.Errorf("failed to query task: %w", err)
		}

		if desc.Valid {
			detail.Description = desc.String
		}
		if parent.Valid {
			detail.Parent = parent.String
		}
		if closed.Valid {
			detail.Closed = closed.String
		}

		// Query blocked_by tasks with context
		depRows, err := db.Query(
			"SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id = ?",
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query dependencies: %w", err)
		}
		defer depRows.Close()

		for depRows.Next() {
			var r relatedTask
			if err := depRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
				return fmt.Errorf("failed to scan dependency row: %w", err)
			}
			blockedBy = append(blockedBy, r)
		}
		if err := depRows.Err(); err != nil {
			return fmt.Errorf("dependency rows error: %w", err)
		}

		// Query children with context
		childRows, err := db.Query(
			"SELECT id, title, status FROM tasks WHERE parent = ?",
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query children: %w", err)
		}
		defer childRows.Close()

		for childRows.Next() {
			var r relatedTask
			if err := childRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
				return fmt.Errorf("failed to scan child row: %w", err)
			}
			children = append(children, r)
		}
		if err := childRows.Err(); err != nil {
			return fmt.Errorf("children rows error: %w", err)
		}

		// Query parent info if parent is set
		if detail.Parent != "" {
			var p relatedTask
			err := db.QueryRow(
				"SELECT id, title, status FROM tasks WHERE id = ?",
				detail.Parent,
			).Scan(&p.ID, &p.Title, &p.Status)
			if err == nil {
				parentInfo = &p
			}
			// If parent not found (orphaned), we just skip showing parent info
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Quiet mode: only ID
	if a.Quiet {
		fmt.Fprintln(a.Stdout, detail.ID)
		return nil
	}

	// Output key-value format
	fmt.Fprintf(a.Stdout, "ID:       %s\n", detail.ID)
	fmt.Fprintf(a.Stdout, "Title:    %s\n", detail.Title)
	fmt.Fprintf(a.Stdout, "Status:   %s\n", detail.Status)
	fmt.Fprintf(a.Stdout, "Priority: %d\n", detail.Priority)
	if parentInfo != nil {
		fmt.Fprintf(a.Stdout, "Parent:   %s  %s\n", parentInfo.ID, parentInfo.Title)
	}
	fmt.Fprintf(a.Stdout, "Created:  %s\n", detail.Created)
	fmt.Fprintf(a.Stdout, "Updated:  %s\n", detail.Updated)
	if detail.Closed != "" {
		fmt.Fprintf(a.Stdout, "Closed:   %s\n", detail.Closed)
	}

	// Blocked by section
	if len(blockedBy) > 0 {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Blocked by:")
		for _, b := range blockedBy {
			fmt.Fprintf(a.Stdout, "  %s  %s (%s)\n", b.ID, b.Title, b.Status)
		}
	}

	// Children section
	if len(children) > 0 {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Children:")
		for _, c := range children {
			fmt.Fprintf(a.Stdout, "  %s  %s (%s)\n", c.ID, c.Title, c.Status)
		}
	}

	// Description section
	if detail.Description != "" {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Description:")
		// Indent each line of description
		for _, line := range strings.Split(detail.Description, "\n") {
			fmt.Fprintf(a.Stdout, "  %s\n", line)
		}
	}

	return nil
}
