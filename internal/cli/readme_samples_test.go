package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	toon "github.com/toon-format/toon-go"

	"github.com/leeovery/tick/internal/task"
	"github.com/leeovery/tick/internal/testutil"
)

const (
	readmeListSampleAnchor       = "tasks[3]{id,title,status,priority,type}:"
	readmeFormatListAnchor       = "tasks[2]{id,title,status,priority,type}:"
	readmeShowSampleAnchor       = "id: tick-a1b2"
	readmeFieldSelectionAnchor   = "notes[2]{index,text,created}:"
	readmeTransitionAnchor       = "changed[1]{id,title,from,to,auto}:"
	readmeCascadeAnchor          = "changed[2]{id,title,from,to,auto}:"
	readmeDepTreeAnchor          = "dep_tree[2]{from,to}:"
	readmeEmptyTypeRow           = `tick-d5c6,Update docs,open,3,""`
	readmeMissingAnchorFixture   = "tasks[9]{nothing}:"
	readmeShellPromptLinePrefix  = "$ tick"
	readmePrettyListAnchor       = "ID          STATUS       PRI  TYPE     TITLE"
	readmePrettyFormatListAnchor = "ID          STATUS  PRI  TYPE     TITLE"
	readmePrettyDepTreeAnchor    = "tick-a1b2  Setup auth (done)"
	readmePrettyStartAnchor      = "tick-a1b2: open → in_progress"
	readmePrettyDoneAnchor       = "tick-a1b2: in_progress → done"
	readmeJSONObjectAnchor       = "{"
	readmeJSONArrayAnchor        = "["
)

type readmeFence struct {
	info string
	body string
}

