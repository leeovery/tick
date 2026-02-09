package cli

import (
	"database/sql"
	"fmt"

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

	store, err := engine.NewStore(tickDir, ctx.storeOpts()...)
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

	return ctx.Fmt.FormatTaskDetail(ctx.Stdout, &data)
}

// taskToShowData converts a task.Task to showData for formatter output.
// It populates basic fields but does not enrich blockedBy or children with
// context (titles/statuses) since those require DB queries. Use the show
// command's full query flow when enriched data is needed.
func taskToShowData(t task.Task) *showData {
	d := &showData{
		id:          t.ID,
		title:       t.Title,
		status:      string(t.Status),
		priority:    t.Priority,
		created:     task.FormatTimestamp(t.Created),
		updated:     task.FormatTimestamp(t.Updated),
		description: t.Description,
		parent:      t.Parent,
	}
	if t.Closed != nil {
		d.closed = task.FormatTimestamp(*t.Closed)
	}
	for _, dep := range t.BlockedBy {
		d.blockedBy = append(d.blockedBy, relatedTask{id: dep})
	}
	return d
}
