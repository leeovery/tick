package cli

// blockedQuery is the SQL for open tasks that are NOT ready
// (have unclosed blockers OR have open/in_progress children).
const blockedQuery = `
SELECT t.id, t.title, t.status, t.priority
FROM tasks t
WHERE t.status = 'open'
  AND (
    EXISTS (
      SELECT 1 FROM dependencies d
      JOIN tasks blocker ON d.blocked_by = blocker.id
      WHERE d.task_id = t.id
        AND blocker.status NOT IN ('done', 'cancelled')
    )
    OR EXISTS (
      SELECT 1 FROM tasks child
      WHERE child.parent = t.id
        AND child.status IN ('open', 'in_progress')
    )
  )
ORDER BY t.priority ASC, t.created ASC
`

func (a *App) cmdBlocked(workDir string, args []string) error {
	return a.cmdListFiltered(workDir, blockedQuery)
}
