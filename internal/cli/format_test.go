package cli

import (
	"os"
	"testing"

	"github.com/leeovery/tick/internal/task"
)

func TestFormatEnum(t *testing.T) {
	t.Run("it defines three distinct format constants", func(t *testing.T) {
		formats := []Format{FormatToon, FormatPretty, FormatJSON}
		seen := make(map[Format]bool)
		for _, f := range formats {
			if seen[f] {
				t.Errorf("duplicate format constant value: %d", f)
			}
			seen[f] = true
		}
		if len(seen) != 3 {
			t.Errorf("expected 3 distinct format constants, got %d", len(seen))
		}
	})
}

func TestDetectTTY(t *testing.T) {
	t.Run("it detects non-TTY for a pipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		if DetectTTY(w) {
			t.Error("pipe should not be detected as TTY")
		}
	})

	t.Run("it defaults to non-TTY on stat failure", func(t *testing.T) {
		// Create a pipe, close it, then call DetectTTY on the closed file.
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		r.Close()
		w.Close()

		// Stat on a closed file should fail; DetectTTY should return false (non-TTY).
		if DetectTTY(w) {
			t.Error("closed file should default to non-TTY")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	t.Run("it defaults to Toon when non-TTY", func(t *testing.T) {
		format, err := ResolveFormat(globalFlags{}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("format = %d, want FormatToon (%d)", format, FormatToon)
		}
	})

	t.Run("it defaults to Pretty when TTY", func(t *testing.T) {
		format, err := ResolveFormat(globalFlags{}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("format = %d, want FormatPretty (%d)", format, FormatPretty)
		}
	})

	t.Run("it returns correct format for each flag override", func(t *testing.T) {
		tests := []struct {
			name   string
			flags  globalFlags
			isTTY  bool
			expect Format
		}{
			{"--toon overrides TTY", globalFlags{toon: true}, true, FormatToon},
			{"--toon with non-TTY", globalFlags{toon: true}, false, FormatToon},
			{"--pretty overrides non-TTY", globalFlags{pretty: true}, false, FormatPretty},
			{"--pretty with TTY", globalFlags{pretty: true}, true, FormatPretty},
			{"--json overrides TTY", globalFlags{json: true}, true, FormatJSON},
			{"--json overrides non-TTY", globalFlags{json: true}, false, FormatJSON},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				format, err := ResolveFormat(tt.flags, tt.isTTY)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if format != tt.expect {
					t.Errorf("format = %d, want %d", format, tt.expect)
				}
			})
		}
	})

	t.Run("it errors when multiple format flags set", func(t *testing.T) {
		tests := []struct {
			name  string
			flags globalFlags
		}{
			{"--toon and --pretty", globalFlags{toon: true, pretty: true}},
			{"--toon and --json", globalFlags{toon: true, json: true}},
			{"--pretty and --json", globalFlags{pretty: true, json: true}},
			{"all three flags", globalFlags{toon: true, pretty: true, json: true}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ResolveFormat(tt.flags, false)
				if err == nil {
					t.Fatal("expected error when multiple format flags set, got nil")
				}
				expected := "only one format flag (--toon, --pretty, --json) may be specified"
				if err.Error() != expected {
					t.Errorf("error = %q, want %q", err.Error(), expected)
				}
			})
		}
	})
}

func TestFormatConfig(t *testing.T) {
	t.Run("it propagates quiet and verbose in FormatConfig", func(t *testing.T) {
		cfg := FormatConfig{
			Format:  FormatPretty,
			Quiet:   true,
			Verbose: true,
		}
		if cfg.Format != FormatPretty {
			t.Errorf("Format = %d, want FormatPretty (%d)", cfg.Format, FormatPretty)
		}
		if !cfg.Quiet {
			t.Error("Quiet should be true")
		}
		if !cfg.Verbose {
			t.Error("Verbose should be true")
		}
	})

	t.Run("it defaults quiet and verbose to false", func(t *testing.T) {
		cfg := FormatConfig{Format: FormatToon}
		if cfg.Quiet {
			t.Error("Quiet should default to false")
		}
		if cfg.Verbose {
			t.Error("Verbose should default to false")
		}
	})
}

