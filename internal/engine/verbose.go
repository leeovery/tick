package engine

import (
	"fmt"
	"io"
)

// VerboseLogger writes verbose debug output to the given writer when enabled.
// When disabled, all calls are no-ops. All output lines are prefixed with
// "verbose: " for grep-ability.
type VerboseLogger struct {
	w       io.Writer
	enabled bool
}

// NewVerboseLogger creates a VerboseLogger. When enabled is false, all Log and
// Logf calls are no-ops regardless of the writer value.
func NewVerboseLogger(w io.Writer, enabled bool) *VerboseLogger {
	return &VerboseLogger{w: w, enabled: enabled}
}

// Log writes a verbose message with the "verbose: " prefix and trailing newline.
func (v *VerboseLogger) Log(msg string) {
	if !v.enabled {
		return
	}
	fmt.Fprintf(v.w, "verbose: %s\n", msg)
}

// Logf writes a formatted verbose message with the "verbose: " prefix and
// trailing newline.
func (v *VerboseLogger) Logf(format string, args ...interface{}) {
	if !v.enabled {
		return
	}
	fmt.Fprintf(v.w, "verbose: "+format+"\n", args...)
}
