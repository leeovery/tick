package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestValidateRef(t *testing.T) {
	t.Run("it validates ref with comma is rejected", func(t *testing.T) {
		err := ValidateRef("gh-123,456")
		if err == nil {
			t.Fatal("expected error for ref with comma, got nil")
		}
	})

	t.Run("it validates ref with whitespace is rejected", func(t *testing.T) {
		err := ValidateRef("gh 123")
		if err == nil {
			t.Fatal("expected error for ref with whitespace, got nil")
		}
	})

	t.Run("it accepts ref at 200 chars", func(t *testing.T) {
		ref := strings.Repeat("a", 200)
		err := ValidateRef(ref)
		if err != nil {
			t.Errorf("ValidateRef(%d chars) returned error: %v", len(ref), err)
		}
	})

	t.Run("it rejects ref at 201 chars", func(t *testing.T) {
		ref := strings.Repeat("a", 201)
		err := ValidateRef(ref)
		if err == nil {
			t.Fatal("expected error for 201-char ref, got nil")
		}
	})

	t.Run("it rejects empty ref", func(t *testing.T) {
		err := ValidateRef("")
		if err == nil {
			t.Fatal("expected error for empty ref, got nil")
		}
	})

	t.Run("it rejects whitespace-only ref", func(t *testing.T) {
		err := ValidateRef("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only ref, got nil")
		}
	})

	t.Run("it trims ref before validation", func(t *testing.T) {
		err := ValidateRef("  gh-123  ")
		if err != nil {
			t.Errorf("ValidateRef with surrounding whitespace returned error: %v", err)
		}
	})

	t.Run("it accepts valid ref formats", func(t *testing.T) {
		for _, ref := range []string{"gh-123", "JIRA-456", "https://example.com/issue/1"} {
			t.Run(ref, func(t *testing.T) {
				err := ValidateRef(ref)
				if err != nil {
					t.Errorf("ValidateRef(%q) returned error: %v", ref, err)
				}
			})
		}
	})
}

func TestValidateRefs(t *testing.T) {
	t.Run("it deduplicates refs silently", func(t *testing.T) {
		refs := []string{"gh-123", "gh-456", "gh-123"}
		err := ValidateRefs(refs)
		if err != nil {
			t.Errorf("ValidateRefs returned error for duplicated refs: %v", err)
		}
	})

	t.Run("it rejects 11 unique refs", func(t *testing.T) {
		refs := make([]string, 11)
		for i := range refs {
			refs[i] = fmt.Sprintf("ref-%d", i)
		}
		err := ValidateRefs(refs)
		if err == nil {
			t.Fatal("expected error for 11 unique refs, got nil")
		}
	})

	t.Run("it accepts 11 refs deduped to 10", func(t *testing.T) {
		refs := []string{"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7", "r8", "r9", "r0"}
		err := ValidateRefs(refs)
		if err != nil {
			t.Errorf("ValidateRefs returned error for 11 refs deduped to 10: %v", err)
		}
	})

	t.Run("it validates each ref individually", func(t *testing.T) {
		refs := []string{"valid-ref", "has space"}
		err := ValidateRefs(refs)
		if err == nil {
			t.Fatal("expected error for invalid ref in list, got nil")
		}
	})

	t.Run("it accepts empty ref list", func(t *testing.T) {
		err := ValidateRefs(nil)
		if err != nil {
			t.Errorf("ValidateRefs(nil) returned error: %v", err)
		}
		err = ValidateRefs([]string{})
		if err != nil {
			t.Errorf("ValidateRefs([]string{}) returned error: %v", err)
		}
	})
}

func TestParseRefs(t *testing.T) {
	t.Run("it splits comma-separated input", func(t *testing.T) {
		refs, err := ParseRefs("gh-123,JIRA-456")
		if err != nil {
			t.Fatalf("ParseRefs returned error: %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("expected 2 refs, got %d", len(refs))
		}
		if refs[0] != "gh-123" {
			t.Errorf("refs[0] = %q, want %q", refs[0], "gh-123")
		}
		if refs[1] != "JIRA-456" {
			t.Errorf("refs[1] = %q, want %q", refs[1], "JIRA-456")
		}
	})

	t.Run("it trims whitespace around comma-separated values", func(t *testing.T) {
		refs, err := ParseRefs("  gh-123 , JIRA-456  ")
		if err != nil {
			t.Fatalf("ParseRefs returned error: %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("expected 2 refs, got %d", len(refs))
		}
		if refs[0] != "gh-123" {
			t.Errorf("refs[0] = %q, want %q", refs[0], "gh-123")
		}
		if refs[1] != "JIRA-456" {
			t.Errorf("refs[1] = %q, want %q", refs[1], "JIRA-456")
		}
	})

	t.Run("it deduplicates parsed refs", func(t *testing.T) {
		refs, err := ParseRefs("gh-123,gh-456,gh-123")
		if err != nil {
			t.Fatalf("ParseRefs returned error: %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("expected 2 refs after dedup, got %d", len(refs))
		}
	})

	t.Run("it rejects ref with whitespace after parse", func(t *testing.T) {
		_, err := ParseRefs("gh 123,JIRA-456")
		if err == nil {
			t.Fatal("expected error for ref with whitespace, got nil")
		}
	})

	t.Run("it rejects empty input", func(t *testing.T) {
		_, err := ParseRefs("")
		if err == nil {
			t.Fatal("expected error for empty input, got nil")
		}
	})
}

func TestRefMarshalJSON(t *testing.T) {
	t.Run("it round-trips refs through Task JSON", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		original := Task{
			ID:       "tick-a1b2c3",
			Title:    "Task with refs",
			Status:   StatusOpen,
			Priority: 2,
			Refs:     []string{"gh-123", "JIRA-456"},
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got Task
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if len(got.Refs) != 2 {
			t.Fatalf("expected 2 refs, got %d", len(got.Refs))
		}
		if got.Refs[0] != "gh-123" {
			t.Errorf("Refs[0] = %q, want %q", got.Refs[0], "gh-123")
		}
		if got.Refs[1] != "JIRA-456" {
			t.Errorf("Refs[1] = %q, want %q", got.Refs[1], "JIRA-456")
		}
	})

	t.Run("it omits empty refs from JSON", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "No refs",
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
		if strings.Contains(s, `"refs"`) {
			t.Errorf("refs field should be omitted when nil/empty, got: %s", s)
		}
	})

	t.Run("it unmarshals refs from JSON", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"With refs","status":"open","priority":2,"refs":["gh-123","JIRA-456"],"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(tk.Refs) != 2 {
			t.Fatalf("expected 2 refs, got %d", len(tk.Refs))
		}
		if tk.Refs[0] != "gh-123" {
			t.Errorf("Refs[0] = %q, want %q", tk.Refs[0], "gh-123")
		}
		if tk.Refs[1] != "JIRA-456" {
			t.Errorf("Refs[1] = %q, want %q", tk.Refs[1], "JIRA-456")
		}
	})

	t.Run("it unmarshals task without refs field (backward compat)", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"Legacy","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if tk.Refs != nil {
			t.Errorf("Refs = %v, want nil for backward compat", tk.Refs)
		}
	})
}
