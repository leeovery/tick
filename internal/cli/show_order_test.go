package cli

import (
	"strconv"
	"testing"
)

const (
	showParentID = "tick-f0000f"
	laterSecond  = "2026-01-19T10:00:01Z"
)

// childLine renders one stored child of showParentID with the given
// sequence, priority and creation instant.
func childLine(id, title string, seq, priority int, created string) string {
	return `{"id":"` + id + `","title":"` + title + `","status":"open","priority":` + strconv.Itoa(priority) +
		`,"parent":"` + showParentID + `","seq":` + strconv.Itoa(seq) +
		`,"created":"` + created + `","updated":"` + created + `"}`
}

func shownChildIDs(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	doc := decodeToonDoc(t, runToonCommand(t, dir, append([]string{"show", showParentID}, args...)...))
	var ids []string
	for _, row := range toonRows(t, doc, "children") {
		id, ok := row["id"].(string)
		if !ok {
			t.Fatalf("children row id = %#v, want a string", row["id"])
		}
		ids = append(ids, id)
	}
	return ids
}

// Authored c1, c2, c3; the IDs and the line order both contradict authoring order.
const (
	authoredC1 = "tick-bbb222"
	authoredC2 = "tick-ccc333"
	authoredC3 = "tick-aaa111"
)

func authoredChildrenLines() []string {
	return []string{
		seqLine(showParentID, "P", "", 1),
		childLine(authoredC2, "c2", 3, 2, sameSecond),
		childLine(authoredC3, "c3", 4, 2, sameSecond),
		childLine(authoredC1, "c1", 2, 2, sameSecond),
	}
}

func TestShowChildrenOrder(t *testing.T) {
	t.Run("it lists same-second children in authoring order against ID and line order", func(t *testing.T) {
		dir, _ := setupRawProject(t, authoredChildrenLines()...)

		assertIDOrder(t, shownChildIDs(t, dir), []string{authoredC1, authoredC2, authoredC3})
	})

	t.Run("it lists an earlier-authored child above a later one with better priority", func(t *testing.T) {
		const (
			earlierID = "tick-bbb222"
			laterID   = "tick-aaa111"
		)
		dir, _ := setupRawProject(t,
			seqLine(showParentID, "P", "", 1),
			childLine(laterID, "later", 3, 0, sameSecond),
			childLine(earlierID, "earlier", 2, 2, sameSecond),
		)

		assertIDOrder(t, shownChildIDs(t, dir), []string{earlierID, laterID})
	})

	t.Run("it lists an earlier-second child first despite its higher sequence", func(t *testing.T) {
		const (
			earlierSecondID = "tick-bbb222"
			laterSecondID   = "tick-aaa111"
		)
		dir, _ := setupRawProject(t,
			seqLine(showParentID, "P", "", 1),
			childLine(laterSecondID, "later second", 2, 2, laterSecond),
			childLine(earlierSecondID, "earlier second", 5, 2, sameSecond),
		)

		assertIDOrder(t, shownChildIDs(t, dir), []string{earlierSecondID, laterSecondID})
	})

	t.Run("it lists children sharing a sequence and a second in ascending ID order on every run", func(t *testing.T) {
		dir, _ := setupRawProject(t,
			seqLine(showParentID, "P", "", 1),
			childLine("tick-ccc333", "c", 7, 2, sameSecond),
			childLine("tick-aaa111", "a", 7, 2, sameSecond),
			childLine("tick-bbb222", "b", 7, 2, sameSecond),
		)

		want := []string{"tick-aaa111", "tick-bbb222", "tick-ccc333"}
		for range 3 {
			assertIDOrder(t, shownChildIDs(t, dir), want)
		}
	})

	for _, tc := range []struct {
		field string
		want  string
	}{
		{"children.1", authoredC1},
		{"children.3", authoredC3},
	} {
		t.Run("it selects the authored child for --field "+tc.field, func(t *testing.T) {
			dir, _ := setupRawProject(t, authoredChildrenLines()...)

			assertIDOrder(t, shownChildIDs(t, dir, "--field", tc.field), []string{tc.want})
		})
	}
}
