package cli

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/storage"
)

const sameSecond = "2026-01-19T10:00:00Z"

// Authored P, step-1 … step-5; IDs descend, so ascending-ID order is the
// reverse of authoring order.
const (
	phaseParentID = "tick-f0000f"
	phaseStep1ID  = "tick-e00005"
	phaseStep2ID  = "tick-d00004"
	phaseStep3ID  = "tick-c00003"
	phaseStep4ID  = "tick-b00002"
	phaseStep5ID  = "tick-a00001"
)

var phaseStepIDs = []string{phaseStep1ID, phaseStep2ID, phaseStep3ID, phaseStep4ID, phaseStep5ID}

// seqLine renders one stored task record carrying a sequence, created and
// updated at sameSecond with priority 2.
func seqLine(id, title, parent string, seq int) string {
	parentField := ""
	if parent != "" {
		parentField = `"parent":"` + parent + `",`
	}
	return `{"id":"` + id + `","title":"` + title + `","status":"open","priority":2,` + parentField +
		`"seq":` + strconv.Itoa(seq) + `,"created":"` + sameSecond + `","updated":"` + sameSecond + `"}`
}

// setupRawProject writes lines verbatim to .tick/tasks.jsonl.
func setupRawProject(t *testing.T, lines ...string) (string, string) {
	t.Helper()
	dir, tickDir := setupTickProject(t)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(tickDir, "tasks.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tasks.jsonl: %v", err)
	}
	return dir, tickDir
}

// phaseLines is a parent and five children sharing one creation second and
// one priority, written and sequenced in authoring order.
func phaseLines() []string {
	lines := []string{seqLine(phaseParentID, "Phase P", "", 1)}
	for i, id := range phaseStepIDs {
		lines = append(lines, seqLine(id, "step-"+strconv.Itoa(i+1), phaseParentID, i+2))
	}
	return lines
}

func listedIDs(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	doc := decodeToonDoc(t, runToonCommand(t, dir, args...))
	var ids []string
	for _, row := range toonRows(t, doc, "tasks") {
		id, ok := row["id"].(string)
		if !ok {
			t.Fatalf("row id = %#v, want a string", row["id"])
		}
		ids = append(ids, id)
	}
	return ids
}

func assertIDOrder(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestSameSecondOrdering(t *testing.T) {
	t.Run("it lists same-second tasks in sequence order", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		want := append([]string{phaseParentID}, phaseStepIDs...)
		assertIDOrder(t, listedIDs(t, dir, "list"), want)
	})

	t.Run("it lists a parent's same-second children in sequence order", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		assertIDOrder(t, listedIDs(t, dir, "list", "--parent", phaseParentID), phaseStepIDs)
	})

	t.Run("it returns the first-authored child for ready --count 1", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		got := listedIDs(t, dir, "ready", "--parent", phaseParentID, "--count", "1")
		assertIDOrder(t, got, []string{phaseStep1ID})
	})

	t.Run("it orders by sequence against both line order and ID order", func(t *testing.T) {
		// Sequence order: first, second, third. Line order: third, first,
		// second. Ascending-ID order: second, third, first.
		const (
			firstID  = "tick-ccc333"
			secondID = "tick-aaa111"
			thirdID  = "tick-bbb222"
		)
		dir, _ := setupRawProject(t,
			seqLine(thirdID, "third", "", 3),
			seqLine(firstID, "first", "", 1),
			seqLine(secondID, "second", "", 2),
		)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{firstID, secondID, thirdID})
	})
}

const v2SchemaSQL = `
CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority INTEGER NOT NULL DEFAULT 2,
  description TEXT,
  type TEXT,
  parent TEXT,
  created TEXT NOT NULL,
  updated TEXT NOT NULL,
  closed TEXT
);
CREATE TABLE dependencies (task_id TEXT NOT NULL, blocked_by TEXT NOT NULL, PRIMARY KEY (task_id, blocked_by));
CREATE TABLE task_tags (task_id TEXT NOT NULL, tag TEXT NOT NULL, PRIMARY KEY (task_id, tag));
CREATE TABLE task_refs (task_id TEXT NOT NULL, ref TEXT NOT NULL, PRIMARY KEY (task_id, ref));
CREATE TABLE task_notes (task_id TEXT NOT NULL, text TEXT NOT NULL, created TEXT NOT NULL);
CREATE TABLE task_transitions (task_id TEXT NOT NULL, from_status TEXT NOT NULL, to_status TEXT NOT NULL, at TEXT NOT NULL, auto INTEGER NOT NULL DEFAULT 0);
CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
`

