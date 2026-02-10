package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	t.Run("it creates .tick/ directory in current working directory", func(t *testing.T) {
		dir := t.TempDir()
		var stdout bytes.Buffer

		err := RunInit(dir, FormatConfig{}, &PrettyFormatter{}, &stdout)
		if err != nil {
			t.Fatalf("RunInit returned error: %v", err)
		}

		tickDir := filepath.Join(dir, ".tick")
		info, err := os.Stat(tickDir)
		if err != nil {
			t.Fatalf(".tick/ directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error(".tick/ is not a directory")
		}
	})

	t.Run("it creates empty tasks.jsonl inside .tick/", func(t *testing.T) {
		dir := t.TempDir()
		var stdout bytes.Buffer

		err := RunInit(dir, FormatConfig{}, &PrettyFormatter{}, &stdout)
		if err != nil {
			t.Fatalf("RunInit returned error: %v", err)
		}

		jsonlPath := filepath.Join(dir, ".tick", "tasks.jsonl")
		info, err := os.Stat(jsonlPath)
		if err != nil {
			t.Fatalf("tasks.jsonl not created: %v", err)
		}
		if info.IsDir() {
			t.Error("tasks.jsonl should be a file, not a directory")
		}
		if info.Size() != 0 {
			t.Errorf("tasks.jsonl should be empty (0 bytes), got %d bytes", info.Size())
		}
	})

	t.Run("it does not create cache.db at init time", func(t *testing.T) {
		dir := t.TempDir()
		var stdout bytes.Buffer

		err := RunInit(dir, FormatConfig{}, &PrettyFormatter{}, &stdout)
		if err != nil {
			t.Fatalf("RunInit returned error: %v", err)
		}

		cachePath := filepath.Join(dir, ".tick", "cache.db")
		_, err = os.Stat(cachePath)
		if err == nil {
			t.Error("cache.db should not exist after init")
		}
		if !os.IsNotExist(err) {
			t.Errorf("unexpected error checking cache.db: %v", err)
		}
	})

	t.Run("it prints confirmation with absolute path on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout bytes.Buffer

		err := RunInit(dir, FormatConfig{}, &PrettyFormatter{}, &stdout)
		if err != nil {
			t.Fatalf("RunInit returned error: %v", err)
		}

		absDir, _ := filepath.Abs(dir)
		expected := "Initialized tick in " + absDir + "/.tick/\n"
		if stdout.String() != expected {
			t.Errorf("stdout = %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("it prints nothing with --quiet flag on success", func(t *testing.T) {
		dir := t.TempDir()
		var stdout bytes.Buffer

		err := RunInit(dir, FormatConfig{Quiet: true}, &PrettyFormatter{}, &stdout)
		if err != nil {
			t.Fatalf("RunInit returned error: %v", err)
		}

		if stdout.String() != "" {
			t.Errorf("stdout should be empty with quiet=true, got %q", stdout.String())
		}
	})

	t.Run("it errors when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick/ for test setup: %v", err)
		}

		var stdout bytes.Buffer
		err := RunInit(dir, FormatConfig{}, &PrettyFormatter{}, &stdout)
		if err == nil {
			t.Fatal("expected error when .tick/ already exists, got nil")
		}

		absDir, _ := filepath.Abs(dir)
		expected := "tick already initialized in " + absDir
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("it returns exit code 1 when .tick/ already exists", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick/ for test setup: %v", err)
		}

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "init"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("it writes error messages to stderr, not stdout", func(t *testing.T) {
		dir := t.TempDir()
		tickDir := filepath.Join(dir, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick/ for test setup: %v", err)
		}

		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		app.Run([]string{"tick", "init"})

		if stdout.String() != "" {
			t.Errorf("stdout should be empty on error, got %q", stdout.String())
		}

		absDir, _ := filepath.Abs(dir)
		expectedErr := "Error: tick already initialized in " + absDir + "\n"
		if stderr.String() != expectedErr {
			t.Errorf("stderr = %q, want %q", stderr.String(), expectedErr)
		}
	})
}

func TestDispatch(t *testing.T) {
	t.Run("it routes unknown subcommands to error", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "foobar"})
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		expected := "Error: Unknown command 'foobar'. Run 'tick help' for usage.\n"
		if stderr.String() != expected {
			t.Errorf("stderr = %q, want %q", stderr.String(), expected)
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty on error, got %q", stdout.String())
		}
	})

	t.Run("it prints usage with exit code 0 when no subcommand given", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.String() == "" {
			t.Error("stdout should contain usage information")
		}
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("it passes --quiet flag to init command via dispatch", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "--quiet", "init"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty with --quiet, got %q", stdout.String())
		}
	})

	t.Run("it passes -q short flag to init command via dispatch", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "-q", "init"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty with -q, got %q", stdout.String())
		}
	})

	t.Run("it accepts global flags after subcommand", func(t *testing.T) {
		dir := t.TempDir()
		var stdout, stderr bytes.Buffer
		app := &App{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  func() (string, error) { return dir, nil },
		}
		exitCode := app.Run([]string{"tick", "init", "--quiet"})
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.String() != "" {
			t.Errorf("stdout should be empty with --quiet after subcommand, got %q", stdout.String())
		}
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})
}

