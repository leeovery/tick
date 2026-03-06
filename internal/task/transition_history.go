package task

import (
	"encoding/json"
	"fmt"
	"time"
)

// TransitionRecord is a historical record of a status transition on a task.
// It captures the previous and new status, the timestamp, and whether the
// transition was triggered automatically by a cascade.
type TransitionRecord struct {
	From Status    `json:"-"`
	To   Status    `json:"-"`
	At   time.Time `json:"-"`
	Auto bool      `json:"-"`
}

// transitionRecordJSON is the JSON serialization form for TransitionRecord.
type transitionRecordJSON struct {
	From string `json:"from"`
	To   string `json:"to"`
	At   string `json:"at"`
	Auto bool   `json:"auto"`
}

// MarshalJSON serializes a TransitionRecord with At formatted as ISO 8601 UTC string.
func (tr TransitionRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(transitionRecordJSON{
		From: string(tr.From),
		To:   string(tr.To),
		At:   FormatTimestamp(tr.At),
		Auto: tr.Auto,
	})
}

// UnmarshalJSON deserializes a TransitionRecord, parsing the At timestamp from ISO 8601.
func (tr *TransitionRecord) UnmarshalJSON(data []byte) error {
	var jt transitionRecordJSON
	if err := json.Unmarshal(data, &jt); err != nil {
		return err
	}

	at, err := time.Parse(TimestampFormat, jt.At)
	if err != nil {
		return fmt.Errorf("invalid transition at timestamp %q: %w", jt.At, err)
	}

	tr.From = Status(jt.From)
	tr.To = Status(jt.To)
	tr.At = at
	tr.Auto = jt.Auto
	return nil
}