// createV2Cache writes a cache.db shaped as schema version 2, holding the
// project's tasks and a hash matching its tasks.jsonl, so only the version is
// stale.
func createV2Cache(t *testing.T, tickDir string) {
	t.Helper()
	tasks, err := storage.ReadJSONL(filepath.Join(tickDir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(tickDir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tickDir, "cache.db"))
	if err != nil {
		t.Fatalf("failed to open cache.db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(v2SchemaSQL); err != nil {
		t.Fatalf("failed to create v2 schema: %v", err)
	}
	for _, tk := range tasks {
		if _, err := db.Exec(
			`INSERT INTO tasks (id, title, status, priority, parent, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tk.ID, tk.Title, string(tk.Status), tk.Priority, tk.Parent, sameSecond, sameSecond,
		); err != nil {
			t.Fatalf("failed to insert v2 task row: %v", err)
		}
		for _, blocker := range tk.BlockedBy {
			if _, err := db.Exec(`INSERT INTO dependencies (task_id, blocked_by) VALUES (?, ?)`, tk.ID, blocker); err != nil {
				t.Fatalf("failed to insert v2 dependency row: %v", err)
			}
		}
	}
	hash := sha256.Sum256(raw)
	if _, err := db.Exec(
		`INSERT INTO metadata (key, value) VALUES ('jsonl_hash', ?), ('schema_version', '2')`,
		hex.EncodeToString(hash[:]),
	); err != nil {
		t.Fatalf("failed to insert v2 metadata: %v", err)
	}
}

func TestSchemaV2CacheUpgrade(t *testing.T) {
	t.Run("it rebuilds a version 2 cache at version 3 with seq populated", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, phaseLines()...)
		createV2Cache(t, tickDir)

		runToonCommand(t, dir, "list")

		db, err := sql.Open("sqlite", filepath.Join(tickDir, "cache.db"))
		if err != nil {
			t.Fatalf("failed to open cache.db: %v", err)
		}
		defer db.Close()

		var version string
		if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'schema_version'`).Scan(&version); err != nil {
			t.Fatalf("failed to read schema_version: %v", err)
		}
		if version != "3" {
			t.Errorf("schema_version = %q, want %q", version, "3")
		}

		rows, err := db.Query(`SELECT id, seq FROM tasks`)
		if err != nil {
			t.Fatalf("failed to query seq column: %v", err)
		}
		defer rows.Close()
		got := map[string]int{}
		for rows.Next() {
			var id string
			var seq int
			if err := rows.Scan(&id, &seq); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}
			got[id] = seq
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("row iteration: %v", err)
		}

		want := map[string]int{phaseParentID: 1}
		for i, id := range phaseStepIDs {
			want[id] = i + 2
		}
		if len(got) != len(want) {
			t.Fatalf("tasks rows = %v, want %v", got, want)
		}
		for id, seq := range want {
			if got[id] != seq {
				t.Errorf("seq for %s = %d, want %d", id, got[id], seq)
			}
		}
	})
}

// hasKey reports whether key appears as an object key anywhere in v.
func hasKey(v any, key string) bool {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			if k == key || hasKey(child, key) {
				return true
			}
		}
	case []any:
		for _, child := range val {
			if hasKey(child, key) {
				return true
			}
		}
	}
	return false
}

