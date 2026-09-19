package cli

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type showFieldKind int

const (
	showFieldUnknown showFieldKind = iota
	showFieldScalar
	showFieldDescription
	showFieldList
)

// showFields is the registry of field names `tick show --field` recognises,
// spelled as the detail document spells them. Only list sections accept a
// positional suffix.
var showFields = map[string]showFieldKind{
	"id":          showFieldScalar,
	"title":       showFieldScalar,
	"status":      showFieldScalar,
	"priority":    showFieldScalar,
	"type":        showFieldScalar,
	"parent":      showFieldScalar,
	"created":     showFieldScalar,
	"updated":     showFieldScalar,
	"closed":      showFieldScalar,
	"description": showFieldDescription,
	"notes":       showFieldList,
	"tags":        showFieldList,
	"refs":        showFieldList,
	"children":    showFieldList,
	"blocked_by":  showFieldList,
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

// Positions returns the 1-based positions requested for name, or nil when the
// name was taken whole or not requested at all.
func (s *FieldSelection) Positions(name string) []int {
	if s.whole[name] {
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
			if showFields[name] == showFieldUnknown {
				return unknownFieldError(name)
			}
			s.addWhole(name)
			continue
		}
		if showFields[base] != showFieldList {
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
