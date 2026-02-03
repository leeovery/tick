package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// showData holds all data needed to render the show output.
type showData struct {
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
	BlockedBy   []relatedTask
	Children    []relatedTask
}

// relatedTask holds the ID, title, and status of a related task (blocker or child).
type relatedTask struct {
	ID     string
	Title  string
	Status string
}

// runShow implements the `tick show <id>` command.
// It displays full details for a single task including dependencies, children, and description.
func (a *App) runShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick show <id>")
	}

	lookupID := task.NormalizeID(args[0])

	tickDir, err := DiscoverTickDir(a.workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	var data *showData

	err = store.Query(func(db *sql.DB) error {
		// Query the task itself
		var d showData
		var description, parent, closed sql.NullString
		err := db.QueryRow(
			"SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?",
			lookupID,
		).Scan(&d.ID, &d.Title, &d.Status, &d.Priority, &description, &parent, &d.Created, &d.Updated, &closed)
		if err == sql.ErrNoRows {
			return fmt.Errorf("Task '%s' not found", lookupID)
		}
		if err != nil {
			return fmt.Errorf("failed to query task: %w", err)
		}

		if description.Valid {
			d.Description = description.String
		}
		if parent.Valid {
			d.Parent = parent.String
			// Look up parent title
			var parentTitle string
			err := db.QueryRow("SELECT title FROM tasks WHERE id = ?", d.Parent).Scan(&parentTitle)
			if err == nil {
				d.ParentTitle = parentTitle
			}
		}
		if closed.Valid {
			d.Closed = closed.String
		}

		// Query blocked_by with context (ID, title, status)
		depRows, err := db.Query(
			"SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id = ?",
			lookupID,
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
			d.BlockedBy = append(d.BlockedBy, r)
		}
		if err := depRows.Err(); err != nil {
			return fmt.Errorf("dependency rows error: %w", err)
		}

		// Query children
		childRows, err := db.Query(
			"SELECT id, title, status FROM tasks WHERE parent = ?",
			lookupID,
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
			d.Children = append(d.Children, r)
		}
		if err := childRows.Err(); err != nil {
			return fmt.Errorf("children rows error: %w", err)
		}

		data = &d
		return nil
	})
	if err != nil {
		return err
	}

	// Output
	if a.config.Quiet {
		fmt.Fprintln(a.stdout, data.ID)
		return nil
	}

	a.printShowOutput(data)
	return nil
}

// printShowOutput renders the full show output for a task.
func (a *App) printShowOutput(d *showData) {
	fmt.Fprintf(a.stdout, "ID:       %s\n", d.ID)
	fmt.Fprintf(a.stdout, "Title:    %s\n", d.Title)
	fmt.Fprintf(a.stdout, "Status:   %s\n", d.Status)
	fmt.Fprintf(a.stdout, "Priority: %d\n", d.Priority)

	if d.Parent != "" {
		if d.ParentTitle != "" {
			fmt.Fprintf(a.stdout, "Parent:   %s  %s\n", d.Parent, d.ParentTitle)
		} else {
			fmt.Fprintf(a.stdout, "Parent:   %s\n", d.Parent)
		}
	}

	fmt.Fprintf(a.stdout, "Created:  %s\n", d.Created)
	fmt.Fprintf(a.stdout, "Updated:  %s\n", d.Updated)

	if d.Closed != "" {
		fmt.Fprintf(a.stdout, "Closed:   %s\n", d.Closed)
	}

	if len(d.BlockedBy) > 0 {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Blocked by:")
		for _, r := range d.BlockedBy {
			fmt.Fprintf(a.stdout, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	if len(d.Children) > 0 {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Children:")
		for _, r := range d.Children {
			fmt.Fprintf(a.stdout, "  %s  %s (%s)\n", r.ID, r.Title, r.Status)
		}
	}

	if d.Description != "" {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Description:")
		fmt.Fprintf(a.stdout, "  %s\n", d.Description)
	}
}
