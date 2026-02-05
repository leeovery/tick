// Package storage provides JSONL file storage for tasks with atomic writes.
package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/leeovery/tick/internal/task"
)

// WriteJSONL writes tasks to a JSONL file using atomic write pattern.
// Each task is serialized as a single JSON line.
// Uses temp file + fsync + rename for crash safety.
func WriteJSONL(path string, tasks []task.Task) error {
	dir := filepath.Dir(path)

	// Create temp file in same directory for atomic rename
	temp, err := os.CreateTemp(dir, ".tasks-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tempPath)
		}
	}()

	// Write each task as a JSON line
	encoder := json.NewEncoder(temp)
	for _, t := range tasks {
		if err := encoder.Encode(t); err != nil {
			temp.Close()
			return err
		}
	}

	// Fsync to flush to disk
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	success = true
	return nil
}

// ReadJSONL reads tasks from a JSONL file.
// Returns empty slice for empty file, error for missing file.
func ReadJSONL(path string) ([]task.Task, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []task.Task
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		var t task.Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return empty slice (not nil) for empty file
	if tasks == nil {
		tasks = []task.Task{}
	}

	return tasks, nil
}
