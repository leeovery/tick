package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const maxNoteTextLen = 500

// Note represents a timestamped text annotation on a task.
type Note struct {
	Text    string    `json:"-"`
	Created time.Time `json:"-"`
}

// noteJSON is the JSON serialization form for Note.
type noteJSON struct {
	Text    string `json:"text"`
	Created string `json:"created"`
}

// MarshalJSON serializes a Note with Created formatted as ISO 8601 UTC string.
func (n Note) MarshalJSON() ([]byte, error) {
	return json.Marshal(noteJSON{
		Text:    n.Text,
		Created: FormatTimestamp(n.Created),
	})
}

// UnmarshalJSON deserializes a Note, parsing the Created timestamp from ISO 8601.
func (n *Note) UnmarshalJSON(data []byte) error {
	var nj noteJSON
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}

	created, err := time.Parse(TimestampFormat, nj.Created)
	if err != nil {
		return fmt.Errorf("invalid note created timestamp %q: %w", nj.Created, err)
	}

	n.Text = nj.Text
	n.Created = created
	return nil
}

// ValidateNoteText checks that note text is non-empty after trimming and at most 500 characters.
func ValidateNoteText(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return errors.New("note text is required and cannot be empty")
	}
	if utf8.RuneCountInString(trimmed) > maxNoteTextLen {
		return fmt.Errorf("note text exceeds maximum length of %d characters", maxNoteTextLen)
	}
	return nil
}

// TrimNoteText removes leading and trailing whitespace from note text.
func TrimNoteText(text string) string {
	return strings.TrimSpace(text)
}
