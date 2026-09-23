package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/storage"
)

// bareLine renders one stored task record with no seq field, created and
// updated at sameSecond with priority 2.
func bareLine(id, title string) string {
	return `{"id":"` + id + `","title":"` + title + `","status":"open","priority":2,"created":"` +
		sameSecond + `","updated":"` + sameSecond + `"}`
}

// storedRecord is the identity and sequence one tasks.jsonl record carries on
// disk, Seq 0 where it carries none.
type storedRecord struct {
	ID  string `json:"id"`
	Seq int    `json:"seq"`
}

// rawRecords decodes each record in tasks.jsonl, in record order, without the
// read-time backfill.
func rawRecords(t *testing.T, tickDir string) []storedRecord {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(tickDir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("failed to read tasks.jsonl: %v", err)
	}
	var records []storedRecord
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == "" {
			continue
		}
		var record storedRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to decode tasks.jsonl line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

// rawSeqs returns the seq each record in tasks.jsonl carries on disk, in
// record order, 0 where the record carries none.
func rawSeqs(t *testing.T, tickDir string) []int {
	t.Helper()
	var seqs []int
	for _, record := range rawRecords(t, tickDir) {
		seqs = append(seqs, record.Seq)
	}
	return seqs
}

func assertSeqsFollowRecordOrder(t *testing.T, tickDir string) {
	t.Helper()
	seqs := rawSeqs(t, tickDir)
	for i, seq := range seqs {
		if seq == 0 {
			t.Errorf("tasks.jsonl record %d carries no seq; seqs = %v", i+1, seqs)
			return
		}
		if i > 0 && seq <= seqs[i-1] {
			t.Errorf("tasks.jsonl seqs = %v, want ascending in record order", seqs)
			return
		}
	}
}

func assertCachedSeqs(t *testing.T, tickDir string, want map[string]int) {
	t.Helper()
	for id, seq := range want {
		if got := cachedSeq(t, tickDir, id); got != seq {
			t.Errorf("cache seq for %s = %d, want %d", id, got, seq)
		}
	}
}

// Merge shape: a, b, c at 1–3, then x and y with none, then d and e at 4 and
// 5. IDs descend through the expected order a, b, c, d, e, x, y.
const (
	mergeA = "tick-a00007"
	mergeB = "tick-a00006"
	mergeC = "tick-a00005"
	mergeD = "tick-a00004"
	mergeE = "tick-a00003"
	mergeX = "tick-a00002"
	mergeY = "tick-a00001"
)

func mergeShapeLines() []string {
	return []string{
		seqLine(mergeA, "a", "", 1),
		seqLine(mergeB, "b", "", 2),
		seqLine(mergeC, "c", "", 3),
		bareLine(mergeX, "x"),
		bareLine(mergeY, "y"),
		seqLine(mergeD, "d", "", 4),
		seqLine(mergeE, "e", "", 5),
	}
}

var mergeShapeSeqs = map[string]int{
	mergeA: 1, mergeB: 2, mergeC: 3, mergeD: 4, mergeE: 5, mergeX: 6, mergeY: 7,
}

// Three records authored first, second, third; IDs descend, so ascending-ID
// order is the reverse of line order.
const (
	legacyFirstID  = "tick-c00003"
	legacySecondID = "tick-b00002"
	legacyThirdID  = "tick-a00001"
)

func legacyLines() []string {
	return []string{
		bareLine(legacyFirstID, "first"),
		bareLine(legacySecondID, "second"),
		bareLine(legacyThirdID, "third"),
	}
}

func TestBackfillSequence(t *testing.T) {
	t.Run("it lists a file with no sequences in line order through update, remove and create", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, legacyLines()...)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacySecondID, legacyThirdID})

		runToonCommand(t, dir, "update", legacySecondID, "--title", "second renamed")
		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacySecondID, legacyThirdID})
		assertSeqsFollowRecordOrder(t, tickDir)

		runToonCommand(t, dir, "remove", legacySecondID, "--force")
		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacyThirdID})
		assertSeqsFollowRecordOrder(t, tickDir)

		newID := createTaskID(t, dir, tickDir, "fourth")
		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacyThirdID, newID})
		assertSeqsFollowRecordOrder(t, tickDir)
	})

	t.Run("it leaves tasks.jsonl byte-identical on list", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, legacyLines()...)
		path := filepath.Join(tickDir, "tasks.jsonl")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}

		runToonCommand(t, dir, "list")

		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read tasks.jsonl: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("tasks.jsonl changed on list:\n before = %q\n  after = %q", before, after)
		}
	})

	t.Run("it numbers a task created in a three-record file with no sequences 4", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, legacyLines()...)

		createTaskID(t, dir, tickDir, "new")

		seqs := rawSeqs(t, tickDir)
		if got := seqs[len(seqs)-1]; got != 4 {
			t.Errorf("new task seq = %d, want 4; seqs = %v", got, seqs)
		}
	})

	t.Run("it lists a newest unnumbered record after records numbered above their line positions", func(t *testing.T) {
		const newestID = "tick-a00001"
		dir, tickDir := setupRawProject(t,
			seqLine("tick-d00004", "one", "", 5),
			seqLine("tick-c00003", "two", "", 6),
			seqLine("tick-b00002", "three", "", 7),
			bareLine(newestID, "newest"),
		)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{"tick-d00004", "tick-c00003", "tick-b00002", newestID})
		assertCachedSeqs(t, tickDir, map[string]int{newestID: 8})
	})

	t.Run("it numbers a merge's unnumbered records above every carried sequence", func(t *testing.T) {
		dir, tickDir := setupRawProject(t, mergeShapeLines()...)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{mergeA, mergeB, mergeC, mergeD, mergeE, mergeX, mergeY})
		assertCachedSeqs(t, tickDir, mergeShapeSeqs)
	})

	t.Run("it restores order to a file stripped of sequences with an appended record last", func(t *testing.T) {
		const appendedID = "tick-000001"
		dir, _ := setupRawProject(t, append(legacyLines(), bareLine(appendedID, "appended"))...)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacySecondID, legacyThirdID, appendedID})
	})

	t.Run("it numbers by record order across blank lines", func(t *testing.T) {
		dir, tickDir := setupRawProject(t,
			bareLine(legacyFirstID, "first"), "",
			bareLine(legacySecondID, "second"), "", "",
			bareLine(legacyThirdID, "third"),
		)

		assertIDOrder(t, listedIDs(t, dir, "list"), []string{legacyFirstID, legacySecondID, legacyThirdID})
		assertCachedSeqs(t, tickDir, map[string]int{legacyFirstID: 1, legacySecondID: 2, legacyThirdID: 3})

		runToonCommand(t, dir, "update", legacyFirstID, "--title", "first renamed")
		if got := rawSeqs(t, tickDir); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
			t.Errorf("tasks.jsonl seqs = %v, want [1 2 3]", got)
		}
	})

	t.Run("it treats seq 0 exactly as an absent seq", func(t *testing.T) {
		zeroDir, zeroTickDir := setupRawProject(t,
			seqLine(mergeA, "a", "", 0),
			seqLine(mergeB, "b", "", 2),
			seqLine(mergeC, "c", "", 0),
		)
		bareDir, bareTickDir := setupRawProject(t,
			bareLine(mergeA, "a"),
			seqLine(mergeB, "b", "", 2),
			bareLine(mergeC, "c"),
		)

		want := []string{mergeB, mergeA, mergeC}
		assertIDOrder(t, listedIDs(t, zeroDir, "list"), want)
		assertIDOrder(t, listedIDs(t, bareDir, "list"), want)
		wantSeqs := map[string]int{mergeA: 3, mergeB: 2, mergeC: 4}
		assertCachedSeqs(t, zeroTickDir, wantSeqs)
		assertCachedSeqs(t, bareTickDir, wantSeqs)
	})

	t.Run("it assigns the same sequences through ReadTasks, rebuild and list", func(t *testing.T) {
		_, readTickDir := setupRawProject(t, mergeShapeLines()...)
		store, err := storage.NewStore(readTickDir)
		if err != nil {
			t.Fatalf("NewStore returned error: %v", err)
		}
		defer store.Close()
		tasks, err := store.ReadTasks()
		if err != nil {
			t.Fatalf("ReadTasks returned error: %v", err)
		}
		for _, tk := range tasks {
			if want := mergeShapeSeqs[tk.ID]; tk.Seq != want {
				t.Errorf("ReadTasks seq for %s = %d, want %d", tk.ID, tk.Seq, want)
			}
		}

		rebuildDir, rebuildTickDir := setupRawProject(t, mergeShapeLines()...)
		runToonCommand(t, rebuildDir, "rebuild")
		assertCachedSeqs(t, rebuildTickDir, mergeShapeSeqs)

		listDir, listTickDir := setupRawProject(t, mergeShapeLines()...)
		runToonCommand(t, listDir, "list")
		assertCachedSeqs(t, listTickDir, mergeShapeSeqs)
	})
}
