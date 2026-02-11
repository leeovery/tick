// Package storage provides JSONL persistence and SQLite cache management for Tick tasks.
package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leeovery/tick/internal/task"
)

// MarshalJSONL serializes tasks to JSONL-formatted bytes (one JSON object per line).
func MarshalJSONL(tasks []task.Task) ([]byte, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

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

// WriteJSONL writes tasks to a JSONL file using the atomic temp file + fsync + rename pattern.
func WriteJSONL(path string, tasks []task.Task) error {
	data, err := MarshalJSONL(tasks)
	if err != nil {
		return err
	}

	return writeAtomic(path, data)
}

// WriteJSONLRaw writes pre-marshaled JSONL bytes to a file using atomic write.
func WriteJSONLRaw(path string, data []byte) error {
	return writeAtomic(path, data)
}

// writeAtomic writes data to a file using the temp file + fsync + rename pattern.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tasks-*.jsonl.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
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

	if data != nil {
		if _, err := tmp.Write(data); err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %w", path, err)
	}

	success = true
	return nil
}

// ParseJSONL parses tasks from raw JSONL-formatted bytes, returning one Task per line.
// Empty input returns an empty task list.
func ParseJSONL(data []byte) ([]task.Task, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var tasks []task.Task
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		tasks = append(tasks, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL data: %w", err)
	}

	return tasks, nil
}

// ReadJSONL reads tasks from a JSONL file, returning one Task per line.
// Empty files return an empty task list. Missing files return an error.
func ReadJSONL(path string) ([]task.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	return ParseJSONL(data)
}
