package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestFormatIntegration(t *testing.T) {
	now := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	t.Run("it formats create as full task detail in each format", func(t *testing.T) {
		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					// Toon format: should contain task{...}: section
					if !strings.Contains(stdout, "task{") {
						t.Errorf("toon format should contain 'task{', got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					// Pretty format: key-value with "ID:" prefix
					if !strings.Contains(stdout, "ID:") {
						t.Errorf("pretty format should contain 'ID:', got %q", stdout)
					}
					if !strings.Contains(stdout, "Title:") {
						t.Errorf("pretty format should contain 'Title:', got %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					// JSON format: should be valid JSON with "id" key
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
						t.Errorf("json format should be valid JSON, got error: %v, output: %q", err, stdout)
						return
					}
					if _, ok := obj["id"]; !ok {
						t.Errorf("json format should contain 'id' key, got %v", obj)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProject(t)
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "create", "Test task"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it formats transitions in each format", func(t *testing.T) {
		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "tick-aaa111: open -> in_progress") {
						t.Errorf("toon transition should be 'id: old -> new', got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "tick-aaa111: open -> in_progress") {
						t.Errorf("pretty transition should be 'id: old -> new', got %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
						t.Errorf("json transition should be valid JSON, got error: %v, output: %q", err, stdout)
						return
					}
					if obj["id"] != "tick-aaa111" {
						t.Errorf("json transition id = %v, want tick-aaa111", obj["id"])
					}
					if obj["from"] != "open" {
						t.Errorf("json transition from = %v, want open", obj["from"])
					}
					if obj["to"] != "in_progress" {
						t.Errorf("json transition to = %v, want in_progress", obj["to"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				openTask := task.Task{
					ID: "tick-aaa111", Title: "Open task", Status: task.StatusOpen,
					Priority: 2, Created: now, Updated: now,
				}
				dir, _ := setupTickProjectWithTasks(t, []task.Task{openTask})
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "start", "tick-aaa111"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it formats dep confirmations in each format", func(t *testing.T) {
		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "Dependency added: tick-aaa111 blocked by tick-bbb222") {
						t.Errorf("toon dep output unexpected: %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "Dependency added: tick-aaa111 blocked by tick-bbb222") {
						t.Errorf("pretty dep output unexpected: %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
						t.Errorf("json dep should be valid JSON, got error: %v, output: %q", err, stdout)
						return
					}
					if obj["action"] != "added" {
						t.Errorf("json dep action = %v, want added", obj["action"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				taskA := task.Task{
					ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
					Priority: 2, Created: now, Updated: now,
				}
				taskB := task.Task{
					ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
					Priority: 2, Created: now, Updated: now,
				}
				dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "dep", "add", "tick-aaa111", "tick-bbb222"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it formats list in each format", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "First task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Second task", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
		}

		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "tasks[") {
						t.Errorf("toon list should contain 'tasks[', got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "STATUS") {
						t.Errorf("pretty list should contain table headers, got %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					var arr []map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &arr); err != nil {
						t.Errorf("json list should be valid JSON array, got error: %v, output: %q", err, stdout)
						return
					}
					if len(arr) != 2 {
						t.Errorf("json list should have 2 items, got %d", len(arr))
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "list"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it formats show in each format", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Show task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}

		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "task{") {
						t.Errorf("toon show should contain 'task{', got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "ID:") {
						t.Errorf("pretty show should contain 'ID:', got %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
						t.Errorf("json show should be valid JSON, got error: %v, output: %q", err, stdout)
						return
					}
					if obj["id"] != "tick-aaa111" {
						t.Errorf("json show id = %v, want tick-aaa111", obj["id"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProjectWithTasks(t, tasks)
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "show", "tick-aaa111"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it formats init in each format", func(t *testing.T) {
		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "Initialized tick") {
						t.Errorf("toon init should contain message, got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "Initialized tick") {
						t.Errorf("pretty init should contain message, got %q", stdout)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
						t.Errorf("json init should be valid JSON, got error: %v, output: %q", err, stdout)
						return
					}
					msg, ok := obj["message"].(string)
					if !ok || !strings.Contains(msg, "Initialized tick") {
						t.Errorf("json init message = %v, want to contain 'Initialized tick'", obj["message"])
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "init"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it applies --quiet override for each command type", func(t *testing.T) {
		t.Run("create outputs ID only when quiet", func(t *testing.T) {
			dir, tickDir := setupTickProject(t)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "create", "Quiet task"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			tasks := readPersistedTasks(t, tickDir)
			expected := tasks[0].ID + "\n"
			if stdoutBuf.String() != expected {
				t.Errorf("stdout = %q, want %q", stdoutBuf.String(), expected)
			}
		})

		t.Run("transition outputs nothing when quiet", func(t *testing.T) {
			openTask := task.Task{
				ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen,
				Priority: 2, Created: now, Updated: now,
			}
			dir, _ := setupTickProjectWithTasks(t, []task.Task{openTask})
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "start", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			if stdoutBuf.String() != "" {
				t.Errorf("quiet transition should produce no output, got %q", stdoutBuf.String())
			}
		})

		t.Run("dep add outputs nothing when quiet", func(t *testing.T) {
			taskA := task.Task{
				ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen,
				Priority: 2, Created: now, Updated: now,
			}
			taskB := task.Task{
				ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen,
				Priority: 2, Created: now, Updated: now,
			}
			dir, _ := setupTickProjectWithTasks(t, []task.Task{taskA, taskB})
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "dep", "add", "tick-aaa111", "tick-bbb222"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			if stdoutBuf.String() != "" {
				t.Errorf("quiet dep add should produce no output, got %q", stdoutBuf.String())
			}
		})

		t.Run("list outputs IDs only when quiet", func(t *testing.T) {
			tasks := []task.Task{
				{ID: "tick-aaa111", Title: "First", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
				{ID: "tick-bbb222", Title: "Second", Status: task.StatusOpen, Priority: 2, Created: now.Add(time.Second), Updated: now.Add(time.Second)},
			}
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "list"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			expected := "tick-aaa111\ntick-bbb222\n"
			if stdoutBuf.String() != expected {
				t.Errorf("stdout = %q, want %q", stdoutBuf.String(), expected)
			}
		})

		t.Run("show outputs ID only when quiet", func(t *testing.T) {
			tasks := []task.Task{
				{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
			}
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			expected := "tick-aaa111\n"
			if stdoutBuf.String() != expected {
				t.Errorf("stdout = %q, want %q", stdoutBuf.String(), expected)
			}
		})

		t.Run("init outputs nothing when quiet", func(t *testing.T) {
			dir := t.TempDir()
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
			}
			exitCode := app.Run([]string{"tick", "--quiet", "init"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			if stdoutBuf.String() != "" {
				t.Errorf("quiet init should produce no output, got %q", stdoutBuf.String())
			}
		})
	})

	t.Run("it handles empty list per format", func(t *testing.T) {
		tests := []struct {
			name      string
			flag      string
			checkFunc func(t *testing.T, stdout string)
		}{
			{
				name: "toon",
				flag: "--toon",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					if !strings.Contains(stdout, "tasks[0]") {
						t.Errorf("toon empty list should contain zero-count, got %q", stdout)
					}
				},
			},
			{
				name: "pretty",
				flag: "--pretty",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					expected := "No tasks found.\n"
					if stdout != expected {
						t.Errorf("pretty empty list = %q, want %q", stdout, expected)
					}
				},
			},
			{
				name: "json",
				flag: "--json",
				checkFunc: func(t *testing.T, stdout string) {
					t.Helper()
					trimmed := strings.TrimSpace(stdout)
					if trimmed != "[]" {
						t.Errorf("json empty list = %q, want '[]'", trimmed)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir, _ := setupTickProject(t)
				var stdoutBuf, stderrBuf bytes.Buffer
				app := &App{
					Stdout: &stdoutBuf,
					Stderr: &stderrBuf,
					Getwd:  func() (string, error) { return dir, nil },
				}
				exitCode := app.Run([]string{"tick", tt.flag, "list"})
				if exitCode != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
				}
				tt.checkFunc(t, stdoutBuf.String())
			})
		}
	})

	t.Run("it defaults to TOON when piped, Pretty when TTY", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}

		t.Run("non-TTY defaults to toon", func(t *testing.T) {
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
				IsTTY:  false,
			}
			exitCode := app.Run([]string{"tick", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			// TOON format uses "task{...}:" header
			if !strings.Contains(stdoutBuf.String(), "task{") {
				t.Errorf("non-TTY should default to toon format, got %q", stdoutBuf.String())
			}
		})

		t.Run("TTY defaults to pretty", func(t *testing.T) {
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
				IsTTY:  true,
			}
			exitCode := app.Run([]string{"tick", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			// Pretty format uses "ID:" prefix
			if !strings.Contains(stdoutBuf.String(), "ID:") {
				t.Errorf("TTY should default to pretty format, got %q", stdoutBuf.String())
			}
		})
	})

	t.Run("it respects --toon/--pretty/--json overrides", func(t *testing.T) {
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen, Priority: 1, Created: now, Updated: now},
		}

		t.Run("--toon overrides TTY default", func(t *testing.T) {
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
				IsTTY:  true, // Would default to pretty
			}
			exitCode := app.Run([]string{"tick", "--toon", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			if !strings.Contains(stdoutBuf.String(), "task{") {
				t.Errorf("--toon should force toon format, got %q", stdoutBuf.String())
			}
		})

		t.Run("--pretty overrides piped default", func(t *testing.T) {
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
				IsTTY:  false, // Would default to toon
			}
			exitCode := app.Run([]string{"tick", "--pretty", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			if !strings.Contains(stdoutBuf.String(), "ID:") {
				t.Errorf("--pretty should force pretty format, got %q", stdoutBuf.String())
			}
		})

		t.Run("--json overrides piped default", func(t *testing.T) {
			dir, _ := setupTickProjectWithTasks(t, tasks)
			var stdoutBuf, stderrBuf bytes.Buffer
			app := &App{
				Stdout: &stdoutBuf,
				Stderr: &stderrBuf,
				Getwd:  func() (string, error) { return dir, nil },
				IsTTY:  false, // Would default to toon
			}
			exitCode := app.Run([]string{"tick", "--json", "show", "tick-aaa111"})
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
			}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(stdoutBuf.String())), &obj); err != nil {
				t.Errorf("--json should force JSON format, got error: %v, output: %q", err, stdoutBuf.String())
			}
		})
	})

	t.Run("quiet plus json: quiet wins, no JSON wrapping", func(t *testing.T) {
		openTask := task.Task{
			ID: "tick-aaa111", Title: "Task", Status: task.StatusOpen,
			Priority: 2, Created: now, Updated: now,
		}
		dir, _ := setupTickProjectWithTasks(t, []task.Task{openTask})
		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		// --quiet + --json on a transition => quiet wins, no output
		exitCode := app.Run([]string{"tick", "--quiet", "--json", "start", "tick-aaa111"})
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderrBuf.String())
		}
		if stdoutBuf.String() != "" {
			t.Errorf("--quiet should win over --json for transitions, got %q", stdoutBuf.String())
		}
	})

	t.Run("errors remain plain text to stderr regardless of format", func(t *testing.T) {
		dir, _ := setupTickProject(t)
		var stdoutBuf, stderrBuf bytes.Buffer
		app := &App{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
			Getwd:  func() (string, error) { return dir, nil },
		}
		// --json with a command that errors (show with no args)
		exitCode := app.Run([]string{"tick", "--json", "show"})
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1", exitCode)
		}
		// Error should be plain text, not JSON
		if strings.HasPrefix(stderrBuf.String(), "{") {
			t.Errorf("error should be plain text, not JSON, got %q", stderrBuf.String())
		}
		if !strings.Contains(stderrBuf.String(), "Error:") {
			t.Errorf("stderr should contain 'Error:', got %q", stderrBuf.String())
		}
	})
}