func TestParseArgs(t *testing.T) {
	t.Run("it parses all global flags", func(t *testing.T) {
		tests := []struct {
			name  string
			args  []string
			check func(t *testing.T, f globalFlags)
		}{
			{
				"--quiet",
				[]string{"--quiet", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.quiet {
						t.Error("quiet should be true")
					}
				},
			},
			{
				"-q short form",
				[]string{"-q", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.quiet {
						t.Error("quiet should be true")
					}
				},
			},
			{
				"--verbose",
				[]string{"--verbose", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.verbose {
						t.Error("verbose should be true")
					}
				},
			},
			{
				"-v short form",
				[]string{"-v", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.verbose {
						t.Error("verbose should be true")
					}
				},
			},
			{
				"--toon",
				[]string{"--toon", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.toon {
						t.Error("toon should be true")
					}
				},
			},
			{
				"--pretty",
				[]string{"--pretty", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.pretty {
						t.Error("pretty should be true")
					}
				},
			},
			{
				"--json",
				[]string{"--json", "init"},
				func(t *testing.T, f globalFlags) {
					t.Helper()
					if !f.json {
						t.Error("json should be true")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				flags, subcmd, _ := parseArgs(tt.args)
				tt.check(t, flags)
				if subcmd != "init" {
					t.Errorf("subcmd = %q, want %q", subcmd, "init")
				}
			})
		}
	})
}

func TestParseArgsGlobalFlagsAfterSubcommand(t *testing.T) {
	t.Run("it extracts global flags after the subcommand", func(t *testing.T) {
		flags, subcmd, subArgs := parseArgs([]string{"init", "--quiet"})
		if subcmd != "init" {
			t.Errorf("subcmd = %q, want %q", subcmd, "init")
		}
		if !flags.quiet {
			t.Error("quiet should be true when --quiet appears after subcommand")
		}
		if len(subArgs) != 0 {
			t.Errorf("subArgs = %v, want empty slice", subArgs)
		}
	})

	t.Run("it extracts global flags from both before and after the subcommand", func(t *testing.T) {
		flags, subcmd, subArgs := parseArgs([]string{"--verbose", "init", "--quiet"})
		if subcmd != "init" {
			t.Errorf("subcmd = %q, want %q", subcmd, "init")
		}
		if !flags.quiet {
			t.Error("quiet should be true")
		}
		if !flags.verbose {
			t.Error("verbose should be true")
		}
		if len(subArgs) != 0 {
			t.Errorf("subArgs = %v, want empty slice", subArgs)
		}
	})

	t.Run("it keeps non-global args in subArgs", func(t *testing.T) {
		flags, subcmd, subArgs := parseArgs([]string{"init", "--quiet", "somefile"})
		if subcmd != "init" {
			t.Errorf("subcmd = %q, want %q", subcmd, "init")
		}
		if !flags.quiet {
			t.Error("quiet should be true")
		}
		if len(subArgs) != 1 || subArgs[0] != "somefile" {
			t.Errorf("subArgs = %v, want [somefile]", subArgs)
		}
		_ = flags
	})
}

func TestDiscoverTickDir(t *testing.T) {
	t.Run("it discovers .tick/ directory by walking up from cwd", func(t *testing.T) {
		root := t.TempDir()
		tickDir := filepath.Join(root, ".tick")
		if err := os.Mkdir(tickDir, 0755); err != nil {
			t.Fatalf("failed to create .tick/: %v", err)
		}

		// Create a nested subdirectory
		nested := filepath.Join(root, "sub", "deep")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}

		found, err := DiscoverTickDir(nested)
		if err != nil {
			t.Fatalf("DiscoverTickDir returned error: %v", err)
		}

		absTickDir, _ := filepath.Abs(tickDir)
		if found != absTickDir {
			t.Errorf("found = %q, want %q", found, absTickDir)
		}
	})

	t.Run("it errors when no .tick/ directory found (not a tick project)", func(t *testing.T) {
		dir := t.TempDir()
		_, err := DiscoverTickDir(dir)
		if err == nil {
			t.Fatal("expected error when no .tick/ found, got nil")
		}
		expected := "not a tick project (no .tick directory found)"
		if err.Error() != expected {
			t.Errorf("error = %q, want %q", err.Error(), expected)
		}
	})
}

func TestTTYDetection(t *testing.T) {
	t.Run("it detects TTY vs non-TTY on stdout", func(t *testing.T) {
		// A pipe (created by os.Pipe) is definitely not a TTY.
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

	t.Run("it defaults to Toon when not TTY", func(t *testing.T) {
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

	t.Run("it overrides with --toon flag", func(t *testing.T) {
		format, err := ResolveFormat(globalFlags{toon: true}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatToon {
			t.Errorf("format = %d, want FormatToon (%d)", format, FormatToon)
		}
	})

	t.Run("it overrides with --pretty flag", func(t *testing.T) {
		format, err := ResolveFormat(globalFlags{pretty: true}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatPretty {
			t.Errorf("format = %d, want FormatPretty (%d)", format, FormatPretty)
		}
	})

	t.Run("it overrides with --json flag", func(t *testing.T) {
		format, err := ResolveFormat(globalFlags{json: true}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if format != FormatJSON {
			t.Errorf("format = %d, want FormatJSON (%d)", format, FormatJSON)
		}
	})
}
