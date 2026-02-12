package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/doctor"
)

// RunDoctor executes the doctor diagnostic command. It creates a DiagnosticRunner,
// registers the CacheStalenessCheck, runs all checks, formats the output to stdout,
// and returns the appropriate exit code. Doctor is read-only and never modifies data.
func RunDoctor(stdout io.Writer, stderr io.Writer, tickDir string) int {
	runner := doctor.NewDiagnosticRunner()
	runner.Register(&doctor.CacheStalenessCheck{})

	ctx := context.WithValue(context.Background(), doctor.TickDirKey, tickDir)
	report := runner.RunAll(ctx)

	doctor.FormatReport(stdout, report)

	return doctor.ExitCode(report)
}

// handleDoctor implements the doctor subcommand. It discovers the .tick directory
// and delegates to RunDoctor. Unlike other commands, doctor bypasses the format/formatter
// machinery and always outputs human-readable text.
func (a *App) handleDoctor() int {
	dir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: could not determine working directory: %s\n", err)
		return 1
	}

	tickDir, err := DiscoverTickDir(dir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error: Not a tick project (no .tick directory found)\n")
		return 1
	}

	return RunDoctor(a.Stdout, a.Stderr, tickDir)
}
