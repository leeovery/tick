// Package cli provides the command-line interface for Tick.
package cli

import (
	"fmt"
	"io"
)

// VerboseLogger writes verbose: prefixed messages to stderr when enabled.
// When disabled, all methods are no-ops.
type VerboseLogger struct {
	w       io.Writer
	enabled bool
}

// NewVerboseLogger creates a VerboseLogger that writes to w when enabled is true.
func NewVerboseLogger(w io.Writer, enabled bool) *VerboseLogger {
	return &VerboseLogger{w: w, enabled: enabled}
}

// Log writes a verbose: prefixed line to stderr if verbose is enabled.
func (v *VerboseLogger) Log(format string, args ...interface{}) {
	if !v.enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(v.w, "verbose: %s\n", msg)
}
