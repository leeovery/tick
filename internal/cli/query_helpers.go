package cli

import "strings"

// ReadyNoUnclosedBlockers returns the SQL NOT EXISTS subquery condition
// that excludes tasks with unclosed (not done/cancelled) blockers.
// Assumes the outer query aliases the tasks table as "t".
func ReadyNoUnclosedBlockers() string {
	return `NOT EXISTS (
				SELECT 1 FROM dependencies d
				JOIN tasks blocker ON blocker.id = d.blocked_by
				WHERE d.task_id = t.id
				  AND blocker.status NOT IN ('done', 'cancelled')
			)`
}

// ReadyNoOpenChildren returns the SQL NOT EXISTS subquery condition
// that excludes tasks with open or in-progress children.
// Assumes the outer query aliases the tasks table as "t".
func ReadyNoOpenChildren() string {
	return `NOT EXISTS (
				SELECT 1 FROM tasks child
				WHERE child.parent = t.id
				  AND child.status IN ('open', 'in_progress')
			)`
}

// ReadyConditions returns the complete set of SQL WHERE conditions that
// define a "ready" task: open status, no unclosed blockers, no open children.
func ReadyConditions() []string {
	return []string{
		`t.status = 'open'`,
		ReadyNoUnclosedBlockers(),
		ReadyNoOpenChildren(),
	}
}

// BlockedConditions returns the SQL WHERE conditions that define a "blocked"
// task: open status AND (has unclosed blockers OR has open children).
// This is the De Morgan inverse of the ready NOT EXISTS conditions.
func BlockedConditions() []string {
	return []string{
		`t.status = 'open'`,
		`(
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
			)`,
	}
}

// ReadyWhereClause returns the ready conditions joined as a single SQL
// WHERE clause fragment (without the WHERE keyword), suitable for embedding
// in larger queries like the stats ready count.
func ReadyWhereClause() string {
	return strings.Join(ReadyConditions(), "\n\t\t\t  AND ")
}
