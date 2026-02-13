package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/leeovery/tick/internal/doctor"
)

// RunDoctor executes the doctor diagnostic command. It creates a DiagnosticRunner,
// registers all 10 checks (CacheStalenessCheck, JsonlSyntaxCheck, IdFormatCheck,
// DuplicateIdCheck, OrphanedParentCheck, OrphanedDependencyCheck, SelfReferentialDepCheck,
// DependencyCycleCheck, ChildBlockedByParentCheck, ParentDoneWithOpenChildrenCheck),
// runs all checks, formats the output to stdout, and returns the appropriate exit code.
// Doctor is read-only and never modifies data.
func RunDoctor(stdout io.Writer, stderr io.Writer, tickDir string) int {
	runner := doctor.NewDiagnosticRunner()
	runner.Register(&doctor.CacheStalenessCheck{})
	runner.Register(&doctor.JsonlSyntaxCheck{})
	runner.Register(&doctor.IdFormatCheck{})
	runner.Register(&doctor.DuplicateIdCheck{})
	runner.Register(&doctor.OrphanedParentCheck{})
	runner.Register(&doctor.OrphanedDependencyCheck{})
	runner.Register(&doctor.SelfReferentialDepCheck{})
	runner.Register(&doctor.DependencyCycleCheck{})
	runner.Register(&doctor.ChildBlockedByParentCheck{})
	runner.Register(&doctor.ParentDoneWithOpenChildrenCheck{})

	ctx := context.Background()

	lines, err := doctor.ScanJSONLines(tickDir)
	if err == nil {
		ctx = context.WithValue(ctx, doctor.JSONLinesKey, lines)
	}

	report := runner.RunAll(ctx, tickDir)

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
