package cli

import (
	"database/sql"
	"fmt"

	"github.com/leeovery/tick/internal/store"
)

// blockedQuery returns open tasks that are NOT ready.
// Derived from readyConditionsFor: blocked = open AND NOT IN ready set.
// This reuses the ready query logic so changes to ready conditions
// automatically propagate to the blocked query.
// Order: priority ASC, created ASC (deterministic)
var blockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND t.id NOT IN (
    SELECT t.id FROM tasks t
    WHERE t.status = 'open'
      AND` + readyConditionsFor("t") + `
  )
ORDER BY t.priority ASC, t.created ASC
`

// runBlocked implements the `tick blocked` command.
// It is an alias for `tick list --blocked`, returning tasks that cannot be worked.
func (a *App) runBlocked(args []string) error {
	tickDir, err := DiscoverTickDir(a.Dir)
	if err != nil {
		return err
	}

	s, err := store.NewStore(tickDir)
	if err != nil {
		return err
	}
	defer s.Close()

	var rows []listRow

	err = s.Query(func(db *sql.DB) error {
		sqlRows, err := db.Query(blockedQuery)
		if err != nil {
			return fmt.Errorf("failed to query blocked tasks: %w", err)
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var r listRow
			if err := sqlRows.Scan(&r.ID, &r.Status, &r.Priority, &r.Title); err != nil {
				return fmt.Errorf("failed to scan task row: %w", err)
			}
			rows = append(rows, r)
		}
		return sqlRows.Err()
	})
	if err != nil {
		return err
	}

	return renderListOutput(rows, a.Stdout, a.Quiet)
}