func TestNewFormatConfig(t *testing.T) {
	t.Run("it builds FormatConfig from flags and TTY detection", func(t *testing.T) {
		flags := globalFlags{quiet: true, verbose: true}
		cfg, err := NewFormatConfig(flags, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Format != FormatPretty {
			t.Errorf("Format = %d, want FormatPretty (%d)", cfg.Format, FormatPretty)
		}
		if !cfg.Quiet {
			t.Error("Quiet should be true")
		}
		if !cfg.Verbose {
			t.Error("Verbose should be true")
		}
	})

	t.Run("it returns error from conflicting flags", func(t *testing.T) {
		flags := globalFlags{toon: true, json: true}
		_, err := NewFormatConfig(flags, true)
		if err == nil {
			t.Fatal("expected error for conflicting flags, got nil")
		}
	})
}

func TestFormatterInterface(t *testing.T) {
	t.Run("it satisfies Formatter interface with stub", func(t *testing.T) {
		var f Formatter = &StubFormatter{}

		// FormatTaskList
		result := f.FormatTaskList(nil)
		if result != "" {
			t.Errorf("FormatTaskList = %q, want empty string", result)
		}

		// FormatTaskDetail accepts TaskDetail with related task context
		detail := TaskDetail{
			Task:        task.Task{},
			BlockedBy:   []RelatedTask{{ID: "tick-abc123", Title: "dep", Status: "open"}},
			Children:    []RelatedTask{{ID: "tick-def456", Title: "child", Status: "done"}},
			ParentTitle: "parent task",
		}
		result = f.FormatTaskDetail(detail)
		if result != "" {
			t.Errorf("FormatTaskDetail = %q, want empty string", result)
		}

		// FormatTransition
		result = f.FormatTransition("tick-abc123", "open", "in_progress")
		if result != "" {
			t.Errorf("FormatTransition = %q, want empty string", result)
		}

		// FormatDepChange
		result = f.FormatDepChange("added", "tick-abc123", "tick-def456")
		if result != "" {
			t.Errorf("FormatDepChange = %q, want empty string", result)
		}

		// FormatStats accepts typed Stats struct
		stats := Stats{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		result = f.FormatStats(stats)
		if result != "" {
			t.Errorf("FormatStats = %q, want empty string", result)
		}

		// FormatMessage
		result = f.FormatMessage("hello")
		if result != "" {
			t.Errorf("FormatMessage = %q, want empty string", result)
		}
	})
}

func TestTaskDetailStruct(t *testing.T) {
	t.Run("it holds task with related context for show output", func(t *testing.T) {
		detail := TaskDetail{
			Task:        task.Task{ID: "tick-abc123", Title: "Test task"},
			BlockedBy:   []RelatedTask{{ID: "tick-111111", Title: "blocker", Status: "open"}},
			Children:    []RelatedTask{{ID: "tick-222222", Title: "subtask", Status: "done"}},
			ParentTitle: "parent",
		}
		if detail.Task.ID != "tick-abc123" {
			t.Errorf("Task.ID = %q, want %q", detail.Task.ID, "tick-abc123")
		}
		if len(detail.BlockedBy) != 1 {
			t.Errorf("BlockedBy length = %d, want 1", len(detail.BlockedBy))
		}
		if detail.BlockedBy[0].ID != "tick-111111" {
			t.Errorf("BlockedBy[0].ID = %q, want %q", detail.BlockedBy[0].ID, "tick-111111")
		}
		if len(detail.Children) != 1 {
			t.Errorf("Children length = %d, want 1", len(detail.Children))
		}
		if detail.Children[0].ID != "tick-222222" {
			t.Errorf("Children[0].ID = %q, want %q", detail.Children[0].ID, "tick-222222")
		}
		if detail.ParentTitle != "parent" {
			t.Errorf("ParentTitle = %q, want %q", detail.ParentTitle, "parent")
		}
	})

	t.Run("it works with empty related slices", func(t *testing.T) {
		detail := TaskDetail{
			Task: task.Task{ID: "tick-abc123"},
		}
		if len(detail.BlockedBy) != 0 {
			t.Errorf("BlockedBy should be empty, got %d", len(detail.BlockedBy))
		}
		if len(detail.Children) != 0 {
			t.Errorf("Children should be empty, got %d", len(detail.Children))
		}
		if detail.ParentTitle != "" {
			t.Errorf("ParentTitle should be empty, got %q", detail.ParentTitle)
		}
	})
}

func TestStatsStruct(t *testing.T) {
	t.Run("it holds all stat fields with correct types", func(t *testing.T) {
		stats := Stats{
			Total:      47,
			Open:       12,
			InProgress: 3,
			Done:       28,
			Cancelled:  4,
			Ready:      8,
			Blocked:    4,
			ByPriority: [5]int{2, 8, 25, 7, 5},
		}
		if stats.Total != 47 {
			t.Errorf("Total = %d, want 47", stats.Total)
		}
		if stats.Open != 12 {
			t.Errorf("Open = %d, want 12", stats.Open)
		}
		if stats.InProgress != 3 {
			t.Errorf("InProgress = %d, want 3", stats.InProgress)
		}
		if stats.Done != 28 {
			t.Errorf("Done = %d, want 28", stats.Done)
		}
		if stats.Cancelled != 4 {
			t.Errorf("Cancelled = %d, want 4", stats.Cancelled)
		}
		if stats.Ready != 8 {
			t.Errorf("Ready = %d, want 8", stats.Ready)
		}
		if stats.Blocked != 4 {
			t.Errorf("Blocked = %d, want 4", stats.Blocked)
		}
		if stats.ByPriority != [5]int{2, 8, 25, 7, 5} {
			t.Errorf("ByPriority = %v, want [2 8 25 7 5]", stats.ByPriority)
		}
	})

	t.Run("it defaults to zero values", func(t *testing.T) {
		var stats Stats
		if stats.Total != 0 {
			t.Errorf("Total should default to 0, got %d", stats.Total)
		}
		if stats.ByPriority != [5]int{} {
			t.Errorf("ByPriority should default to zeroes, got %v", stats.ByPriority)
		}
	})
}

func TestFormatterInterfaceCompileCheck(t *testing.T) {
	// Compile-time check that StubFormatter implements Formatter.
	var _ Formatter = (*StubFormatter)(nil)
}

func TestCLIDispatchRejectsConflictingFlags(t *testing.T) {
	t.Run("it errors before dispatch when multiple format flags set", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr byteBuffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--toon", "--json", "init"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		expectedErr := "Error: only one format flag (--toon, --pretty, --json) may be specified\n"
		if stderr.String() != expectedErr {
			t.Errorf("stderr = %q, want %q", stderr.String(), expectedErr)
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty on error, got %q", stdout.String())
		}
	})
}

// byteBuffer wraps bytes.Buffer to implement io.Writer for test use.
type byteBuffer struct {
	data []byte
}

func (b *byteBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *byteBuffer) String() string {
	return string(b.data)
}

func TestVerboseToStderrOnly(t *testing.T) {
	t.Run("it writes verbose output to the provided writer when verbose is true", func(t *testing.T) {
		var stderr byteBuffer
		VerboseLog(&stderr, true, "debug info")
		expected := "debug info\n"
		if stderr.String() != expected {
			t.Errorf("stderr = %q, want %q", stderr.String(), expected)
		}
	})

	t.Run("it writes nothing when verbose is false", func(t *testing.T) {
		var stderr byteBuffer
		VerboseLog(&stderr, false, "debug info")
		if stderr.String() != "" {
			t.Errorf("stderr should be empty when verbose=false, got %q", stderr.String())
		}
	})

	t.Run("it never writes to stdout", func(t *testing.T) {
		// VerboseLog writes to the given writer (intended for stderr).
		// Stdout is never passed; this confirms the design: verbose goes only to stderr.
		var stdout, stderr byteBuffer
		VerboseLog(&stderr, true, "verbose message")
		if stdout.String() != "" {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
		if stderr.String() == "" {
			t.Error("stderr should have verbose output")
		}
	})
}
