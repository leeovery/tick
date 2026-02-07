package task

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteJSONL writes tasks to a JSONL file using atomic write (temp file + fsync + rename).
// Each task is serialized as a single JSON line. Optional fields are omitted when empty.
func WriteJSONL(path string, tasks []Task) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tasks-*.jsonl.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			tmp.Close()
			os.Remove(tmpPath)
		}
	}()

	for _, t := range tasks {
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
		}
		if _, err := tmp.Write(data); err != nil {
			return fmt.Errorf("failed to write task %s: %w", t.ID, err)
		}
		if _, err := tmp.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	// Fsync to ensure data is flushed to disk
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to fsync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// ReadJSONL reads tasks from a JSONL file, parsing one task per line.
// Empty lines are skipped. Returns an error if the file does not exist.
// An empty file (0 bytes) returns an empty task list.
func ReadJSONL(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tasks file: %w", err)
	}
	defer f.Close()

	var tasks []Task
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var t Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, fmt.Errorf("failed to parse task on line %d: %w", lineNum, err)
		}
		tasks = append(tasks, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	if tasks == nil {
		tasks = []Task{}
	}

	return tasks, nil
}
