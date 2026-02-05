package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// runShow executes the show subcommand.
func (a *App) runShow(args []string) int {
	// Check for ID argument
	// args[0] is "tick", args[1] is "show", args[2] should be the ID
	if len(args) < 3 || args[2] == "" {
		fmt.Fprintf(a.Stderr, "Error: Task ID is required. Usage: tick show <id>\n")
		return 1
	}

	// Normalize input ID to lowercase
	taskID := task.NormalizeID(args[2])

	// Discover .tick directory
	tickDir, err := DiscoverTickDir(a.Cwd)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Open store
	store, err := storage.NewStore(tickDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}
	defer store.Close()

	// Task details
	type taskDetails struct {
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

	// Related task info (for blocked_by and children)
	type relatedTask struct {
		ID     string
		Title  string
		Status string
	}

	var t taskDetails
	var blockedBy []relatedTask
	var children []relatedTask
	var found bool

	// Query task details
	err = store.Query(func(db *sql.DB) error {
		// Get main task
		row := db.QueryRow(`SELECT id, title, status, priority, COALESCE(description, ''), COALESCE(parent, ''), created, updated, COALESCE(closed, '') FROM tasks WHERE id = ?`, taskID)
		err := row.Scan(&t.ID, &t.Title, &t.Status, &t.Priority, &t.Description, &t.Parent, &t.Created, &t.Updated, &t.Closed)
		if err == sql.ErrNoRows {
			return nil // Not found, but not an error
		}
		if err != nil {
			return err
		}
		found = true

		// Get parent title if parent exists
		if t.Parent != "" {
			var parentTitle string
			err := db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, t.Parent).Scan(&parentTitle)
			if err == nil {
				t.ParentTitle = parentTitle
			}
		}

		// Get blocked_by tasks (from dependencies table)
		depRows, err := db.Query(`
			SELECT t.id, t.title, t.status
			FROM dependencies d
			JOIN tasks t ON d.blocked_by = t.id
			WHERE d.task_id = ?`, taskID)
		if err != nil {
			return err
		}
		defer depRows.Close()

		for depRows.Next() {
			var r relatedTask
			if err := depRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
				return err
			}
			blockedBy = append(blockedBy, r)
		}
		if err := depRows.Err(); err != nil {
			return err
		}

		// Get children (tasks where parent = this task's ID)
		childRows, err := db.Query(`SELECT id, title, status FROM tasks WHERE parent = ?`, taskID)
		if err != nil {
			return err
		}
		defer childRows.Close()

		for childRows.Next() {
			var r relatedTask
			if err := childRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
				return err
			}
			children = append(children, r)
		}
		return childRows.Err()
	})

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Check if task was found
	if !found {
		fmt.Fprintf(a.Stderr, "Error: Task '%s' not found\n", taskID)
		return 1
	}

	// Output
	if a.flags.Quiet {
		// --quiet: output only task ID
		fmt.Fprintln(a.Stdout, t.ID)
		return 0
	}

	// Full output format as key-value
	fmt.Fprintf(a.Stdout, "ID:       %s\n", t.ID)
	fmt.Fprintf(a.Stdout, "Title:    %s\n", t.Title)
	fmt.Fprintf(a.Stdout, "Status:   %s\n", t.Status)
	fmt.Fprintf(a.Stdout, "Priority: %d\n", t.Priority)
	if t.Parent != "" {
		if t.ParentTitle != "" {
			fmt.Fprintf(a.Stdout, "Parent:   %s (%s)\n", t.Parent, t.ParentTitle)
		} else {
			fmt.Fprintf(a.Stdout, "Parent:   %s\n", t.Parent)
		}
	}
	fmt.Fprintf(a.Stdout, "Created:  %s\n", t.Created)
	fmt.Fprintf(a.Stdout, "Updated:  %s\n", t.Updated)
	if t.Closed != "" {
		fmt.Fprintf(a.Stdout, "Closed:   %s\n", t.Closed)
	}

	// Blocked by section (if any)
	if len(blockedBy) > 0 {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Blocked by:")
		for _, r := range blockedBy {
			fmt.Fprintf(a.Stdout, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	// Children section (if any)
	if len(children) > 0 {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Children:")
		for _, r := range children {
			fmt.Fprintf(a.Stdout, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	// Description section (if present)
	if t.Description != "" {
		fmt.Fprintln(a.Stdout)
		fmt.Fprintln(a.Stdout, "Description:")
		// Indent description lines
		for _, line := range strings.Split(t.Description, "\n") {
			fmt.Fprintf(a.Stdout, "  %s\n", line)
		}
	}

	return 0
}
