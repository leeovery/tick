package cli

import "io"

// RunDepTree executes the dep tree command: displays the dependency tree
// for all tasks (full graph mode) or a specific task (focused mode).
func RunDepTree(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error {
	return nil
}
