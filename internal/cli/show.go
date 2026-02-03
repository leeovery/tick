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

	data, err := queryShowData(store, lookupID)
	if err != nil {
		return err
	}

	// Output
	if a.config.Quiet {
		fmt.Fprintln(a.stdout, data.ID)
		return nil
	}

	return a.formatter.FormatTaskDetail(a.stdout, data)
}

// queryShowData queries full task details from the store for display.
// Reused by show, create, and update commands.
func queryShowData(store *storage.Store, lookupID string) (*showData, error) {
	var data *showData

	err := store.Query(func(db *sql.DB) error {
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

	return data, err
}
