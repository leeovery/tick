// Package storage provides JSONL persistence and SQLite cache management for Tick tasks.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/leeovery/tick/internal/task"
)

// jsonlTask is the serialization type for JSONL output, controlling field order and optional field omission.
type jsonlTask struct {
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

func toJSONL(t task.Task) jsonlTask {
	jt := jsonlTask{
		ID:          t.ID,
		Title:       t.Title,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Description: t.Description,
		BlockedBy:   t.BlockedBy,
		Parent:      t.Parent,
		Created:     task.FormatTimestamp(t.Created),
		Updated:     task.FormatTimestamp(t.Updated),
	}
	if t.Closed != nil {
		jt.Closed = task.FormatTimestamp(*t.Closed)
	}
	return jt
}

func fromJSONL(jt jsonlTask) (task.Task, error) {
	created, err := time.Parse(task.TimestampFormat, jt.Created)
	if err != nil {
		return task.Task{}, fmt.Errorf("invalid created timestamp %q: %w", jt.Created, err)
	}
	updated, err := time.Parse(task.TimestampFormat, jt.Updated)
	if err != nil {
		return task.Task{}, fmt.Errorf("invalid updated timestamp %q: %w", jt.Updated, err)
	}

	t := task.Task{
		ID:          jt.ID,
		Title:       jt.Title,
		Status:      task.Status(jt.Status),
		Priority:    jt.Priority,
		Description: jt.Description,
		BlockedBy:   jt.BlockedBy,
		Parent:      jt.Parent,
		Created:     created,
		Updated:     updated,
	}

	if jt.Closed != "" {
		closed, err := time.Parse(task.TimestampFormat, jt.Closed)
		if err != nil {
			return task.Task{}, fmt.Errorf("invalid closed timestamp %q: %w", jt.Closed, err)
		}
		t.Closed = &closed
	}

	return t, nil
}

// WriteJSONL writes tasks to a JSONL file using the atomic temp file + fsync + rename pattern.
func WriteJSONL(path string, tasks []task.Task) error {
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

	for _, t := range tasks {
		data, err := json.Marshal(toJSONL(t))
		if err != nil {
			return fmt.Errorf("failed to marshal task %s: %w", t.ID, err)
		}
		data = append(data, '\n')
		if _, err := tmp.Write(data); err != nil {
			return fmt.Errorf("failed to write task %s: %w", t.ID, err)
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

// ReadJSONL reads tasks from a JSONL file, returning one Task per line.
// Empty files return an empty task list. Missing files return an error.
func ReadJSONL(path string) ([]task.Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
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

		var jt jsonlTask
		if err := json.Unmarshal([]byte(line), &jt); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		t, err := fromJSONL(jt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task on line %d: %w", lineNum, err)
		}

		tasks = append(tasks, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return tasks, nil
}