func TestSequenceNotSurfaced(t *testing.T) {
	t.Run("it carries no seq key in toon show output", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		doc := decodeToonDoc(t, runToonCommand(t, dir, "show", phaseStep1ID))
		if hasKey(doc, "seq") {
			t.Errorf("toon show output carries a seq key: %v", doc)
		}
	})

	t.Run("it carries no seq key in JSON show output", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		if code := app.Run([]string{"tick", "--json", "show", phaseStep1ID}); code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderrBuf.String())
		}
		stdout := stdoutBuf.String()
		var doc any
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("show --json output is not JSON: %v\n%s", err, stdout)
		}
		if hasKey(doc, "seq") {
			t.Errorf("JSON show output carries a seq key: %s", stdout)
		}
	})

	t.Run("it carries no seq line in pretty show output", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		stdout, stderr, code := runShow(t, dir, phaseStep1ID)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr)
		}
		if strings.Contains(strings.ToLower(stdout), "seq") {
			t.Errorf("pretty show output mentions seq:\n%s", stdout)
		}
	})

	t.Run("it rejects seq as an unknown show field", func(t *testing.T) {
		dir, _ := setupRawProject(t, phaseLines()...)

		_, stderr, code := runShow(t, dir, phaseStep1ID, "--field", "seq")
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		want := "Error: unknown field \"seq\" for \"show\". Run 'tick help show' for usage.\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})
}

// fixtureTask is one stored task record for ordering fixtures; every record is
// created and updated at sameSecond with priority 2.
type fixtureTask struct {
	id, status, parent string
	blockedBy          []string
	seq                int
}

func (f fixtureTask) line(t *testing.T) string {
	t.Helper()
	record := map[string]any{
		"id":       f.id,
		"title":    "task " + f.id,
		"status":   f.status,
		"priority": 2,
		"seq":      f.seq,
		"created":  sameSecond,
		"updated":  sameSecond,
	}
	if f.parent != "" {
		record["parent"] = f.parent
	}
	if len(f.blockedBy) > 0 {
		record["blocked_by"] = f.blockedBy
	}
	b, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal fixture task: %v", err)
	}
	return string(b)
}

func fixtureLines(t *testing.T, tasks ...fixtureTask) []string {
	t.Helper()
	lines := make([]string, len(tasks))
	for i, f := range tasks {
		lines[i] = f.line(t)
	}
	return lines
}

func TestTotalListFamilyOrdering(t *testing.T) {
	t.Run("it keeps a later-authored in_progress task above tied open tasks in ready", func(t *testing.T) {
		const (
			open1ID      = "tick-e00001"
			open2ID      = "tick-d00002"
			open3ID      = "tick-c00003"
			inProgressID = "tick-a00004"
		)
		dir, _ := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: open1ID, status: "open", seq: 1},
			fixtureTask{id: open2ID, status: "open", seq: 2},
			fixtureTask{id: open3ID, status: "open", seq: 3},
			fixtureTask{id: inProgressID, status: "in_progress", seq: 4},
		)...)

		want := []string{inProgressID, open1ID, open2ID, open3ID}
		assertIDOrder(t, listedIDs(t, dir, "ready"), want)
	})

	t.Run("it lists same-second blocked tasks in authoring order", func(t *testing.T) {
		const (
			blockerID  = "tick-000b10"
			blocked1ID = "tick-e00001"
			blocked2ID = "tick-d00002"
			blocked3ID = "tick-c00003"
		)
		blockers := []string{blockerID}
		dir, _ := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: blockerID, status: "open", seq: 1},
			fixtureTask{id: blocked1ID, status: "open", seq: 2, blockedBy: blockers},
			fixtureTask{id: blocked2ID, status: "open", seq: 3, blockedBy: blockers},
			fixtureTask{id: blocked3ID, status: "open", seq: 4, blockedBy: blockers},
		)...)

		want := []string{blocked1ID, blocked2ID, blocked3ID}
		assertIDOrder(t, listedIDs(t, dir, "blocked"), want)
	})

	t.Run("it falls back to ascending ID order for children sharing a sequence", func(t *testing.T) {
		// Line order c, a, d, b contradicts ascending-ID order a, b, c, d.
		const (
			parentID  = "tick-f00000"
			childA    = "tick-a00000"
			childB    = "tick-b00000"
			childC    = "tick-c00000"
			childD    = "tick-d00000"
			sharedSeq = 7
		)
		dir, _ := setupRawProject(t, fixtureLines(t,
			fixtureTask{id: parentID, status: "open", seq: 1},
			fixtureTask{id: childC, status: "open", parent: parentID, seq: sharedSeq},
			fixtureTask{id: childA, status: "open", parent: parentID, seq: sharedSeq},
			fixtureTask{id: childD, status: "open", parent: parentID, seq: sharedSeq},
			fixtureTask{id: childB, status: "open", parent: parentID, seq: sharedSeq},
		)...)

		children := []string{childA, childB, childC, childD}
		for run := range 3 {
			assertIDOrder(t, listedIDs(t, dir, "list"), append([]string{parentID}, children...))
			assertIDOrder(t, listedIDs(t, dir, "list", "--parent", parentID), children)
			assertIDOrder(t, listedIDs(t, dir, "ready"), children)
			if t.Failed() {
				t.Fatalf("order diverged on run %d", run+1)
			}
		}
	})
}
