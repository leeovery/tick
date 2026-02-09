package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/leeovery/tick/internal/engine"
	"github.com/leeovery/tick/internal/task"
)

// relatedTask holds minimal info about a related task (blocker, child, or parent).
type relatedTask struct {
	id     string
	title  string
	status string
}

// showData holds all data needed to display a single task's details.
type showData struct {
	id          string
	title       string
	status      string
	priority    int
	created     string
	updated     string
	closed      string
	description string
	parent      string
	parentTitle string
	blockedBy   []relatedTask
	children    []relatedTask
}

// runShow implements the "tick show" command. It displays full details of a
// task including its dependencies, children, and description.
func runShow(ctx *Context) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick show <id>")
	}

	id := task.NormalizeID(ctx.Args[0])

	tickDir, err := DiscoverTickDir(ctx.WorkDir)
	if err != nil {
		return err
	}

	store, err := engine.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var data showData
	var found bool

	err = store.Query(func(db *sql.DB) error {
		// Query main task.
		var description, parent, closed sql.NullString
		err := db.QueryRow(
			`SELECT id, title, status, priority, created, updated, closed, description, parent
			 FROM tasks WHERE id = ?`, id,
		).Scan(&data.id, &data.title, &data.status, &data.priority,
			&data.created, &data.updated, &closed, &description, &parent)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("querying task: %w", err)
		}
		found = true

		if description.Valid {
			data.description = description.String
		}
		if closed.Valid {
			data.closed = closed.String
		}
		if parent.Valid {
			data.parent = parent.String
			// Look up parent title.
			var parentTitle string
			err := db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, data.parent).Scan(&parentTitle)
			if err == nil {
				data.parentTitle = parentTitle
			}
		}

		// Query blocked_by (dependencies with context).
		depRows, err := db.Query(
			`SELECT t.id, t.title, t.status
			 FROM dependencies d
			 JOIN tasks t ON t.id = d.blocked_by
			 WHERE d.task_id = ?`, id,
		)
		if err != nil {
			return fmt.Errorf("querying dependencies: %w", err)
		}
		defer depRows.Close()

		for depRows.Next() {
			var rt relatedTask
			if err := depRows.Scan(&rt.id, &rt.title, &rt.status); err != nil {
				return fmt.Errorf("scanning dependency: %w", err)
			}
			data.blockedBy = append(data.blockedBy, rt)
		}
		if err := depRows.Err(); err != nil {
			return fmt.Errorf("iterating dependencies: %w", err)
		}

		// Query children.
		childRows, err := db.Query(
			`SELECT id, title, status FROM tasks WHERE parent = ?`, id,
		)
		if err != nil {
			return fmt.Errorf("querying children: %w", err)
		}
		defer childRows.Close()

		for childRows.Next() {
			var rt relatedTask
			if err := childRows.Scan(&rt.id, &rt.title, &rt.status); err != nil {
				return fmt.Errorf("scanning child: %w", err)
			}
			data.children = append(data.children, rt)
		}
		return childRows.Err()
	})
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("Task '%s' not found", id)
	}

	if ctx.Quiet {
		fmt.Fprintln(ctx.Stdout, data.id)
		return nil
	}

	printShowDetails(ctx.Stdout, data)
	return nil
}

// printShowDetails outputs the full task detail view in key-value format.
func printShowDetails(w io.Writer, d showData) {
	fmt.Fprintf(w, "ID:       %s\n", d.id)
	fmt.Fprintf(w, "Title:    %s\n", d.title)
	fmt.Fprintf(w, "Status:   %s\n", d.status)
	fmt.Fprintf(w, "Priority: %d\n", d.priority)
	fmt.Fprintf(w, "Created:  %s\n", d.created)
	fmt.Fprintf(w, "Updated:  %s\n", d.updated)

	if d.closed != "" {
		fmt.Fprintf(w, "Closed:   %s\n", d.closed)
	}

	if d.parent != "" {
		if d.parentTitle != "" {
			fmt.Fprintf(w, "Parent:   %s  %s\n", d.parent, d.parentTitle)
		} else {
			fmt.Fprintf(w, "Parent:   %s\n", d.parent)
		}
	}

	if len(d.blockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, rt := range d.blockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", rt.id, rt.title, rt.status)
		}
	}

	if len(d.children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, rt := range d.children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", rt.id, rt.title, rt.status)
		}
	}

	if strings.TrimSpace(d.description) != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		fmt.Fprintf(w, "  %s\n", d.description)
	}
}
