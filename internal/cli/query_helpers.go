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

// ReadyNoBlockedAncestor returns a SQL NOT EXISTS subquery with a recursive
// CTE that walks the ancestor chain via parent pointers. It excludes tasks
// where any ancestor has an unclosed dependency blocker.
// Assumes the outer query aliases the tasks table as "t".
func ReadyNoBlockedAncestor() string {
	return `NOT EXISTS (
				WITH RECURSIVE ancestors(id) AS (
					SELECT parent FROM tasks WHERE id = t.id AND parent IS NOT NULL
					UNION ALL
					SELECT t2.parent FROM tasks t2
					JOIN ancestors a ON t2.id = a.id
					WHERE t2.parent IS NOT NULL
				)
				SELECT 1 FROM ancestors a
				JOIN dependencies d ON d.task_id = a.id
				JOIN tasks blocker ON blocker.id = d.blocked_by
				WHERE blocker.status NOT IN ('done', 'cancelled')
			)`
}

// ReadyConditions returns the complete set of SQL WHERE conditions that
// define a "ready" task: open status, no unclosed blockers, no open children,
// no dependency-blocked ancestor.
func ReadyConditions() []string {
	return []string{
		`t.status = 'open'`,
		ReadyNoUnclosedBlockers(),
		ReadyNoOpenChildren(),
		ReadyNoBlockedAncestor(),
	}
}

// negateNotExists converts a "NOT EXISTS (...)" condition to "EXISTS (...)"
// by stripping the leading "NOT " prefix.
func negateNotExists(s string) string {
	return strings.TrimPrefix(s, "NOT ")
}

// BlockedConditions returns the SQL WHERE conditions that define a "blocked"
// task: open status AND (has unclosed blockers OR has open children OR has
// dependency-blocked ancestor). This is the De Morgan inverse of the ready
// NOT EXISTS conditions, derived from the ReadyNo*() helpers.
func BlockedConditions() []string {
	parts := []string{
		negateNotExists(ReadyNoUnclosedBlockers()),
		negateNotExists(ReadyNoOpenChildren()),
		negateNotExists(ReadyNoBlockedAncestor()),
	}
	return []string{
		`t.status = 'open'`,
		"(" + strings.Join(parts, "\n\t\t\t\tOR ") + ")",
	}
}

// ReadyWhereClause returns the ready conditions joined as a single SQL
// WHERE clause fragment (without the WHERE keyword), suitable for embedding
// in larger queries like the stats ready count.
func ReadyWhereClause() string {
	return strings.Join(ReadyConditions(), "\n\t\t\t  AND ")
}
