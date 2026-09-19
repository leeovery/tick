package cli

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/leeovery/tick/internal/task"
)

type showFieldKind int

const (
	showFieldUnknown showFieldKind = iota
	showFieldScalar
	showFieldDescription
	showFieldList
)

// showField describes one recognised field name. bare renders the field's
// single value and is nil for the list sections.
// items renders a list section's values and is nil for a section whose items
// are rows rather than values.
type showField struct {
	kind  showFieldKind
	bare  func(TaskDetail) string
	items func(TaskDetail) []string
}

// showFields is the registry of field names `tick show --field` recognises,
// spelled as the detail document spells them. Only list sections accept a
// positional suffix.
var showFields = map[string]showField{
	"id":          {kind: showFieldScalar, bare: func(d TaskDetail) string { return d.Task.ID }},
	"title":       {kind: showFieldScalar, bare: func(d TaskDetail) string { return d.Task.Title }},
	"status":      {kind: showFieldScalar, bare: func(d TaskDetail) string { return string(d.Task.Status) }},
	"priority":    {kind: showFieldScalar, bare: func(d TaskDetail) string { return strconv.Itoa(d.Task.Priority) }},
	"type":        {kind: showFieldScalar, bare: func(d TaskDetail) string { return d.Task.Type }},
	"parent":      {kind: showFieldScalar, bare: func(d TaskDetail) string { return d.Task.Parent }},
	"created":     {kind: showFieldScalar, bare: func(d TaskDetail) string { return task.FormatTimestamp(d.Task.Created) }},
	"updated":     {kind: showFieldScalar, bare: func(d TaskDetail) string { return task.FormatTimestamp(d.Task.Updated) }},
	"closed":      {kind: showFieldScalar, bare: bareClosed},
	"description": {kind: showFieldDescription, bare: func(d TaskDetail) string { return d.Task.Description }},
	"notes":       {kind: showFieldList, items: noteTexts},
	"tags":        {kind: showFieldList, items: func(d TaskDetail) []string { return d.Tags }},
	"refs":        {kind: showFieldList, items: func(d TaskDetail) []string { return d.Refs }},
	"children":    {kind: showFieldList},
	"blocked_by":  {kind: showFieldList},
}

// showSections holds each list section's length and the noun an out-of-range
// error spells it with, in the order the toon document renders them.
var showSections = []struct {
	name   string
	noun   string
	length func(TaskDetail) int
}{
	{"blocked_by", "blocker(s)", func(d TaskDetail) int { return len(d.BlockedBy) }},
	{"children", "child(ren)", func(d TaskDetail) int { return len(d.Children) }},
	{"tags", "tag(s)", func(d TaskDetail) int { return len(d.Tags) }},
	{"refs", "ref(s)", func(d TaskDetail) int { return len(d.Refs) }},
	{"notes", "note(s)", func(d TaskDetail) int { return len(d.Notes) }},
}

// selectedItems narrows a section's items to the requested 1-based positions,
// returning the surviving items alongside the positions they hold in the whole
// section. Nil positions keep every item. Positions are ordered ascending and
// deduplicated, and one outside the section's range is skipped.
func selectedItems[T any](items []T, positions []int) ([]T, []int) {
	if positions == nil {
		all := make([]int, len(items))
		for i := range items {
			all[i] = i + 1
		}
		return items, all
	}

	ordered := slices.Sorted(slices.Values(positions))
	ordered = slices.Compact(ordered)

	kept := make([]T, 0, len(ordered))
	keptPositions := make([]int, 0, len(ordered))
	for _, pos := range ordered {
		if pos < 1 || pos > len(items) {
			continue
		}
		kept = append(kept, items[pos-1])
		keptPositions = append(keptPositions, pos)
	}
	return kept, keptPositions
}

func noteTexts(d TaskDetail) []string {
	texts := make([]string, len(d.Notes))
	for i, n := range d.Notes {
		texts[i] = n.Text
	}
	return texts
}

func bareClosed(d TaskDetail) string {
	if d.Task.Closed == nil {
		return ""
	}
	return task.FormatTimestamp(*d.Task.Closed)
}

// bareFieldValue returns the value of a selection that names exactly one field
// resolving to a single value, with ok false for every other selection. A field
// the task does not carry yields the empty string with ok true.
func bareFieldValue(detail TaskDetail, sel *FieldSelection) (string, bool) {
	if sel == nil {
		return "", false
	}
	name, ok := sel.Only()
	if !ok {
		return "", false
	}
	if value, ok := barePositionValue(detail, name, sel.Positions(name)); ok {
		return value, true
	}
	field := showFields[name]
	if field.bare == nil {
		return "", false
	}
	return field.bare(detail), true
}

