package cli

// ReadyQuery is the SQL query that returns tasks matching all three ready
// conditions: status is open, all blockers are closed (done/cancelled), and
// no children have status open or in_progress. Results are ordered by
// priority ASC then created ASC for deterministic output.
const ReadyQuery = `
SELECT t.id, t.status, t.priority, t.title
FROM tasks t
WHERE t.status = 'open'
  AND NOT EXISTS (
    SELECT 1 FROM dependencies d
    JOIN tasks blocker ON blocker.id = d.blocked_by
    WHERE d.task_id = t.id
      AND blocker.status NOT IN ('done', 'cancelled')
  )
  AND NOT EXISTS (
    SELECT 1 FROM tasks child
    WHERE child.parent = t.id
      AND child.status IN ('open', 'in_progress')
  )
ORDER BY t.priority ASC, t.created ASC
`

// runReady implements the "tick ready" command, which is an alias for
// "tick list --ready". It delegates to runList with --ready prepended to args.
func runReady(ctx *Context) error {
	ctx.Args = append([]string{"--ready"}, ctx.Args...)
	return runList(ctx)
}
