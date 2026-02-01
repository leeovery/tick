package cli

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/storage"
	"github.com/leeovery/tick/internal/task"
)

// buildTaskDetail queries full task detail from the database for formatted output.
// Used by show, create, and update commands.
func (a *App) buildAndFormatTaskDetail(workDir, taskID string) error {
	tickDir, err := FindTickDir(workDir)
	if err != nil {
		return err
	}

	store, err := storage.NewStore(tickDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	return a.queryAndFormatTaskDetail(store, taskID)
}

// queryAndFormatTaskDetail queries and formats task detail using an open store.
func (a *App) queryAndFormatTaskDetail(store *storage.Store, taskID string) error {
	type depInfo struct {
		ID     string
		Title  string
		Status string
	}

	type taskDetailRow struct {
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

	var td taskDetailRow
	var found bool
	var blockers []depInfo
	var children []depInfo
	var parentInfo *depInfo

	err := store.Query(func(db *sql.DB) error {
		var description, parent, closed sql.NullString
		var created, updated string
		err := db.QueryRow(
			"SELECT id, title, status, priority, description, parent, created, updated, closed FROM tasks WHERE id=?",
			taskID,
		).Scan(&td.ID, &td.Title, &td.Status, &td.Priority, &description, &parent, &created, &updated, &closed)

		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("querying task: %w", err)
		}
		found = true
		td.Created = created
		td.Updated = updated
		if description.Valid {
			td.Description = description.String
		}
		if parent.Valid {
			td.Parent = parent.String
		}
		if closed.Valid {
			td.Closed = closed.String
		}

		rows, err := db.Query(
			"SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id=?",
			taskID,
		)
		if err != nil {
			return fmt.Errorf("querying dependencies: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var d depInfo
			if err := rows.Scan(&d.ID, &d.Title, &d.Status); err != nil {
				return fmt.Errorf("scanning dependency: %w", err)
			}
			blockers = append(blockers, d)
		}

		childRows, err := db.Query("SELECT id, title, status FROM tasks WHERE parent=?", taskID)
		if err != nil {
			return fmt.Errorf("querying children: %w", err)
		}
		defer childRows.Close()
		for childRows.Next() {
			var c depInfo
			if err := childRows.Scan(&c.ID, &c.Title, &c.Status); err != nil {
				return fmt.Errorf("scanning child: %w", err)
			}
			children = append(children, c)
		}

		if td.Parent != "" {
			var p depInfo
			err := db.QueryRow("SELECT id, title, status FROM tasks WHERE id=?", td.Parent).
				Scan(&p.ID, &p.Title, &p.Status)
			if err == nil {
				parentInfo = &p
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("Task '%s' not found", taskID)
	}

	if a.opts.Quiet {
		fmt.Fprintln(a.stdout, td.ID)
		return nil
	}

	detail := TaskDetail{
		ID:          td.ID,
		Title:       td.Title,
		Status:      td.Status,
		Priority:    td.Priority,
		Description: td.Description,
		Created:     td.Created,
		Updated:     td.Updated,
		Closed:      td.Closed,
	}

	if parentInfo != nil {
		detail.Parent = &RelatedTask{ID: parentInfo.ID, Title: parentInfo.Title, Status: parentInfo.Status}
	}

	detail.BlockedBy = make([]RelatedTask, len(blockers))
	for i, b := range blockers {
		detail.BlockedBy[i] = RelatedTask{ID: b.ID, Title: b.Title, Status: b.Status}
	}

	detail.Children = make([]RelatedTask, len(children))
	for i, c := range children {
		detail.Children[i] = RelatedTask{ID: c.ID, Title: c.Title, Status: c.Status}
	}

	return a.fmtr.FormatTaskDetail(a.stdout, detail)
}

var validStatuses = map[string]bool{
	"open": true, "in_progress": true, "done": true, "cancelled": true,
}

func (a *App) cmdList(workDir string, args []string) error {
	// Parse list filter flags.
	var (
		filterReady    bool
		filterBlocked  bool
		filterStatus   string
		filterPriority = -1 // sentinel: no filter
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ready":
			filterReady = true
		case "--blocked":
			filterBlocked = true
		case "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			filterStatus = args[i]
			if !validStatuses[filterStatus] {
				return fmt.Errorf("invalid status '%s'. Valid: open, in_progress, done, cancelled", filterStatus)
			}
		case "--priority":
			if i+1 >= len(args) {
				return fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid priority value: %s", args[i])
			}
			if err := task.ValidatePriority(p); err != nil {
				return err
			}
			filterPriority = p
		}
	}

	if filterReady && filterBlocked {
		return fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	// Build query based on filters.
	var query string
	switch {
	case filterReady:
		query = readyQuery
	case filterBlocked:
		query = blockedQuery
	default:
		query = "SELECT t.id, t.title, t.status, t.priority FROM tasks t"
	}

	// Apply additional filters.
	query = a.applyListFilters(query, filterStatus, filterPriority, filterReady || filterBlocked)

	return a.cmdListFiltered(workDir, query)
}

// applyListFilters adds WHERE/AND clauses for --status and --priority to a query.
func (a *App) applyListFilters(baseQuery string, status string, priority int, hasBaseFilter bool) string {
	var conditions []string

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("t.status = '%s'", status))
	}
	if priority >= 0 {
		conditions = append(conditions, fmt.Sprintf("t.priority = %d", priority))
	}

	if len(conditions) == 0 {
		if !hasBaseFilter {
			return baseQuery + " ORDER BY t.priority ASC, t.created ASC"
		}
		return baseQuery
	}

	extra := strings.Join(conditions, " AND ")

	if hasBaseFilter {
		// Ready/blocked queries already have WHERE. Insert additional conditions before ORDER BY.
		// The queries end with ORDER BY clause. Insert AND before it.
		orderIdx := strings.LastIndex(baseQuery, "ORDER BY")
		if orderIdx > 0 {
			return baseQuery[:orderIdx] + "AND " + extra + "\n" + baseQuery[orderIdx:]
		}
		return baseQuery + " AND " + extra
	}

	return baseQuery + " WHERE " + extra + " ORDER BY t.priority ASC, t.created ASC"
}

func (a *App) cmdShow(workDir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Task ID is required. Usage: tick show <id>")
	}

	taskID := task.NormalizeID(args[0])
	return a.buildAndFormatTaskDetail(workDir, taskID)
}
