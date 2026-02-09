// Package storage provides JSONL-based persistent storage for tasks with
// atomic write support using the temp file + fsync + rename pattern.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leeovery/tick/internal/task"
)

// WriteTasks writes tasks to the given path as JSONL (one JSON object per line).
// It uses the atomic write pattern: write to temp file, fsync, then rename.
func WriteTasks(path string, tasks []task.Task) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on error.
	success := false
	defer func() {
		if !success {
			tmp.Close()
			os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmp)
	for _, t := range tasks {
		if err := encoder.Encode(t); err != nil {
			return fmt.Errorf("encoding task %s: %w", t.ID, err)
		}
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}

// ReadTasks reads tasks from a JSONL file at the given path. Each line is
// parsed as a single JSON task object. Empty lines are skipped. Returns an
// error if the file does not exist.
func ReadTasks(path string) ([]task.Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening tasks file: %w", err)
	}
	defer f.Close()

	var tasks []task.Task
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var t task.Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, fmt.Errorf("parsing task at line %d: %w", lineNum, err)
		}
		tasks = append(tasks, t)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks file: %w", err)
	}

	return tasks, nil
}
