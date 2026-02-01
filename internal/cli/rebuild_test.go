package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRebuildCommand(t *testing.T) {
	t.Run("rebuilds cache from JSONL", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task one")
		createTask(t, dir, "Task two")

		stdout, _, code := runCmd(t, dir, "tick", "rebuild")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "2") {
			t.Errorf("expected task count in output, got %q", stdout)
		}

		// Verify data still accessible after rebuild.
		listOut, _, _ := runCmd(t, dir, "tick", "list")
		if !strings.Contains(listOut, "Task one") {
			t.Error("task one should be listed after rebuild")
		}
	})

	t.Run("handles missing cache.db (fresh build)", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		// Delete cache.db
		os.Remove(filepath.Join(dir, ".tick", "cache.db"))

		_, _, code := runCmd(t, dir, "tick", "rebuild")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}

		// Verify data accessible after rebuild.
		listOut, _, _ := runCmd(t, dir, "tick", "list")
		if !strings.Contains(listOut, "Task") {
			t.Error("task should be listed after rebuild from scratch")
		}
	})

	t.Run("overwrites valid existing cache", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		// Rebuild twice — second should succeed.
		_, _, code1 := runCmd(t, dir, "tick", "rebuild")
		if code1 != 0 {
			t.Fatalf("first rebuild failed")
		}
		_, _, code2 := runCmd(t, dir, "tick", "rebuild")
		if code2 != 0 {
			t.Fatalf("second rebuild failed")
		}
	})

	t.Run("handles empty JSONL", func(t *testing.T) {
		dir := initTickDir(t)

		stdout, _, code := runCmd(t, dir, "tick", "rebuild")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if !strings.Contains(stdout, "0") {
			t.Errorf("expected 0 tasks in output, got %q", stdout)
		}
	})

	t.Run("outputs confirmation message with task count", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "A")
		createTask(t, dir, "B")
		createTask(t, dir, "C")

		stdout, _, _ := runCmd(t, dir, "tick", "rebuild")
		if !strings.Contains(stdout, "3") {
			t.Errorf("expected count of 3, got %q", stdout)
		}
		if !strings.Contains(strings.ToLower(stdout), "rebuilt") {
			t.Errorf("expected 'rebuilt' in message, got %q", stdout)
		}
	})

	t.Run("suppresses output with --quiet", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		code := app.Run([]string{"tick", "--quiet", "rebuild"}, dir)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		if outBuf.Len() != 0 {
			t.Errorf("--quiet should suppress output, got %q", outBuf.String())
		}
	})

	t.Run("logs rebuild steps with --verbose", func(t *testing.T) {
		dir := initTickDir(t)
		createTask(t, dir, "Task")

		var outBuf, errBuf bytes.Buffer
		app := NewApp(&outBuf, &errBuf)
		app.Run([]string{"tick", "--verbose", "rebuild"}, dir)

		stderr := errBuf.String()
		if !strings.Contains(stderr, "verbose:") {
			t.Errorf("expected verbose output on stderr, got %q", stderr)
		}
	})
}
