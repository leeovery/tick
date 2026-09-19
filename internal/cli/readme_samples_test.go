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
	readmeBareDescriptionAnchor  = "Full task description here."
	readmeBareNoteAnchor         = "Blocked on the migration landing"
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
	info   string
	prompt string
	body   string
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

	return fencesIn(readmeContent(t))
}

// fencesIn returns the fenced blocks of a markdown document with their info
// string, each body stripped of any leading shell prompt line.
func fencesIn(content string) []readmeFence {
	var fences []readmeFence
	var current []string
	info := ""
	inBlock := false
	for line := range strings.SplitSeq(content, "\n") {
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
		if prompt, body, ok := fenceBody(current); ok {
			fences = append(fences, readmeFence{info: info, prompt: prompt, body: body})
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
		blocks[firstLineOf(fence.body)] = fence.body
	}
	return blocks
}

// fenceBody splits a fenced block into its leading shell prompt line, if any,
// and the remaining body.
func fenceBody(lines []string) (prompt string, body string, ok bool) {
	if len(lines) > 0 && strings.HasPrefix(lines[0], readmeShellPromptLinePrefix) {
		prompt, lines = lines[0], lines[1:]
	}
	if len(lines) == 0 {
		return "", "", false
	}
	return prompt, strings.Join(lines, "\n"), true
}

func firstLineOf(body string) string {
	first, _, _ := strings.Cut(body, "\n")
	return first
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
			name: "it reproduces the README bare description sample",
			samples: []readmeSample{
				{name: "bare description", firstLine: readmeBareDescriptionAnchor, occurrence: 1, format: "--toon", args: []string{"show", "tick-a1b2", "--field", "description"}, tasks: fieldSelectionTasks},
			},
		},
		{
			name: "it reproduces the README bare note position sample",
			samples: []readmeSample{
				{name: "bare note position", firstLine: readmeBareNoteAnchor, occurrence: 1, format: "--toon", args: []string{"show", "tick-a1b2", "--field", "notes.2"}, tasks: fieldSelectionTasks},
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
		if firstLineOf(fence.body) != sample.firstLine {
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

// readmeSampleExemptions maps the prompt line of a prompted README fence that
// no readmeSample renders to the reason it renders none.
var readmeSampleExemptions = map[string]string{
	"$ tick list --stauts open": "documents a flag error, not rendered sample output",
}

type readmeAnchor struct {
	info      string
	firstLine string
}

type readmeClaim struct {
	anchor     readmeAnchor
	occurrence int
}

// readmeCoverageErrors reports every prompted fence that no sample and no
// exemption claims, and every exemption that claims no prompted fence.
func readmeCoverageErrors(fences []readmeFence, samples []readmeSample, exemptions map[string]string) []error {
	claimed := make(map[readmeClaim]bool, len(samples))
	for _, sample := range samples {
		anchor := readmeAnchor{info: sample.info, firstLine: sample.firstLine}
		claimed[readmeClaim{anchor: anchor, occurrence: sample.occurrence}] = true
	}

	var errs []error
	exempted := make(map[string]bool, len(exemptions))
	occurrences := make(map[readmeAnchor]int, len(fences))
	for _, fence := range fences {
		anchor := readmeAnchor{info: fence.info, firstLine: firstLineOf(fence.body)}
		occurrences[anchor]++
		if fence.prompt == "" {
			continue
		}
		if _, ok := exemptions[fence.prompt]; ok {
			exempted[fence.prompt] = true
			continue
		}
		if !claimed[readmeClaim{anchor: anchor, occurrence: occurrences[anchor]}] {
			errs = append(errs, fmt.Errorf("README sample %q is rendered by no readmeSample", fence.prompt))
		}
	}

	for _, prompt := range slices.Sorted(maps.Keys(exemptions)) {
		if !exempted[prompt] {
			errs = append(errs, fmt.Errorf("README sample exemption %q (%s) matches no prompted fence", prompt, exemptions[prompt]))
		}
	}
	return errs
}

func allREADMESamples(t *testing.T) []readmeSample {
	t.Helper()
	var samples []readmeSample
	for _, group := range readmeSampleGroups(t) {
		samples = append(samples, group.samples...)
	}
	return samples
}

func TestREADMEPromptedSamplesAreClaimed(t *testing.T) {
	t.Run("it claims every prompted README sample", func(t *testing.T) {
		for _, err := range readmeCoverageErrors(readmeFences(t), allREADMESamples(t), readmeSampleExemptions) {
			t.Error(err)
		}
	})

	t.Run("it fails when a prompted sample is claimed by nothing", func(t *testing.T) {
		fences := []readmeFence{{prompt: "$ tick list", body: readmeListSampleAnchor}}
		errs := readmeCoverageErrors(fences, nil, nil)
		if len(errs) != 1 {
			t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "$ tick list") {
			t.Errorf("error %q does not name the unclaimed prompt line", errs[0])
		}
	})

	t.Run("it fails when an exemption names no fence", func(t *testing.T) {
		exemptions := map[string]string{"$ tick list --stauts open": "documents a flag error"}
		errs := readmeCoverageErrors(nil, nil, exemptions)
		if len(errs) != 1 {
			t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "$ tick list --stauts open") {
			t.Errorf("error %q does not name the unused exemption", errs[0])
		}
	})
}

// readmeShowSection returns the body of the README's `### show` section.
func readmeShowSection(t *testing.T) string {
	t.Helper()
	return readmeSection(t, "### `show`")
}

// readmeSection returns the body of the README section under heading, up to the
// next heading at the same level.
func readmeSection(t *testing.T, heading string) string {
	t.Helper()
	_, after, ok := strings.Cut(readmeContent(t), "\n"+heading+"\n")
	if !ok {
		t.Fatalf("README has no %q section", heading)
	}
	level, _, _ := strings.Cut(heading, " ")
	section, _, _ := strings.Cut(after, "\n"+level+" ")
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

// readmeGlobalFlagLabels returns the labels listed in the README's Global Flags
// fenced block, each the text before its description column.
func readmeGlobalFlagLabels(t *testing.T) []string {
	t.Helper()
	for _, fence := range fencesIn(readmeSection(t, "## Global Flags")) {
		if fence.info != "" || fence.prompt != "" {
			continue
		}
		var labels []string
		for line := range strings.SplitSeq(fence.body, "\n") {
			label, _, ok := strings.Cut(line, "  ")
			if !ok {
				t.Fatalf("global flag line %q separates no label from its description", line)
			}
			labels = append(labels, label)
		}
		return labels
	}
	t.Fatal("README Global Flags section has no flag listing block")
	return nil
}

func TestREADMEDocumentsEndOfFlagsMarker(t *testing.T) {
	t.Run("it documents the marker in the README global flags block", func(t *testing.T) {
		if labels := readmeGlobalFlagLabels(t); !slices.Contains(labels, endOfFlagsMarker) {
			t.Errorf("README global flags are %v, want one labelled %q", labels, endOfFlagsMarker)
		}
	})

	t.Run("it states that an argument spelling a global flag is text after the marker", func(t *testing.T) {
		section := readmeSection(t, "## Global Flags")
		var prose strings.Builder
		inFence := false
		for line := range strings.SplitSeq(section, "\n") {
			if strings.HasPrefix(line, "```") {
				inFence = !inFence
				continue
			}
			if !inFence {
				prose.WriteString(line + "\n")
			}
		}
		for _, want := range []string{backticked(endOfFlagsMarker), backticked("--json")} {
			if !strings.Contains(prose.String(), want) {
				t.Errorf("README Global Flags prose does not mention %s:\n%s", want, prose.String())
			}
		}
	})

	for _, heading := range []string{"### `create`", "### `note`"} {
		t.Run("it shows a marker invocation in the "+heading+" examples", func(t *testing.T) {
			var marked int
			for _, fence := range fencesIn(readmeSection(t, heading)) {
				if fence.info != "bash" {
					t.Errorf("%s section carries a %q fence, want shell examples only", heading, fence.info)
					continue
				}
				for line := range strings.SplitSeq(fence.body, "\n") {
					if strings.Contains(line, " "+endOfFlagsMarker+" ") {
						marked++
					}
				}
			}
			if marked != 1 {
				t.Errorf("%s section has %d %q invocations, want 1", heading, marked, endOfFlagsMarker)
			}
		})
	}
}
