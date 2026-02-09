package cli

// blockedWhereClause contains the WHERE conditions for blocked tasks: status
// is open AND (has unclosed blocker OR has open/in_progress children). Shared
// by BlockedQuery and StatsBlockedCountQuery.
const blockedWhereClause = `t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON blocker.id = d.blocked_by
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )`

// BlockedQuery is the SQL query that returns open tasks failing ready
// conditions. This is the inverse of ReadyQuery -- blocked = open AND NOT
// ready. Results are ordered by priority ASC then created ASC for
// deterministic output.
const BlockedQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE ` + blockedWhereClause + `
ORDER BY t.priority ASC, t.created ASC
`

// runBlocked implements the "tick blocked" command, which is an alias for
// "tick list --blocked". It delegates to runList with --blocked prepended to args.
func runBlocked(ctx *Context) error {
	ctx.Args = append([]string{"--blocked"}, ctx.Args...)
	return runList(ctx)
}
