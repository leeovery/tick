package cli

import (
	"database/sql"
	"fmt"
	"strings"

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

	s, err := a.openStore(tickDir)
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
		ParentTitle string
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
			// Query parent task's title
			var parentTitle string
			err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", parent.String).Scan(&parentTitle)
			if err == nil {
				detail.ParentTitle = parentTitle
			}
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

	// Build TaskDetail for formatter
	td := TaskDetail{
		ID:          detail.ID,
		Title:       detail.Title,
		Status:      detail.Status,
		Priority:    detail.Priority,
		Description: detail.Description,
		Parent:      detail.Parent,
		ParentTitle: detail.ParentTitle,
		Created:     detail.Created,
		Updated:     detail.Updated,
		Closed:      detail.Closed,
	}

	for _, b := range blockedBy {
		td.BlockedBy = append(td.BlockedBy, RelatedTask{
			ID:     b.ID,
			Title:  b.Title,
			Status: b.Status,
		})
	}

	for _, c := range children {
		td.Children = append(td.Children, RelatedTask{
			ID:     c.ID,
			Title:  c.Title,
			Status: c.Status,
		})
	}

	return a.Formatter.FormatTaskDetail(a.Stdout, td)
}
