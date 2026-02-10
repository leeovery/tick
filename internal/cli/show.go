package cli

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// showData holds all data needed to render the show command output.
type showData struct {
	id          string
	title       string
	status      string
	priority    int
	description string
	parentID    string
	parentTitle string
	created     string
	updated     string
	closed      string
	blockedBy   []relatedTask
	children    []relatedTask
}

// relatedTask represents a task referenced in blocked_by or children sections.
type relatedTask struct {
	id     string
	title  string
	status string
}

// RunShow executes the show command: queries a single task by ID from SQLite and
// outputs its full details including blocked_by, children, and description sections.
func RunShow(dir string, quiet bool, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick show <id>")
	}

	id := task.NormalizeID(args[0])

	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer store.Close()

	data, err := queryShowData(store, id)
	if err != nil {
		return err
	}

	if quiet {
		fmt.Fprintln(stdout, data.id)
		return nil
	}

	printShowOutput(stdout, data)
	return nil
}

// queryShowData queries a task by ID from SQLite and returns its full details
// including blocked_by, children, and parent context.
func queryShowData(store *storage.Store, id string) (showData, error) {
	var data showData

	err := store.Query(func(db *sql.DB) error {
		var descPtr, parentPtr, closedPtr *string
		err := db.QueryRow(
			`SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id = ?`,
			id,
		).Scan(&data.id, &data.title, &data.status, &data.priority, &descPtr, &parentPtr, &data.created, &data.updated, &closedPtr)
		if err == sql.ErrNoRows {
			return fmt.Errorf("task '%s' not found", id)
		}
		if err != nil {
			return fmt.Errorf("failed to query task: %w", err)
		}

		if descPtr != nil {
			data.description = *descPtr
		}
		if parentPtr != nil {
			data.parentID = *parentPtr
		}
		if closedPtr != nil {
			data.closed = *closedPtr
		}

		// Query parent title if parent is set.
		if data.parentID != "" {
			var parentTitle string
			err := db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, data.parentID).Scan(&parentTitle)
			if err == nil {
				data.parentTitle = parentTitle
			}
			// If parent not found, we still show the parent ID without title
		}

		// Query blocked_by dependencies with context.
		depRows, err := db.Query(
			`SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id = ? ORDER BY t.id`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query dependencies: %w", err)
		}
		defer depRows.Close()

		for depRows.Next() {
			var r relatedTask
			if err := depRows.Scan(&r.id, &r.title, &r.status); err != nil {
				return fmt.Errorf("failed to scan dependency row: %w", err)
			}
			data.blockedBy = append(data.blockedBy, r)
		}
		if err := depRows.Err(); err != nil {
			return err
		}

		// Query children with context.
		childRows, err := db.Query(
			`SELECT id, title, status FROM tasks WHERE parent = ? ORDER BY id`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query children: %w", err)
		}
		defer childRows.Close()

		for childRows.Next() {
			var r relatedTask
			if err := childRows.Scan(&r.id, &r.title, &r.status); err != nil {
				return fmt.Errorf("failed to scan child row: %w", err)
			}
			data.children = append(data.children, r)
		}
		return childRows.Err()
	})

	return data, err
}

// printShowOutput renders the show command output in key-value format.
func printShowOutput(w io.Writer, d showData) {
	fmt.Fprintf(w, "ID:       %s\n", d.id)
	fmt.Fprintf(w, "Title:    %s\n", d.title)
	fmt.Fprintf(w, "Status:   %s\n", d.status)
	fmt.Fprintf(w, "Priority: %d\n", d.priority)

	if d.parentID != "" {
		if d.parentTitle != "" {
			fmt.Fprintf(w, "Parent:   %s (%s)\n", d.parentID, d.parentTitle)
		} else {
			fmt.Fprintf(w, "Parent:   %s\n", d.parentID)
		}
	}

	fmt.Fprintf(w, "Created:  %s\n", d.created)
	fmt.Fprintf(w, "Updated:  %s\n", d.updated)

	if d.closed != "" {
		fmt.Fprintf(w, "Closed:   %s\n", d.closed)
	}

	if len(d.blockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Blocked by:")
		for _, r := range d.blockedBy {
			fmt.Fprintf(w, "  %s  %s (%s)\n", r.id, r.title, r.status)
		}
	}

	if len(d.children) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Children:")
		for _, r := range d.children {
			fmt.Fprintf(w, "  %s  %s (%s)\n", r.id, r.title, r.status)
		}
	}

	if d.description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Description:")
		fmt.Fprintf(w, "  %s\n", d.description)
	}
}
