package task

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SerializeJSONL serializes tasks to JSONL bytes in memory.
// Each task is serialized as a single JSON line. Optional fields are omitted when empty.
func SerializeJSONL(tasks []Task) ([]byte, error) {
	var buf bytes.Buffer
	for _, t := range tasks {
		data, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// WriteJSONL writes tasks to a JSONL file using atomic write (temp file + fsync + rename).
// Each task is serialized as a single JSON line. Optional fields are omitted when empty.
func WriteJSONL(path string, tasks []Task) error {
	data, err := SerializeJSONL(tasks)
	if err != nil {
		return err
	}

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

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tasks file: %w", err)
	}
	return ReadJSONLFromBytes(data)
}

// ReadJSONLFromBytes parses tasks from in-memory JSONL bytes, one task per line.
// Empty lines are skipped. Empty input returns an empty task list.
func ReadJSONLFromBytes(data []byte) ([]Task, error) {
	var tasks []Task
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
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
		return nil, fmt.Errorf("failed to read tasks data: %w", err)
	}

	if tasks == nil {
		tasks = []Task{}
	}

	return tasks, nil
}
