package cli

import (
	"database/sql"
	"fmt"

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
	a.WriteVerbose("store open %s", tickDir)
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
	a.WriteVerbose("lock acquire shared")
	a.WriteVerbose("cache freshness check")
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
	a.WriteVerbose("lock release")

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
	if a.formatConfig.Quiet {
		// --quiet: output only task ID
		fmt.Fprintln(a.Stdout, t.ID)
		return 0
	}

	// Build task detail data for formatter
	data := &TaskDetailData{
		ID:          t.ID,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		Description: t.Description,
		Parent:      t.Parent,
		ParentTitle: t.ParentTitle,
		Created:     t.Created,
		Updated:     t.Updated,
		Closed:      t.Closed,
		BlockedBy:   make([]RelatedTaskData, len(blockedBy)),
		Children:    make([]RelatedTaskData, len(children)),
	}
	for i, r := range blockedBy {
		data.BlockedBy[i] = RelatedTaskData{ID: r.ID, Title: r.Title, Status: r.Status}
	}
	for i, r := range children {
		data.Children[i] = RelatedTaskData{ID: r.ID, Title: r.Title, Status: r.Status}
	}

	formatter := a.formatConfig.Formatter()
	fmt.Fprint(a.Stdout, formatter.FormatTaskDetail(data))

	return 0
}
