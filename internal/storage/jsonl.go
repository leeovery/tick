// Package storage provides JSONL file storage and SQLite cache management
// for the Tick task tracker.
package storage

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

// jsonlTask is the JSON serialization format for a task.
// Custom struct to control field ordering and omission of optional fields.
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

const timeFormat = "2006-01-02T15:04:05Z"

func toJSONL(t task.Task) jsonlTask {
	jt := jsonlTask{
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
		jt.Closed = t.Closed.UTC().Format(timeFormat)
	}
	return jt
}

func fromJSONL(jt jsonlTask) (task.Task, error) {
	created, err := time.Parse(timeFormat, jt.Created)
	if err != nil {
		return task.Task{}, fmt.Errorf("parsing created timestamp: %w", err)
	}
	updated, err := time.Parse(timeFormat, jt.Updated)
	if err != nil {
		return task.Task{}, fmt.Errorf("parsing updated timestamp: %w", err)
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
		closed, err := time.Parse(timeFormat, jt.Closed)
		if err != nil {
			return task.Task{}, fmt.Errorf("parsing closed timestamp: %w", err)
		}
		t.Closed = &closed
	}

	return t, nil
}

// ReadJSONL reads tasks from a JSONL file at the given path.
// Returns an empty slice for an empty file.
// Returns an error if the file does not exist.
func ReadJSONL(path string) ([]task.Task, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening JSONL file: %w", err)
	}
	defer f.Close()

	var tasks []task.Task
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var jt jsonlTask
		if err := json.Unmarshal([]byte(line), &jt); err != nil {
			return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}
		t, err := fromJSONL(jt)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		tasks = append(tasks, t)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading JSONL file: %w", err)
	}
	return tasks, nil
}

// ReadJSONLBytes parses tasks from raw JSONL content.
// Returns an empty slice for empty content.
func ReadJSONLBytes(data []byte) ([]task.Task, error) {
	content := string(data)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var tasks []task.Task
	for lineNum, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var jt jsonlTask
		if err := json.Unmarshal([]byte(line), &jt); err != nil {
			return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum+1, err)
		}
		t, err := fromJSONL(jt)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// WriteJSONL writes tasks to a JSONL file at the given path using atomic writes.
// Uses temp file + fsync + rename for crash safety.
func WriteJSONL(path string, tasks []task.Task) error {
	dir := filepath.Dir(path)

	// Create temp file in same directory for atomic rename
	tmp, err := os.CreateTemp(dir, ".tasks-*.jsonl.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write all tasks as JSONL
	for _, t := range tasks {
		jt := toJSONL(t)
		data, err := json.Marshal(jt)
		if err != nil {
			tmp.Close()
			return fmt.Errorf("marshaling task %s: %w", t.ID, err)
		}
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			return fmt.Errorf("writing task %s: %w", t.ID, err)
		}
		if _, err := tmp.WriteString("\n"); err != nil {
			tmp.Close()
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	// fsync to ensure data is on disk
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}

// MarshalJSONL serializes tasks to JSONL bytes.
func MarshalJSONL(tasks []task.Task) ([]byte, error) {
	var result []byte
	for _, t := range tasks {
		jt := toJSONL(t)
		data, err := json.Marshal(jt)
		if err != nil {
			return nil, fmt.Errorf("marshaling task %s: %w", t.ID, err)
		}
		result = append(result, data...)
		result = append(result, '\n')
	}
	return result, nil
}
