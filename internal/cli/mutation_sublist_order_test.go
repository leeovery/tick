package cli

import "testing"

// Children authored c1, c2, c3 in one second; IDs and line order contradict
// that. The blocker created later is declared first and carries the higher ID.
const (
	mutationParentID     = "tick-f0000f"
	mutationEarlyBlocker = "tick-000b01"
	mutationLateBlocker  = "tick-000e02"
)

var (
	mutationAuthoredChildren = []string{authoredC1, authoredC2, authoredC3}
	mutationDeclaredBlockers = []string{mutationLateBlocker, mutationEarlyBlocker}
)

func mutationSubListLines(t *testing.T) []string {
	t.Helper()
	return fixtureLines(t,
		fixtureTask{id: mutationParentID, status: "open", seq: 1, blockedBy: mutationDeclaredBlockers},
		fixtureTask{id: authoredC2, status: "open", parent: mutationParentID, seq: 3},
		fixtureTask{id: authoredC3, status: "open", parent: mutationParentID, seq: 4},
		fixtureTask{id: authoredC1, status: "open", parent: mutationParentID, seq: 2},
		fixtureTask{id: mutationEarlyBlocker, status: "open", seq: 5},
		fixtureTask{id: mutationLateBlocker, status: "open", seq: 6},
	)
}

func sectionIDs(t *testing.T, doc map[string]any, key string) []string {
	t.Helper()
	var ids []string
	for _, row := range toonRows(t, doc, key) {
		id, ok := row["id"].(string)
		if !ok {
			t.Fatalf("%s row id = %#v, want a string", key, row["id"])
		}
		ids = append(ids, id)
	}
	return ids
}

func assertSubListsMatchShow(t *testing.T, dir, output string) {
	t.Helper()
	doc := decodeToonDoc(t, output)
	shown := decodeToonDoc(t, runToonCommand(t, dir, "show", mutationParentID))

	assertIDOrder(t, sectionIDs(t, doc, "children"), mutationAuthoredChildren)
	assertIDOrder(t, sectionIDs(t, doc, "blocked_by"), mutationDeclaredBlockers)
	assertIDOrder(t, sectionIDs(t, doc, "children"), sectionIDs(t, shown, "children"))
	assertIDOrder(t, sectionIDs(t, doc, "blocked_by"), sectionIDs(t, shown, "blocked_by"))
}

func TestMutationDetailSubListOrder(t *testing.T) {
	t.Run("it lists children in creation order and blockers in declaration order after update", func(t *testing.T) {
		dir, _ := setupRawProject(t, mutationSubListLines(t)...)

		output := runToonCommand(t, dir, "update", mutationParentID, "--title", "Renamed")

		assertSubListsMatchShow(t, dir, output)
	})

	t.Run("it lists children in creation order and blockers in declaration order after note add", func(t *testing.T) {
		dir, _ := setupRawProject(t, mutationSubListLines(t)...)

		output := runToonCommand(t, dir, "note", "add", mutationParentID, "text")

		assertSubListsMatchShow(t, dir, output)
	})

	t.Run("it lists children in creation order and blockers in declaration order after note remove", func(t *testing.T) {
		dir, _ := setupRawProject(t, mutationSubListLines(t)...)
		runToonCommand(t, dir, "note", "add", mutationParentID, "text")

		output := runToonCommand(t, dir, "note", "remove", mutationParentID, "1")

		assertSubListsMatchShow(t, dir, output)
	})

	t.Run("it lists a later-created blocker declared first at the top after create", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: mutationEarlyBlocker, status: "open", seq: 1},
			fixtureTask{id: mutationLateBlocker, status: "open", seq: 2},
		)...)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "create", "T", "--blocked-by", mutationLateBlocker+","+mutationEarlyBlocker))
		createdID, ok := doc["id"].(string)
		if !ok {
			t.Fatalf("id = %#v, want a string", doc["id"])
		}

		blockers := sectionIDs(t, doc, "blocked_by")
		assertIDOrder(t, blockers, mutationDeclaredBlockers)
		assertIDOrder(t, storedBlockedBy(t, tickDir, createdID), blockers)
		assertIDOrder(t, shownBlockerIDs(t, dir, createdID), blockers)
	})
}
