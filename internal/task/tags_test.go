package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestTagMarshalJSON(t *testing.T) {
	t.Run("it marshals tags to JSON array", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "Tagged task",
			Status:   StatusOpen,
			Priority: 2,
			Tags:     []string{"frontend", "ui-component"},
			Created:  created,
			Updated:  created,
		}

		data, err := json.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal raw error: %v", err)
		}

		tagsRaw, ok := raw["tags"]
		if !ok {
			t.Fatal("expected 'tags' key in JSON output, not found")
		}
		tagsArr, ok := tagsRaw.([]interface{})
		if !ok {
			t.Fatalf("expected tags to be an array, got %T", tagsRaw)
		}
		if len(tagsArr) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(tagsArr))
		}
		if tagsArr[0] != "frontend" {
			t.Errorf("tags[0] = %q, want %q", tagsArr[0], "frontend")
		}
		if tagsArr[1] != "ui-component" {
			t.Errorf("tags[1] = %q, want %q", tagsArr[1], "ui-component")
		}
	})

	t.Run("it omits empty tags from JSON (omitempty)", func(t *testing.T) {
		created := time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)
		tk := Task{
			ID:       "tick-a1b2c3",
			Title:    "No tags",
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
		if strings.Contains(s, `"tags"`) {
			t.Errorf("tags field should be omitted when nil/empty, got: %s", s)
		}
	})

	t.Run("it unmarshals tags from JSON", func(t *testing.T) {
		jsonStr := `{"id":"tick-a1b2c3","title":"Tagged","status":"open","priority":2,"tags":["frontend","v2"],"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`
		var tk Task
		if err := json.Unmarshal([]byte(jsonStr), &tk); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(tk.Tags) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(tk.Tags))
		}
		if tk.Tags[0] != "frontend" {
			t.Errorf("Tags[0] = %q, want %q", tk.Tags[0], "frontend")
		}
		if tk.Tags[1] != "v2" {
			t.Errorf("Tags[1] = %q, want %q", tk.Tags[1], "v2")
		}
	})
}

func TestValidateTag(t *testing.T) {
	t.Run("it validates valid kebab-case tag (frontend)", func(t *testing.T) {
		err := ValidateTag("frontend")
		if err != nil {
			t.Errorf("ValidateTag(%q) returned error: %v", "frontend", err)
		}
	})

	t.Run("it validates multi-segment kebab-case tag (ui-component)", func(t *testing.T) {
		err := ValidateTag("ui-component")
		if err != nil {
			t.Errorf("ValidateTag(%q) returned error: %v", "ui-component", err)
		}
	})

	t.Run("it rejects tag with double hyphens", func(t *testing.T) {
		err := ValidateTag("my--tag")
		if err == nil {
			t.Fatal("expected error for tag with double hyphens, got nil")
		}
	})

	t.Run("it rejects tag with leading hyphen", func(t *testing.T) {
		err := ValidateTag("-frontend")
		if err == nil {
			t.Fatal("expected error for tag with leading hyphen, got nil")
		}
	})

	t.Run("it rejects tag with trailing hyphen", func(t *testing.T) {
		err := ValidateTag("frontend-")
		if err == nil {
			t.Fatal("expected error for tag with trailing hyphen, got nil")
		}
	})

	t.Run("it rejects tag with spaces", func(t *testing.T) {
		err := ValidateTag("my tag")
		if err == nil {
			t.Fatal("expected error for tag with spaces, got nil")
		}
	})

	t.Run("it rejects tag exceeding 30 chars", func(t *testing.T) {
		tag := strings.Repeat("a", 31)
		err := ValidateTag(tag)
		if err == nil {
			t.Fatalf("expected error for tag exceeding 30 chars, got nil")
		}
	})

	t.Run("it accepts tag at exactly 30 chars", func(t *testing.T) {
		tag := strings.Repeat("a", 30)
		err := ValidateTag(tag)
		if err != nil {
			t.Errorf("ValidateTag(%q) returned error: %v", tag, err)
		}
	})

	t.Run("it rejects empty tag", func(t *testing.T) {
		err := ValidateTag("")
		if err == nil {
			t.Fatal("expected error for empty tag, got nil")
		}
	})

	t.Run("it validates additional kebab-case formats", func(t *testing.T) {
		for _, tag := range []string{"v2", "a1-b2-c3"} {
			t.Run(tag, func(t *testing.T) {
				err := ValidateTag(tag)
				if err != nil {
					t.Errorf("ValidateTag(%q) returned error: %v", tag, err)
				}
			})
		}
	})
}

func TestNormalizeTag(t *testing.T) {
	t.Run("it normalizes tag to trimmed lowercase", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"  Frontend  ", "frontend"},
			{"UI-COMPONENT", "ui-component"},
			{" V2 ", "v2"},
			{"already-lowercase", "already-lowercase"},
			{"", ""},
			{"  ", ""},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
				got := NormalizeTag(tt.input)
				if got != tt.want {
					t.Errorf("NormalizeTag(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestDeduplicateTags(t *testing.T) {
	t.Run("it deduplicates tags preserving first-occurrence order", func(t *testing.T) {
		input := []string{"frontend", "backend", "frontend", "api", "backend"}
		got := DeduplicateTags(input)
		want := []string{"frontend", "backend", "api"}
		if len(got) != len(want) {
			t.Fatalf("DeduplicateTags returned %d tags, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("it filters empty strings from tag list", func(t *testing.T) {
		input := []string{"frontend", "", "backend", "  ", ""}
		got := DeduplicateTags(input)
		want := []string{"frontend", "backend"}
		if len(got) != len(want) {
			t.Fatalf("DeduplicateTags returned %d tags, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("it normalizes tags during deduplication", func(t *testing.T) {
		input := []string{"Frontend", "FRONTEND", "frontend"}
		got := DeduplicateTags(input)
		want := []string{"frontend"}
		if len(got) != len(want) {
			t.Fatalf("DeduplicateTags returned %d tags, want %d", len(got), len(want))
		}
		if got[0] != "frontend" {
			t.Errorf("got[0] = %q, want %q", got[0], "frontend")
		}
	})
}

func TestValidateTags(t *testing.T) {
	t.Run("it accepts 11 tags deduped to 10", func(t *testing.T) {
		tags := []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10", "t1"}
		err := ValidateTags(tags)
		if err != nil {
			t.Errorf("ValidateTags returned error for 11 tags deduped to 10: %v", err)
		}
	})

	t.Run("it rejects 11 unique tags", func(t *testing.T) {
		tags := make([]string, 11)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag%d", i)
		}
		err := ValidateTags(tags)
		if err == nil {
			t.Fatal("expected error for 11 unique tags, got nil")
		}
	})

	t.Run("it filters empty strings from tag list", func(t *testing.T) {
		tags := []string{"frontend", "", "backend", ""}
		err := ValidateTags(tags)
		if err != nil {
			t.Errorf("ValidateTags returned error for tags with empty strings: %v", err)
		}
	})

	t.Run("it validates each tag individually", func(t *testing.T) {
		tags := []string{"valid-tag", "my--bad"}
		err := ValidateTags(tags)
		if err == nil {
			t.Fatal("expected error for invalid tag in list, got nil")
		}
	})

	t.Run("it accepts empty tag list", func(t *testing.T) {
		err := ValidateTags(nil)
		if err != nil {
			t.Errorf("ValidateTags(nil) returned error: %v", err)
		}
		err = ValidateTags([]string{})
		if err != nil {
			t.Errorf("ValidateTags([]string{}) returned error: %v", err)
		}
	})
}
