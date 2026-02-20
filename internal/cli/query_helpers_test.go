package cli

import (
	"strings"
	"testing"
)

func TestReadyConditions(t *testing.T) {
	t.Run("it provides a non-empty no-unclosed-blockers condition", func(t *testing.T) {
		cond := ReadyNoUnclosedBlockers()
		if cond == "" {
			t.Error("ReadyNoUnclosedBlockers() returned empty string")
		}
	})

	t.Run("it provides a non-empty no-open-children condition", func(t *testing.T) {
		cond := ReadyNoOpenChildren()
		if cond == "" {
			t.Error("ReadyNoOpenChildren() returned empty string")
		}
	})

	t.Run("it provides a non-empty no-blocked-ancestor condition", func(t *testing.T) {
		cond := ReadyNoBlockedAncestor()
		if cond == "" {
			t.Error("ReadyNoBlockedAncestor() returned empty string")
		}
		if !strings.Contains(cond, "NOT EXISTS") {
			t.Error("ReadyNoBlockedAncestor() should contain NOT EXISTS")
		}
		if !strings.Contains(cond, "WITH RECURSIVE") {
			t.Error("ReadyNoBlockedAncestor() should contain WITH RECURSIVE")
		}
	})

	t.Run("ReadyConditions returns status open plus all four conditions", func(t *testing.T) {
		conditions := ReadyConditions()
		if len(conditions) != 4 {
			t.Fatalf("ReadyConditions() returned %d conditions, want 4", len(conditions))
		}
		if conditions[0] != `t.status = 'open'` {
			t.Errorf("conditions[0] = %q, want %q", conditions[0], `t.status = 'open'`)
		}
		if conditions[1] != ReadyNoUnclosedBlockers() {
			t.Errorf("conditions[1] does not match ReadyNoUnclosedBlockers()")
		}
		if conditions[2] != ReadyNoOpenChildren() {
			t.Errorf("conditions[2] does not match ReadyNoOpenChildren()")
		}
		if conditions[3] != ReadyNoBlockedAncestor() {
			t.Errorf("conditions[3] does not match ReadyNoBlockedAncestor()")
		}
	})

	t.Run("BlockedCondition returns open AND negation of ready subconditions", func(t *testing.T) {
		conditions := BlockedConditions()
		if len(conditions) != 2 {
			t.Fatalf("BlockedConditions() returned %d conditions, want 2", len(conditions))
		}
		if conditions[0] != `t.status = 'open'` {
			t.Errorf("conditions[0] = %q, want %q", conditions[0], `t.status = 'open'`)
		}
		// The second condition should be the disjunction (EXISTS blockers OR EXISTS open children)
		if conditions[1] == "" {
			t.Error("conditions[1] is empty")
		}
	})

	t.Run("BlockedConditions includes ancestor blocker in OR clause", func(t *testing.T) {
		conditions := BlockedConditions()
		if len(conditions) != 2 {
			t.Fatalf("BlockedConditions() returned %d conditions, want 2", len(conditions))
		}
		if !strings.Contains(conditions[1], "ancestors") {
			t.Error("conditions[1] should contain 'ancestors' for the ancestor blocker CTE")
		}
	})

	t.Run("BlockedConditions derives subqueries from ReadyNo helpers", func(t *testing.T) {
		conditions := BlockedConditions()
		if len(conditions) != 2 {
			t.Fatalf("BlockedConditions() returned %d conditions, want 2", len(conditions))
		}
		orClause := conditions[1]

		// Each ReadyNo*() helper returns "NOT EXISTS (...)". The blocked
		// condition should use "EXISTS (...)" with the identical inner body.
		// Verify by stripping the "NOT " prefix from each helper and checking
		// that the resulting "EXISTS (...)" string appears in the OR clause.
		helpers := []struct {
			name   string
			output string
		}{
			{"ReadyNoUnclosedBlockers", ReadyNoUnclosedBlockers()},
			{"ReadyNoOpenChildren", ReadyNoOpenChildren()},
			{"ReadyNoBlockedAncestor", ReadyNoBlockedAncestor()},
		}
		for _, h := range helpers {
			negated := strings.TrimPrefix(h.output, "NOT ")
			if negated == h.output {
				t.Fatalf("%s() output does not start with 'NOT ': %q", h.name, h.output)
			}
			if !strings.Contains(orClause, negated) {
				t.Errorf("BlockedConditions OR clause does not contain negated %s().\nWant substring:\n%s\nGot:\n%s", h.name, negated, orClause)
			}
		}
	})

	t.Run("BlockedConditions contains no SQL literals beyond status check", func(t *testing.T) {
		conditions := BlockedConditions()
		if len(conditions) != 2 {
			t.Fatalf("BlockedConditions() returned %d conditions, want 2", len(conditions))
		}
		// The status condition is the only SQL literal allowed
		if conditions[0] != `t.status = 'open'` {
			t.Errorf("conditions[0] = %q, want %q", conditions[0], `t.status = 'open'`)
		}
		// The OR clause must not contain SELECT directly — it should be
		// composed from the ReadyNo*() helpers, not hand-written SQL.
		// If someone adds a hand-written EXISTS subquery, this catches it:
		// count the EXISTS occurrences and ensure they match exactly 3
		// (one per helper).
		orClause := conditions[1]
		existsCount := strings.Count(orClause, "EXISTS")
		// Each helper contributes one "EXISTS" — exactly 3 total
		if existsCount != 3 {
			t.Errorf("expected exactly 3 EXISTS in OR clause, got %d", existsCount)
		}
	})

	t.Run("ReadyWhereClause returns composable SQL WHERE fragment", func(t *testing.T) {
		clause := ReadyWhereClause()
		if clause == "" {
			t.Error("ReadyWhereClause() returned empty string")
		}
	})
}
