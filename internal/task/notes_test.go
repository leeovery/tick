package task

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateNoteText(t *testing.T) {
	t.Run("it validates non-empty note text", func(t *testing.T) {
		err := ValidateNoteText("This is a valid note")
		if err != nil {
			t.Errorf("expected no error for valid note text, got: %v", err)
		}
	})

	t.Run("it rejects empty note text", func(t *testing.T) {
		err := ValidateNoteText("")
		if err == nil {
			t.Fatal("expected error for empty note text, got nil")
		}
	})

	t.Run("it rejects whitespace-only note text", func(t *testing.T) {
		err := ValidateNoteText("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only note text, got nil")
		}
	})

	t.Run("it accepts note text exactly 500 chars", func(t *testing.T) {
		text := strings.Repeat("a", 500)
		err := ValidateNoteText(text)
		if err != nil {
			t.Errorf("expected no error for 500-char note text, got: %v", err)
		}
	})

	t.Run("it rejects note text of 501 chars", func(t *testing.T) {
		text := strings.Repeat("a", 501)
		err := ValidateNoteText(text)
		if err == nil {
			t.Fatal("expected error for 501-char note text, got nil")
		}
	})

	t.Run("it trims note text before validation", func(t *testing.T) {
		trimmed := TrimNoteText("  hello world  ")
		expected := "hello world"
		if trimmed != expected {
			t.Errorf("TrimNoteText(%q) = %q, want %q", "  hello world  ", trimmed, expected)
		}
	})
}

func TestNoteMarshalJSON(t *testing.T) {
	t.Run("it serializes Note to JSON with timestamp", func(t *testing.T) {
		created := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
		note := Note{
			Text:    "Decided to use approach B",
			Created: created,
		}

		data, err := json.Marshal(note)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal raw error: %v", err)
		}

		text, ok := raw["text"]
		if !ok {
			t.Fatal("expected 'text' key in JSON output, not found")
		}
		if text != "Decided to use approach B" {
			t.Errorf("text = %q, want %q", text, "Decided to use approach B")
		}

		createdStr, ok := raw["created"]
		if !ok {
			t.Fatal("expected 'created' key in JSON output, not found")
		}
		if createdStr != "2026-02-27T10:00:00Z" {
			t.Errorf("created = %q, want %q", createdStr, "2026-02-27T10:00:00Z")
		}
	})

	t.Run("it deserializes Note from JSON", func(t *testing.T) {
		jsonStr := `{"text":"Progress update","created":"2026-02-27T14:30:00Z"}`
		var note Note
		if err := json.Unmarshal([]byte(jsonStr), &note); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if note.Text != "Progress update" {
			t.Errorf("Text = %q, want %q", note.Text, "Progress update")
		}

		expectedTime := time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC)
		if !note.Created.Equal(expectedTime) {
			t.Errorf("Created = %v, want %v", note.Created, expectedTime)
		}
	})
}

func TestNoteTaskJSON(t *testing.T) {
	t.Run("it round-trips notes through Task JSON serialization", func(t *testing.T) {
		created := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
		noteTime1 := time.Date(2026, 2, 27, 11, 0, 0, 0, time.UTC)
		noteTime2 := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

		original := Task{
			ID:       "tick-a1b2c3",
			Title:    "Task with notes",
			Status:   StatusOpen,
			Priority: 2,
			Notes: []Note{
				{Text: "First note", Created: noteTime1},
				{Text: "Second note", Created: noteTime2},
			},
			Created: created,
			Updated: created,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got Task
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if len(got.Notes) != 2 {
			t.Fatalf("expected 2 notes, got %d", len(got.Notes))
		}
		if got.Notes[0].Text != "First note" {
			t.Errorf("Notes[0].Text = %q, want %q", got.Notes[0].Text, "First note")
		}
		if !got.Notes[0].Created.Equal(noteTime1) {
			t.Errorf("Notes[0].Created = %v, want %v", got.Notes[0].Created, noteTime1)
		}
		if got.Notes[1].Text != "Second note" {
			t.Errorf("Notes[1].Text = %q, want %q", got.Notes[1].Text, "Second note")
		}
		if !got.Notes[1].Created.Equal(noteTime2) {
			t.Errorf("Notes[1].Created = %v, want %v", got.Notes[1].Created, noteTime2)
		}
	})

	t.Run("it omits empty notes slice from Task JSON", func(t *testing.T) {
		created := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "No notes",
			Status:   StatusOpen,
			Priority: 2,
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		s := string(data)
		if strings.Contains(s, `"notes"`) {
			t.Errorf("notes field should be omitted when nil/empty, got: %s", s)
		}
	})
}
