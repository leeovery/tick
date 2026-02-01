package cli

import (
	"fmt"
	"io"
)

// VerboseLogger writes debug messages to a writer when enabled.
// All messages are prefixed with "verbose: " for grep-ability.
// When disabled, Log is a no-op.
type VerboseLogger struct {
	w       io.Writer
	enabled bool
}

// NewVerboseLogger creates a verbose logger that writes to w when enabled.
func NewVerboseLogger(w io.Writer, enabled bool) *VerboseLogger {
	return &VerboseLogger{w: w, enabled: enabled}
}

// Log writes a verbose message if enabled. No-op otherwise.
func (v *VerboseLogger) Log(format string, args ...any) {
	if !v.enabled {
		return
	}
	fmt.Fprintf(v.w, "verbose: "+format+"\n", args...)
}
