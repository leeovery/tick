package cli

import "testing"

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

	t.Run("ReadyConditions returns status open plus both NOT EXISTS conditions", func(t *testing.T) {
		conditions := ReadyConditions()
		if len(conditions) != 3 {
			t.Fatalf("ReadyConditions() returned %d conditions, want 3", len(conditions))
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

	t.Run("ReadyWhereClause returns composable SQL WHERE fragment", func(t *testing.T) {
		clause := ReadyWhereClause()
		if clause == "" {
			t.Error("ReadyWhereClause() returned empty string")
		}
	})
}
