package cli

import (
	"fmt"
	"io"
)

// VerboseLogger writes verbose debug output to a writer (typically stderr).
// When disabled, all calls are no-ops.
// All output is prefixed with "verbose: " for grep-ability.
type VerboseLogger struct {
	w       io.Writer
	enabled bool
}

// NewVerboseLogger creates a VerboseLogger that writes to w when enabled is true.
// When enabled is false, Log calls are no-ops.
func NewVerboseLogger(w io.Writer, enabled bool) *VerboseLogger {
	return &VerboseLogger{w: w, enabled: enabled}
}

// Log writes a verbose message with the "verbose: " prefix.
// No-op when the logger is disabled.
func (v *VerboseLogger) Log(msg string) {
	if !v.enabled {
		return
	}
	fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