func readmeContent(t *testing.T) string {
	t.Helper()
	path := filepath.Join(testutil.FindRepoRoot(t), "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	return string(content)
}

// readmeFences returns the README's fenced blocks with their info string, each
// body stripped of any leading shell prompt line.
func readmeFences(t *testing.T) []readmeFence {
	t.Helper()

	var fences []readmeFence
	var current []string
	info := ""
	inBlock := false
	for line := range strings.SplitSeq(readmeContent(t), "\n") {
		if !strings.HasPrefix(line, "```") {
			if inBlock {
				current = append(current, line)
			}
			continue
		}
		if !inBlock {
			inBlock = true
			info = strings.TrimSpace(strings.TrimPrefix(line, "```"))
			current = nil
			continue
		}
		if body, ok := fenceBody(current); ok {
			fences = append(fences, readmeFence{info: info, body: body})
		}
		inBlock = false
	}
	return fences
}

// readmeToonBlocks returns the README's fenced blocks that carry no info
// string, keyed by their first line once any shell prompt line is stripped.
func readmeToonBlocks(t *testing.T) map[string]string {
	t.Helper()

	blocks := make(map[string]string)
	for _, fence := range readmeFences(t) {
		if fence.info != "" {
			continue
		}
		anchor, _, _ := strings.Cut(fence.body, "\n")
		blocks[anchor] = fence.body
	}
	return blocks
}

func fenceBody(lines []string) (string, bool) {
	if len(lines) > 0 && strings.HasPrefix(lines[0], readmeShellPromptLinePrefix) {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

func readmeChangedRows(t *testing.T, blocks map[string]string, anchor string) []map[string]any {
	t.Helper()
	block, ok := blocks[anchor]
	if !ok {
		t.Fatalf("no README toon sample anchored on %q", anchor)
	}
	decoded, err := toon.DecodeString(block)
	if err != nil {
		t.Fatalf("README toon sample anchored on %q does not decode: %v", anchor, err)
	}
	return changedRowsOf(t, decoded, anchor)
}

func changedRowsOf(t *testing.T, decoded any, source string) []map[string]any {
	t.Helper()
	document, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("%s sample is %T, want an object", source, decoded)
	}
	if keys := slices.Sorted(maps.Keys(document)); !slices.Equal(keys, []string{"changed"}) {
		t.Fatalf("%s sample has keys %v, want only [changed]", source, keys)
	}
	entries, ok := document["changed"].([]any)
	if !ok {
		t.Fatalf("%s sample changed is %T, want a list", source, document["changed"])
	}
	rows := make([]map[string]any, 0, len(entries))
	for i, entry := range entries {
		row, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("%s sample row %d is %T, want an object", source, i, entry)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestREADMEToonSamplesDecode(t *testing.T) {
	blocks := readmeToonBlocks(t)

	t.Run("it carries the title column in the README samples", func(t *testing.T) {
		for _, anchor := range []string{readmeTransitionAnchor, readmeCascadeAnchor} {
			for i, row := range readmeChangedRows(t, blocks, anchor) {
				if title, _ := row["title"].(string); title == "" {
					t.Errorf("sample %q row %d has no title: %v", anchor, i, row)
				}
			}
		}
	})

	t.Run("it marks only cascaded rows as automatic", func(t *testing.T) {
		for anchor, want := range map[string][]bool{
			readmeTransitionAnchor: {false},
			readmeCascadeAnchor:    {false, true},
		} {
			rows := readmeChangedRows(t, blocks, anchor)
			if len(rows) != len(want) {
				t.Fatalf("sample %q has %d rows, want %d", anchor, len(rows), len(want))
			}
			for i, row := range rows {
				if row["auto"] != want[i] {
					t.Errorf("sample %q row %d auto = %#v, want %v", anchor, i, row["auto"], want[i])
				}
			}
		}
	})

	t.Run("it renders an empty type as a quoted empty string", func(t *testing.T) {
		block, ok := blocks[readmeListSampleAnchor]
		if !ok {
			t.Fatalf("no README toon sample anchored on %q", readmeListSampleAnchor)
		}
		lines := strings.Split(block, "\n")
		if len(lines) != 4 {
			t.Fatalf("list sample has %d lines, want 4:\n%s", len(lines), block)
		}
		if got := strings.TrimSpace(lines[3]); got != readmeEmptyTypeRow {
			t.Errorf("empty-type row = %q, want %q", got, readmeEmptyTypeRow)
		}
	})
}

func TestREADMETransitionSamples(t *testing.T) {
	t.Run("it parses the README JSON transition sample", func(t *testing.T) {
		var objects []string
		for _, fence := range readmeFences(t) {
			if fence.info == "json" && strings.HasPrefix(fence.body, "{") {
				objects = append(objects, fence.body)
			}
		}
		if len(objects) != 1 {
			t.Fatalf("README has %d JSON object samples, want 1", len(objects))
		}

		var decoded any
		if err := json.Unmarshal([]byte(objects[0]), &decoded); err != nil {
			t.Fatalf("README JSON transition sample does not parse: %v", err)
		}
		rows := changedRowsOf(t, decoded, "README JSON transition")
		if len(rows) != 1 {
			t.Fatalf("JSON transition sample has %d rows, want 1", len(rows))
		}
		if _, ok := rows[0]["auto"].(bool); !ok {
			t.Errorf("JSON transition auto = %#v, want an unquoted boolean", rows[0]["auto"])
		}
	})
}

// readmeSample pairs a README fenced block with the seeded project and command
// that must render it byte-for-byte. occurrence selects among blocks sharing an
// info string and first line, counting from one in README order.
type readmeSample struct {
	name       string
	info       string
	firstLine  string
	occurrence int
	format     string
	args       []string
	tasks      []task.Task
}

type readmeSampleGroup struct {
	name    string
	samples []readmeSample
}

func readmeTimestamp(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(task.TimestampFormat, value)
	if err != nil {
		t.Fatalf("cannot parse timestamp %q: %v", value, err)
	}
	return parsed
}

func readmeSampleGroups(t *testing.T) []readmeSampleGroup {
	t.Helper()

	at := func(value string) time.Time { return readmeTimestamp(t, value) }

	depTreeTasks := []task.Task{
		{ID: "tick-a1b2", Title: "Setup auth", Status: task.StatusDone, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
		{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Type: "task", BlockedBy: []string{"tick-a1b2"}, Created: at("2026-01-19T10:01:00Z"), Updated: at("2026-01-19T10:01:00Z")},
		{ID: "tick-f3e4", Title: "Write tests", Status: task.StatusOpen, Priority: 2, Type: "task", BlockedBy: []string{"tick-c3d4"}, Created: at("2026-01-19T10:02:00Z"), Updated: at("2026-01-19T10:02:00Z")},
	}

	threeTaskList := []task.Task{
		{ID: "tick-a1b2", Title: "Auth middleware", Status: task.StatusInProgress, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
		{ID: "tick-f3e4", Title: "Write tests", Status: task.StatusOpen, Priority: 2, Type: "task", Created: at("2026-01-19T10:01:00Z"), Updated: at("2026-01-19T10:01:00Z")},
		{ID: "tick-d5c6", Title: "Update docs", Status: task.StatusOpen, Priority: 3, Created: at("2026-01-19T10:02:00Z"), Updated: at("2026-01-19T10:02:00Z")},
	}

	twoTaskList := []task.Task{
		{ID: "tick-a1b2", Title: "Setup auth", Status: task.StatusDone, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
		{ID: "tick-c3d4", Title: "Login endpoint", Status: task.StatusOpen, Priority: 1, Type: "task", Created: at("2026-01-19T10:01:00Z"), Updated: at("2026-01-19T10:01:00Z")},
	}

	showTasks := []task.Task{
		{
			ID:          "tick-a1b2",
			Title:       "Setup auth",
			Status:      task.StatusInProgress,
			Priority:    1,
			Type:        "feature",
			Tags:        []string{"auth", "backend"},
			Refs:        []string{"https://github.com/org/repo/issues/42"},
			Description: "Full task description here.\nCan be multiple lines.",
			Notes:       []task.Note{{Text: "Discussed approach with team", Created: at("2026-01-19T14:00:00Z")}},
			BlockedBy:   []string{"tick-c3d4"},
			Created:     at("2026-01-19T10:00:00Z"),
			Updated:     at("2026-01-19T14:30:00Z"),
		},
		{ID: "tick-c3d4", Title: "Database migrations", Status: task.StatusDone, Priority: 1, Type: "task", Created: at("2026-01-19T09:00:00Z"), Updated: at("2026-01-19T09:30:00Z")},
	}

	startTasks := []task.Task{
		{ID: "tick-a1b2", Title: "Setup auth", Status: task.StatusOpen, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
	}

	cascadeTasks := []task.Task{
		{ID: "tick-a1b2", Title: "Setup auth", Status: task.StatusInProgress, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
		{ID: "tick-c3d4", Title: "Subtask one", Status: task.StatusOpen, Priority: 1, Type: "task", Parent: "tick-a1b2", Created: at("2026-01-19T10:01:00Z"), Updated: at("2026-01-19T10:01:00Z")},
	}

	fieldSelectionTasks := []task.Task{
		{
			ID:          "tick-a1b2",
			Title:       "Setup auth",
			Status:      task.StatusInProgress,
			Priority:    1,
			Type:        "feature",
			Tags:        []string{"auth", "backend"},
			Refs:        []string{"https://github.com/org/repo/issues/42"},
			Description: "Full task description here.\nCan be multiple lines.",
			Notes: []task.Note{
				{Text: "Discussed approach with team", Created: at("2026-01-19T14:00:00Z")},
				{Text: "Blocked on the migration landing", Created: at("2026-01-19T15:00:00Z")},
			},
			BlockedBy: []string{"tick-c3d4"},
			Created:   at("2026-01-19T10:00:00Z"),
			Updated:   at("2026-01-19T14:30:00Z"),
		},
		{ID: "tick-c3d4", Title: "Database migrations", Status: task.StatusDone, Priority: 1, Type: "task", Created: at("2026-01-19T09:00:00Z"), Updated: at("2026-01-19T09:30:00Z")},
	}

	jsonListTasks := []task.Task{
		{ID: "tick-a1b2", Title: "Setup auth", Status: task.StatusInProgress, Priority: 1, Type: "feature", Created: at("2026-01-19T10:00:00Z"), Updated: at("2026-01-19T10:00:00Z")},
	}

	return []readmeSampleGroup{
		{
			name: "it reproduces the README show sample",
			samples: []readmeSample{
				{name: "show toon", firstLine: readmeShowSampleAnchor, occurrence: 1, format: "--toon", args: []string{"show", "tick-a1b2"}, tasks: showTasks},
			},
		},
		{
			name: "it reproduces the README field selection sample",
			samples: []readmeSample{
				{name: "show filtered toon", firstLine: readmeFieldSelectionAnchor, occurrence: 1, format: "--toon", args: []string{"show", "tick-a1b2", "--field", "description,notes"}, tasks: fieldSelectionTasks},
			},
		},
		{
			name: "it reproduces the README list samples",
			samples: []readmeSample{
				{name: "list toon", firstLine: readmeListSampleAnchor, occurrence: 1, format: "--toon", args: []string{"list"}, tasks: threeTaskList},
				{name: "list pretty", firstLine: readmePrettyListAnchor, occurrence: 1, format: "--pretty", args: []string{"list"}, tasks: threeTaskList},
				{name: "toon format list", firstLine: readmeFormatListAnchor, occurrence: 1, format: "--toon", args: []string{"list"}, tasks: twoTaskList},
				{name: "pretty format list", firstLine: readmePrettyFormatListAnchor, occurrence: 1, format: "--pretty", args: []string{"list"}, tasks: twoTaskList},
				{name: "list json", info: "json", firstLine: readmeJSONArrayAnchor, occurrence: 1, format: "--json", args: []string{"list"}, tasks: jsonListTasks},
			},
		},
		{
			name: "it reproduces the README transition samples",
			samples: []readmeSample{
				{name: "start toon", firstLine: readmeTransitionAnchor, occurrence: 1, format: "--toon", args: []string{"start", "tick-a1b2"}, tasks: startTasks},
				{name: "start pretty", firstLine: readmePrettyStartAnchor, occurrence: 1, format: "--pretty", args: []string{"start", "tick-a1b2"}, tasks: startTasks},
				{name: "start json", info: "json", firstLine: readmeJSONObjectAnchor, occurrence: 1, format: "--json", args: []string{"start", "tick-a1b2"}, tasks: startTasks},
				{name: "done toon", firstLine: readmeCascadeAnchor, occurrence: 1, format: "--toon", args: []string{"done", "tick-a1b2"}, tasks: cascadeTasks},
				{name: "done pretty", firstLine: readmePrettyDoneAnchor, occurrence: 1, format: "--pretty", args: []string{"done", "tick-a1b2"}, tasks: cascadeTasks},
			},
		},
		{
			name: "it reproduces the README dep tree samples",
			samples: []readmeSample{
				{name: "dep tree pretty", firstLine: readmePrettyDepTreeAnchor, occurrence: 1, format: "--pretty", args: []string{"dep", "tree"}, tasks: depTreeTasks},
				{name: "dep tree toon", firstLine: readmeDepTreeAnchor, occurrence: 1, format: "--toon", args: []string{"dep", "tree"}, tasks: depTreeTasks},
			},
		},
	}
}

// findREADMEBlock returns the body of the fence matching the sample's info
// string, first line and occurrence.
func findREADMEBlock(fences []readmeFence, sample readmeSample) (string, error) {
	seen := 0
	for _, fence := range fences {
		if fence.info != sample.info {
			continue
		}
		if first, _, _ := strings.Cut(fence.body, "\n"); first != sample.firstLine {
			continue
		}
		seen++
		if seen == sample.occurrence {
			return fence.body, nil
		}
	}
	return "", fmt.Errorf("no README block with info %q, first line %q, occurrence %d", sample.info, sample.firstLine, sample.occurrence)
}

func runREADMECommand(t *testing.T, dir string, format string, args []string) string {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	app := &App{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Getwd:  func() (string, error) { return dir, nil },
	}
	full := append([]string{"tick", format}, args...)
	if code := app.Run(full); code != 0 {
		t.Fatalf("%v exit code = %d, want 0; stderr = %q", full, code, stderrBuf.String())
	}
	return stdoutBuf.String()
}

func TestREADMESamplesMatchRenderedOutput(t *testing.T) {
	fences := readmeFences(t)

	for _, group := range readmeSampleGroups(t) {
		t.Run(group.name, func(t *testing.T) {
			for _, sample := range group.samples {
				t.Run(sample.name, func(t *testing.T) {
					block, err := findREADMEBlock(fences, sample)
					if err != nil {
						t.Fatalf("%s: %v", sample.name, err)
					}
					dir, _ := setupTickProjectWithTasks(t, sample.tasks)
					got := strings.TrimRight(runREADMECommand(t, dir, sample.format, sample.args), "\n")
					if got != block {
						t.Errorf("%s output does not match its README block\ngot:\n%s\nwant:\n%s", sample.name, got, block)
					}
				})
			}
		})
	}

	t.Run("it fails when a documented sample has no matching README block", func(t *testing.T) {
		missing := readmeSample{firstLine: readmeMissingAnchorFixture, occurrence: 1}
		if _, err := findREADMEBlock(fences, missing); err == nil {
			t.Errorf("expected an error for first line %q, got nil", readmeMissingAnchorFixture)
		}
	})
}

// readmeShowSection returns the body of the README's `### show` section.
func readmeShowSection(t *testing.T) string {
	t.Helper()
	_, after, ok := strings.Cut(readmeContent(t), "\n### `show`\n")
	if !ok {
		t.Fatal("README has no `### show` section")
	}
	section, _, _ := strings.Cut(after, "\n### ")
	return section
}

func TestREADMEDocumentsFieldSelection(t *testing.T) {
	section := readmeShowSection(t)

	t.Run("it documents both field flag spellings", func(t *testing.T) {
		for _, spelling := range []string{"--field", "--fields"} {
			if !strings.Contains(section, backticked(spelling)) {
				t.Errorf("README `show` section does not document %s", spelling)
			}
		}
	})

	t.Run("it matches the help text for show", func(t *testing.T) {
		for _, flag := range showHelpFlagNames(t) {
			if !strings.Contains(section, backticked(flag)) {
				t.Errorf("README `show` section does not document %s, which `tick help show` lists", flag)
			}
		}
	})
}

func backticked(flag string) string {
	return "`" + flag + "`"
}

// showHelpFlagNames returns every long flag name listed in show's help entry.
func showHelpFlagNames(t *testing.T) []string {
	t.Helper()
	command := findCommand("show")
	if command == nil {
		t.Fatal("no help entry for show")
	}
	var names []string
	for _, flag := range command.Flags {
		for spelling := range strings.SplitSeq(flag.Name, ", ") {
			if strings.HasPrefix(spelling, "--") {
				names = append(names, spelling)
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("show's help entry lists no long flags")
	}
	return names
}
