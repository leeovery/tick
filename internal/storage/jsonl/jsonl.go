// Package jsonl provides JSONL (JSON Lines) reading and writing for task storage.
// It implements atomic writes using the temp file + fsync + rename pattern.
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leeovery/tick/internal/task"
)

const timeFormat = "2006-01-02T15:04:05Z"

// taskJSON is the serialization representation for JSONL output.
// It controls field ordering and timestamp format.
type taskJSON struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Description string   `json:"description,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Closed      string   `json:"closed,omitempty"`
}

// toJSON converts a Task to its JSON serialization representation.
func toJSON(t task.Task) taskJSON {
	j := taskJSON{
		ID:          t.ID,
		Title:       t.Title,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Description: t.Description,
		BlockedBy:   t.BlockedBy,
		Parent:      t.Parent,
		Created:     t.Created.UTC().Format(timeFormat),
		Updated:     t.Updated.UTC().Format(timeFormat),
	}
	if t.Closed != nil {
		j.Closed = t.Closed.UTC().Format(timeFormat)
	}
	return j
}

// fromJSON converts a JSON serialization representation back to a Task.
func fromJSON(j taskJSON) (task.Task, error) {
	created, err := time.Parse(timeFormat, j.Created)
	if err != nil {
		return task.Task{}, fmt.Errorf("invalid created timestamp %q: %w", j.Created, err)
	}

	updated, err := time.Parse(timeFormat, j.Updated)
	if err != nil {
		return task.Task{}, fmt.Errorf("invalid updated timestamp %q: %w", j.Updated, err)
	}

	t := task.Task{
		ID:          j.ID,
		Title:       j.Title,
		Status:      task.Status(j.Status),
		Priority:    j.Priority,
		Description: j.Description,
		BlockedBy:   j.BlockedBy,
		Parent:      j.Parent,
		Created:     created,
		Updated:     updated,
	}

	if j.Closed != "" {
		closed, err := time.Parse(timeFormat, j.Closed)
		if err != nil {
			return task.Task{}, fmt.Errorf("invalid closed timestamp %q: %w", j.Closed, err)
		}
		t.Closed = &closed
	}

	return t, nil
}

// WriteTasks writes the given tasks to the specified path using the atomic
// write pattern: write to temp file, fsync, then rename. Each task occupies
// exactly one line as a JSON object with no pretty-printing.
func WriteTasks(path string, tasks []task.Task) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	writer := bufio.NewWriter(tmpFile)
	for _, t := range tasks {
		data, err := json.Marshal(toJSON(t))
		if err != nil {
			return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write task %s: %w", t.ID, err)
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to fsync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// ReadTasks reads tasks from a JSONL file at the specified path.
// Empty lines are skipped. An empty file returns an empty task list.
// A missing file returns an error.
func ReadTasks(path string) ([]task.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tasks file: %w", err)
	}
	return ParseTasks(data)
}

// ParseTasks parses tasks from raw JSONL bytes.
// Empty lines are skipped. Empty input returns an empty task list.
func ParseTasks(data []byte) ([]task.Task, error) {
	var tasks []task.Task

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var j taskJSON
		if err := json.Unmarshal([]byte(line), &j); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		t, err := fromJSON(j)
		if err != nil {
			return nil, fmt.Errorf("invalid task on line %d: %w", lineNum, err)
		}

		tasks = append(tasks, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read tasks data: %w", err)
	}

	return tasks, nil
}
