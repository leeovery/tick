package cli

import (
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/storage"
)

// VerboseLogger writes verbose debug messages to a writer (intended for stderr).
// All messages are prefixed with "verbose: " for grep-ability.
// A nil VerboseLogger is a no-op (safe to call Log on nil receiver).
type VerboseLogger struct {
	w io.Writer
}

// NewVerboseLogger creates a VerboseLogger that writes to the given writer.
func NewVerboseLogger(w io.Writer) *VerboseLogger {
	return &VerboseLogger{w: w}
}

// Log writes a verbose-prefixed message. Safe to call on a nil receiver (no-op).
func (vl *VerboseLogger) Log(msg string) {
	if vl == nil {
		return
	}
	fmt.Fprintf(vl.w, "verbose: %s\n", msg)
}

// storeOpts returns storage.StoreOption(s) that configure verbose logging
// based on the FormatConfig. Returns nil if verbose is not enabled.
func storeOpts(fc FormatConfig) []storage.StoreOption {
	if fc.Logger == nil {
		return nil
	}
	return []storage.StoreOption{
		storage.WithVerbose(fc.Logger.Log),
	}
}
