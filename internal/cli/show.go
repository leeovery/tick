package cli

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// showData holds all data needed to render the show command output.
type showData struct {
	id          string
	title       string
	status      string
	priority    int
	taskType    string
	description string
	parentID    string
	parentTitle string
	created     string
	updated     string
	closed      string
	blockedBy   []RelatedTask
	children    []RelatedTask
	tags        []string
	refs        []string
	notes       []task.Note
}

// RunShow executes the show command: queries a single task by ID from SQLite and
// outputs its full details via the Formatter, including blocked_by, children, and description sections.
func RunShow(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required. Usage: tick show <id>")
	}

	store, err := openStore(dir, fc)
	if err != nil {
		return err
	}
	defer store.Close()

	id, err := store.ResolveID(args[0])
	if err != nil {
		return err
	}

	data, err := queryShowData(store, id)
	if err != nil {
		return err
	}

	if fc.Quiet {
		fmt.Fprintln(stdout, data.id)
		return nil
	}

	detail := showDataToTaskDetail(data)
	fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
	return nil
}

// queryShowData queries a task by ID from SQLite and returns its full details
// including blocked_by, children, and parent context.
func queryShowData(store *storage.Store, id string) (showData, error) {
	var data showData

	err := store.Query(func(db *sql.DB) error {
		var descPtr, parentPtr, closedPtr, typePtr *string
		err := db.QueryRow(
			`SELECT id, title, status, priority, type, description, parent, created, updated, closed FROM tasks WHERE id = ?`,
			id,
		).Scan(&data.id, &data.title, &data.status, &data.priority, &typePtr, &descPtr, &parentPtr, &data.created, &data.updated, &closedPtr)
		if err == sql.ErrNoRows {
			return fmt.Errorf("task '%s' not found", id)
		}
		if err != nil {
			return fmt.Errorf("failed to query task: %w", err)
		}

		if typePtr != nil {
			data.taskType = *typePtr
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
			var r RelatedTask
			if err := depRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
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
			var r RelatedTask
			if err := childRows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
				return fmt.Errorf("failed to scan child row: %w", err)
			}
			data.children = append(data.children, r)
		}
		if err := childRows.Err(); err != nil {
			return err
		}

		// Query tags.
		tagRows, err := db.Query(
			`SELECT tag FROM task_tags WHERE task_id = ? ORDER BY tag`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query tags: %w", err)
		}
		defer tagRows.Close()

		for tagRows.Next() {
			var tag string
			if err := tagRows.Scan(&tag); err != nil {
				return fmt.Errorf("failed to scan tag row: %w", err)
			}
			data.tags = append(data.tags, tag)
		}
		if err := tagRows.Err(); err != nil {
			return err
		}

		// Query refs.
		refRows, err := db.Query(
			`SELECT ref FROM task_refs WHERE task_id = ? ORDER BY ref`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query refs: %w", err)
		}
		defer refRows.Close()

		for refRows.Next() {
			var ref string
			if err := refRows.Scan(&ref); err != nil {
				return fmt.Errorf("failed to scan ref row: %w", err)
			}
			data.refs = append(data.refs, ref)
		}
		if err := refRows.Err(); err != nil {
			return err
		}

		// Query notes.
		noteRows, err := db.Query(
			`SELECT text, created FROM task_notes WHERE task_id = ? ORDER BY created ASC`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to query notes: %w", err)
		}
		defer noteRows.Close()

		for noteRows.Next() {
			var text, createdStr string
			if err := noteRows.Scan(&text, &createdStr); err != nil {
				return fmt.Errorf("failed to scan note row: %w", err)
			}
			created, err := time.Parse(task.TimestampFormat, createdStr)
			if err != nil {
				return fmt.Errorf("failed to parse note timestamp: %w", err)
			}
			data.notes = append(data.notes, task.Note{Text: text, Created: created})
		}
		return noteRows.Err()
	})

	return data, err
}

// showDataToTaskDetail converts a showData struct (from SQL query) to a TaskDetail
// struct suitable for the Formatter interface.
func showDataToTaskDetail(d showData) TaskDetail {
	created, _ := time.Parse(task.TimestampFormat, d.created)
	updated, _ := time.Parse(task.TimestampFormat, d.updated)

	t := task.Task{
		ID:          d.id,
		Title:       d.title,
		Status:      task.Status(d.status),
		Priority:    d.priority,
		Type:        d.taskType,
		Description: d.description,
		Parent:      d.parentID,
		Created:     created,
		Updated:     updated,
	}

	if d.closed != "" {
		closedTime, _ := time.Parse(task.TimestampFormat, d.closed)
		t.Closed = &closedTime
	}

	return TaskDetail{
		Task:        t,
		BlockedBy:   d.blockedBy,
		Children:    d.children,
		ParentTitle: d.parentTitle,
		Tags:        d.tags,
		Refs:        d.refs,
		Notes:       d.notes,
	}
}
