package cli

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/leeovery/tick/internal/storage"
)

// Declared blocker order: first, second, third. Creation order: second, third,
// first. Ascending-ID order: second, third, first.
const (
	blockedTaskID       = "tick-f0000f"
	declaredFirstID     = "tick-ccc333"
	declaredSecondID    = "tick-aaa111"
	declaredThirdID     = "tick-bbb222"
	unrelatedBlockedID  = "tick-e0000e"
	unrelatedBlockerAID = "tick-000a01"
	unrelatedBlockerBID = "tick-000b02"
)

var declaredBlockerIDs = []string{declaredFirstID, declaredSecondID, declaredThirdID}

func declaredBlockersLines(t *testing.T) []string {
	t.Helper()
	return fixtureLines(t,
		fixtureTask{id: declaredSecondID, status: "open", seq: 1},
		fixtureTask{id: declaredThirdID, status: "open", seq: 2},
		fixtureTask{id: declaredFirstID, status: "open", seq: 3},
		fixtureTask{id: blockedTaskID, status: "open", seq: 4, blockedBy: declaredBlockerIDs},
	)
}

func shownBlockerIDs(t *testing.T, dir, id string, args ...string) []string {
	t.Helper()
	doc := decodeToonDoc(t, runToonCommand(t, dir, append([]string{"show", id}, args...)...))
	var ids []string
	for _, row := range toonRows(t, doc, "blocked_by") {
		blockerID, ok := row["id"].(string)
		if !ok {
			t.Fatalf("blocked_by row id = %#v, want a string", row["id"])
		}
		ids = append(ids, blockerID)
	}
	return ids
}

func storedBlockedBy(t *testing.T, tickDir, id string) []string {
	t.Helper()
	tasks, err := storage.ReadJSONL(filepath.Join(tickDir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	for _, tk := range tasks {
		if tk.ID == id {
			return tk.BlockedBy
		}
	}
	t.Fatalf("task %s not found in tasks.jsonl", id)
	return nil
}

func depTreeBlockersOf(t *testing.T, dir, id string) []string {
	t.Helper()
	doc := decodeToonDoc(t, runToonCommand(t, dir, "dep", "tree"))
	var ids []string
	for _, row := range toonRows(t, doc, "dep_tree") {
		if row["to"] != id {
			continue
		}
		from, ok := row["from"].(string)
		if !ok {
			t.Fatalf("dep_tree row from = %#v, want a string", row["from"])
		}
		ids = append(ids, from)
	}
	return ids
}

func cachedBlockersByOrdinal(t *testing.T, tickDir string) map[string][]string {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tickDir, "cache.db"))
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT task_id, blocked_by FROM dependencies ORDER BY task_id, ordinal`)
	if err != nil {
		t.Fatalf("failed to query dependency ordinals: %v", err)
	}
	defer rows.Close()
	got := map[string][]string{}
	for rows.Next() {
		var taskID, blocker string
		if err := rows.Scan(&taskID, &blocker); err != nil {
			t.Fatalf("failed to scan dependency row: %v", err)
		}
		got[taskID] = append(got[taskID], blocker)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("dependency row iteration: %v", err)
	}
	return got
}

func TestShowBlockersOrder(t *testing.T) {
	t.Run("it lists a later-created blocker declared first at the top", func(t *testing.T) {
		dir, _ := setupRawProject(t, declaredBlockersLines(t)...)

		assertIDOrder(t, shownBlockerIDs(t, dir, blockedTaskID), declaredBlockerIDs)
	})

	t.Run("it agrees with the stored blocked_by array and dep tree's edges", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, declaredBlockersLines(t)...)

		shown := shownBlockerIDs(t, dir, blockedTaskID)
		assertIDOrder(t, storedBlockedBy(t, tickDir, blockedTaskID), shown)
		assertIDOrder(t, depTreeBlockersOf(t, dir, blockedTaskID), shown)
	})

	t.Run("it lists same-second same-sequence blockers in declaration order on every run", func(t *testing.T) {
		const sharedSeq = 7
		dir, _ := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: declaredSecondID, status: "open", seq: sharedSeq},
			fixtureTask{id: declaredThirdID, status: "open", seq: sharedSeq},
			fixtureTask{id: declaredFirstID, status: "open", seq: sharedSeq},
			fixtureTask{id: blockedTaskID, status: "open", seq: sharedSeq, blockedBy: declaredBlockerIDs},
		)...)

		for range 3 {
			assertIDOrder(t, shownBlockerIDs(t, dir, blockedTaskID), declaredBlockerIDs)
		}
	})

	t.Run("it rebuilds a version 2 cache with dependency ordinals", func(t *testing.T) {
		unrelatedBlockers := []string{unrelatedBlockerBID, unrelatedBlockerAID}
		dir, tickDir := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: declaredSecondID, status: "open", seq: 1},
			fixtureTask{id: declaredThirdID, status: "open", seq: 2},
			fixtureTask{id: declaredFirstID, status: "open", seq: 3},
			fixtureTask{id: unrelatedBlockerAID, status: "open", seq: 4},
			fixtureTask{id: unrelatedBlockerBID, status: "open", seq: 5},
			fixtureTask{id: blockedTaskID, status: "open", seq: 6, blockedBy: declaredBlockerIDs},
			fixtureTask{id: unrelatedBlockedID, status: "open", seq: 7, blockedBy: unrelatedBlockers},
		)...)
		createV2Cache(t, tickDir)

		assertIDOrder(t, shownBlockerIDs(t, dir, blockedTaskID), declaredBlockerIDs)

		assertCacheSchemaVersion(t, tickDir, "3")
		got := cachedBlockersByOrdinal(t, tickDir)
		want := map[string][]string{
			blockedTaskID:      declaredBlockerIDs,
			unrelatedBlockedID: unrelatedBlockers,
		}
		if len(got) != len(want) {
			t.Fatalf("dependencies by task = %v, want %v", got, want)
		}
		for id, blockers := range want {
			assertIDOrder(t, got[id], blockers)
		}
	})

	t.Run("it selects the first-declared blocker for --field blocked_by.1", func(t *testing.T) {
		dir, _ := setupRawProject(t, declaredBlockersLines(t)...)

		assertIDOrder(t, shownBlockerIDs(t, dir, blockedTaskID, "--field", "blocked_by.1"), []string{declaredFirstID})
	})
}

func assertCacheSchemaVersion(t *testing.T, tickDir, want string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tickDir, "cache.db"))
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatalf("failed to read schema_version: %v", err)
	}
	if version != want {
		t.Errorf("schema_version = %q, want %q", version, want)
	}
}
