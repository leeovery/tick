package cli

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/leeovery/tick/internal/storage"
)

// listFlags holds parsed flags for the list command.
type listFlags struct {
	ready    bool
	blocked  bool
	status   string // empty = no filter
	priority *int   // nil = no filter
}

// parseListFlags extracts list-specific flags from args.
// Returns parsed flags and remaining args.
func parseListFlags(args []string) (listFlags, []string, error) {
	var flags listFlags
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--ready":
			flags.ready = true
		case "--blocked":
			flags.blocked = true
		case "--status":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("--status requires a value")
			}
			i++
			flags.status = args[i]
		case "--priority":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("--priority requires a value")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return flags, nil, fmt.Errorf("invalid priority value '%s': must be a number 0-4", args[i])
			}
			flags.priority = &p
		default:
			remaining = append(remaining, arg)
		}
	}

	return flags, remaining, nil
}

// validateListFlags validates the parsed list flags.
func validateListFlags(flags listFlags) error {
	// --ready and --blocked are mutually exclusive
	if flags.ready && flags.blocked {
		return fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	// Validate status value
	if flags.status != "" {
		validStatuses := map[string]bool{
			"open":        true,
			"in_progress": true,
			"done":        true,
			"cancelled":   true,
		}
		if !validStatuses[flags.status] {
			return fmt.Errorf("invalid status '%s': must be one of open, in_progress, done, cancelled", flags.status)
		}
	}

	// Validate priority range
	if flags.priority != nil {
		if *flags.priority < 0 || *flags.priority > 4 {
			return fmt.Errorf("invalid priority %d: must be between 0 and 4", *flags.priority)
		}
	}

	return nil
}

// runList executes the list subcommand.
func (a *App) runList(args []string) int {
	// Parse list-specific flags
	flags, _, err := parseListFlags(args)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Validate flags
	if err := validateListFlags(flags); err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

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

	var tasks []taskRow

	// Query tasks from SQLite based on filters
	a.WriteVerbose("lock acquire shared")
	a.WriteVerbose("cache freshness check")
	err = store.Query(func(db *sql.DB) error {
		var queryErr error
		tasks, queryErr = queryListTasks(db, flags)
		return queryErr
	})
	a.WriteVerbose("lock release")

	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: %s\n", err)
		return 1
	}

	// Output
	if a.formatConfig.Quiet {
		// --quiet: output only task IDs, one per line
		for _, t := range tasks {
			fmt.Fprintln(a.Stdout, t.ID)
		}
	} else {
		// Build task list data for formatter
		data := &TaskListData{
			Tasks: make([]TaskRowData, len(tasks)),
		}
		for i, t := range tasks {
			data.Tasks[i] = TaskRowData{
				ID:       t.ID,
				Title:    t.Title,
				Status:   t.Status,
				Priority: t.Priority,
			}
		}
		formatter := a.formatConfig.Formatter()
		fmt.Fprint(a.Stdout, formatter.FormatTaskList(data))
	}

	return 0
}

// queryListTasks builds and executes query with filters applied.
// Returns slice of taskRow ordered by priority ASC, created ASC.
func queryListTasks(db *sql.DB, flags listFlags) ([]taskRow, error) {
	// Use existing query functions for --ready and --blocked
	if flags.ready {
		return queryReadyTasksWithFilters(db, flags)
	}
	if flags.blocked {
		return queryBlockedTasksWithFilters(db, flags)
	}

	// Build WHERE clause for status and priority filters
	query := `SELECT t.id, t.title, t.status, t.priority FROM tasks t`
	var conditions []string
	var args []interface{}

	if flags.status != "" {
		conditions = append(conditions, "t.status = ?")
		args = append(args, flags.status)
	}

	if flags.priority != nil {
		conditions = append(conditions, "t.priority = ?")
		args = append(args, *flags.priority)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " ORDER BY t.priority ASC, t.created ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskRow
	for rows.Next() {
		var t taskRow
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// queryReadyTasksWithFilters queries ready tasks with additional filters.
func queryReadyTasksWithFilters(db *sql.DB, flags listFlags) ([]taskRow, error) {
	// Start with ReadyCondition and add additional filters
	query := `
		SELECT t.id, t.title, t.status, t.priority
		FROM tasks t
		WHERE ` + ReadyCondition

	var args []interface{}

	// Status filter - ready already implies open, but if user specifies something else,
	// it becomes contradictory and will return empty (which is correct behavior)
	if flags.status != "" {
		query += " AND t.status = ?"
		args = append(args, flags.status)
	}

	if flags.priority != nil {
		query += " AND t.priority = ?"
		args = append(args, *flags.priority)
	}

	query += " ORDER BY t.priority ASC, t.created ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskRow
	for rows.Next() {
		var t taskRow
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// queryBlockedTasksWithFilters queries blocked tasks with additional filters.
func queryBlockedTasksWithFilters(db *sql.DB, flags listFlags) ([]taskRow, error) {
	// Start with BlockedCondition and add additional filters
	query := `
		SELECT t.id, t.title, t.status, t.priority
		FROM tasks t
		WHERE ` + BlockedCondition

	var args []interface{}

	if flags.status != "" {
		query += " AND t.status = ?"
		args = append(args, flags.status)
	}

	if flags.priority != nil {
		query += " AND t.priority = ?"
		args = append(args, *flags.priority)
	}

	query += " ORDER BY t.priority ASC, t.created ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskRow
	for rows.Next() {
		var t taskRow
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}