// barePositionValue returns the value a single position names within a list
// section that holds values, with ok false for every other selection.
func barePositionValue(detail TaskDetail, name string, positions []int) (string, bool) {
	items := showFields[name].items
	if items == nil || len(positions) != 1 {
		return "", false
	}
	values, _ := selectedItems(items(detail), positions)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

// FieldSelection is a validated set of field names requested via --field,
// each optionally narrowed to 1-based positions within a list section.
type FieldSelection struct {
	names     []string
	whole     map[string]bool
	positions map[string][]int
}

func newFieldSelection() *FieldSelection {
	return &FieldSelection{
		whole:     make(map[string]bool),
		positions: make(map[string][]int),
	}
}

// Selected reports whether the name was requested.
func (s *FieldSelection) Selected(name string) bool {
	return s.whole[name] || len(s.positions[name]) > 0
}

// includes reports whether name belongs in the document. A nil selection is the
// whole document and includes every name.
func (s *FieldSelection) includes(name string) bool {
	return s == nil || s.Selected(name)
}

// Positions returns the 1-based positions requested for name, or nil when the
// name was taken whole or not requested at all. A nil selection requests no
// position, the whole section being the whole document's.
func (s *FieldSelection) Positions(name string) []int {
	if s == nil || s.whole[name] {
		return nil
	}
	return s.positions[name]
}

// Len returns the number of distinct names requested.
func (s *FieldSelection) Len() int {
	return len(s.names)
}

// Only returns the sole requested name, and false when several were requested.
func (s *FieldSelection) Only() (string, bool) {
	if len(s.names) != 1 {
		return "", false
	}
	return s.names[0], true
}

func (s *FieldSelection) record(name string) {
	if !s.whole[name] && len(s.positions[name]) == 0 {
		s.names = append(s.names, name)
	}
}

func (s *FieldSelection) addWhole(name string) {
	s.record(name)
	s.whole[name] = true
	delete(s.positions, name)
}

func (s *FieldSelection) addPosition(name string, pos int) {
	if s.whole[name] || slices.Contains(s.positions[name], pos) {
		return
	}
	s.record(name)
	s.positions[name] = append(s.positions[name], pos)
}

// addValue parses one --field value: a comma-separated list of names, each
// optionally suffixed with a position.
func (s *FieldSelection) addValue(value string) error {
	for part := range strings.SplitSeq(value, ",") {
		name := strings.TrimSpace(part)
		base, suffix, hasSuffix := strings.Cut(name, ".")
		if !hasSuffix {
			if showFields[name].kind == showFieldUnknown {
				return unknownFieldError(name)
			}
			s.addWhole(name)
			continue
		}
		if showFields[base].kind != showFieldList {
			return unknownFieldError(name)
		}
		pos, err := strconv.Atoi(suffix)
		if err != nil {
			return unknownFieldError(name)
		}
		s.addPosition(base, pos)
	}
	return nil
}

func unknownFieldError(name string) error {
	return fmt.Errorf("unknown field %q for \"show\". Run 'tick help show' for usage.", name)
}

// parseShowArgs separates the task ID from the field selection in show's
// arguments. The selection is nil when neither --field nor --fields appeared.
func parseShowArgs(args []string) (string, *FieldSelection, error) {
	var id string
	var selection *FieldSelection

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--field" || arg == "--fields" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--field requires a value")
			}
			i++
			if selection == nil {
				selection = newFieldSelection()
			}
			if err := selection.addValue(args[i]); err != nil {
				return "", nil, err
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if id == "" {
			id = arg
		}
	}

	return id, selection, nil
}

// ValidatePositions returns an error for the first selected position falling
// outside its section in detail, sections in document order and positions
// ascending. A section named whole carries no position and cannot fail.
func (s *FieldSelection) ValidatePositions(detail TaskDetail) error {
	if s == nil {
		return nil
	}
	for _, section := range showSections {
		length := section.length(detail)
		for _, pos := range slices.Sorted(slices.Values(s.Positions(section.name))) {
			if pos < 1 || pos > length {
				return fmt.Errorf("%s.%d out of range: task has %d %s", section.name, pos, length, section.noun)
			}
		}
	}
	return nil
}
