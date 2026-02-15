package migrate

import (
	"fmt"
	"sort"
	"strings"
)

// UnknownProviderError is returned when a provider name is not recognized.
// It includes the unknown name and a sorted list of available provider names,
// producing a multi-line error message matching the spec format.
type UnknownProviderError struct {
	Name      string
	Available []string
}

// Error produces the spec-mandated multi-line format:
//
//	Unknown provider "<name>"
//
//	Available providers:
//	  - <provider1>
//	  - <provider2>
func (e *UnknownProviderError) Error() string {
	sorted := make([]string, len(e.Available))
	copy(sorted, e.Available)
	sort.Strings(sorted)

	var b strings.Builder
	fmt.Fprintf(&b, "Unknown provider %q", e.Name)
	b.WriteString("\n\nAvailable providers:")
	for _, p := range sorted {
		fmt.Fprintf(&b, "\n  - %s", p)
	}
	return b.String()
}
