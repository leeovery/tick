package task

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTransitionRecordMarshalJSON(t *testing.T) {
	t.Run("it marshals Transition to JSON with ISO 8601 timestamp", func(t *testing.T) {
		at := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
		tr := TransitionRecord{
			From: StatusOpen,
			To:   StatusInProgress,
			At:   at,
			Auto: false,
		}

		data, err := json.Marshal(tr)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal raw error: %v", err)
		}

		if raw["from"] != "open" {
			t.Errorf("from = %q, want %q", raw["from"], "open")
		}
		if raw["to"] != "in_progress" {
			t.Errorf("to = %q, want %q", raw["to"], "in_progress")
		}
		if raw["at"] != "2026-03-05T14:30:00Z" {
			t.Errorf("at = %q, want %q", raw["at"], "2026-03-05T14:30:00Z")
		}
		if raw["auto"] != false {
			t.Errorf("auto = %v, want false", raw["auto"])
		}
	})

	t.Run("it unmarshals Transition from valid JSON", func(t *testing.T) {
		jsonStr := `{"from":"in_progress","to":"done","at":"2026-03-05T16:00:00Z","auto":true}`
		var tr TransitionRecord
		if err := json.Unmarshal([]byte(jsonStr), &tr); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if tr.From != StatusInProgress {
			t.Errorf("From = %q, want %q", tr.From, StatusInProgress)
		}
		if tr.To != StatusDone {
			t.Errorf("To = %q, want %q", tr.To, StatusDone)
		}
		expectedAt := time.Date(2026, 3, 5, 16, 0, 0, 0, time.UTC)
		if !tr.At.Equal(expectedAt) {
			t.Errorf("At = %v, want %v", tr.At, expectedAt)
		}
		if tr.Auto != true {
			t.Errorf("Auto = %v, want true", tr.Auto)
		}
	})

	t.Run("it round-trips Transition through JSON", func(t *testing.T) {
		at := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
		original := TransitionRecord{
			From: StatusOpen,
			To:   StatusDone,
			At:   at,
			Auto: true,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got TransitionRecord
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if got.From != original.From {
			t.Errorf("From = %q, want %q", got.From, original.From)
		}
		if got.To != original.To {
			t.Errorf("To = %q, want %q", got.To, original.To)
		}
		if !got.At.Equal(original.At) {
			t.Errorf("At = %v, want %v", got.At, original.At)
		}
		if got.Auto != original.Auto {
			t.Errorf("Auto = %v, want %v", got.Auto, original.Auto)
		}
	})

	t.Run("it preserves auto boolean false vs true", func(t *testing.T) {
		at := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)

		falseRecord := TransitionRecord{From: StatusOpen, To: StatusInProgress, At: at, Auto: false}
		trueRecord := TransitionRecord{From: StatusOpen, To: StatusDone, At: at, Auto: true}

		falseData, err := json.Marshal(falseRecord)
		if err != nil {
			t.Fatalf("Marshal false error: %v", err)
		}
		trueData, err := json.Marshal(trueRecord)
		if err != nil {
			t.Fatalf("Marshal true error: %v", err)
		}

		var gotFalse TransitionRecord
		if err := json.Unmarshal(falseData, &gotFalse); err != nil {
			t.Fatalf("Unmarshal false error: %v", err)
		}
		var gotTrue TransitionRecord
		if err := json.Unmarshal(trueData, &gotTrue); err != nil {
			t.Fatalf("Unmarshal true error: %v", err)
		}

		if gotFalse.Auto != false {
			t.Errorf("Auto = %v, want false", gotFalse.Auto)
		}
		if gotTrue.Auto != true {
			t.Errorf("Auto = %v, want true", gotTrue.Auto)
		}
	})
}

func TestTransitionRecordTaskJSON(t *testing.T) {
	t.Run("it omits transitions from task JSON when empty", func(t *testing.T) {
		created := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "No transitions",
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
		if strings.Contains(s, `"transitions"`) {
			t.Errorf("transitions field should be omitted when nil/empty, got: %s", s)
		}
	})

	t.Run("it includes transitions in task JSON when present", func(t *testing.T) {
		created := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
		transAt := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "With transitions",
			Status:   StatusInProgress,
			Priority: 2,
			Transitions: []TransitionRecord{
				{From: StatusOpen, To: StatusInProgress, At: transAt, Auto: false},
			},
			Created: created,
			Updated: created,
		}

		data, err := json.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got Task
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if len(got.Transitions) != 1 {
			t.Fatalf("expected 1 transition, got %d", len(got.Transitions))
		}
		if got.Transitions[0].From != StatusOpen {
			t.Errorf("From = %q, want %q", got.Transitions[0].From, StatusOpen)
		}
		if got.Transitions[0].To != StatusInProgress {
			t.Errorf("To = %q, want %q", got.Transitions[0].To, StatusInProgress)
		}
		if !got.Transitions[0].At.Equal(transAt) {
			t.Errorf("At = %v, want %v", got.Transitions[0].At, transAt)
		}
		if got.Transitions[0].Auto != false {
			t.Errorf("Auto = %v, want false", got.Transitions[0].Auto)
		}
	})

	t.Run("it deserializes task without transitions field", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"Legacy","status":"open","priority":2,"created":"2026-03-05T10:00:00Z","updated":"2026-03-05T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if tk.Transitions != nil {
			t.Errorf("Transitions = %v, want nil for backward compat", tk.Transitions)
		}
	})
}
