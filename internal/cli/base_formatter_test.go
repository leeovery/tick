package cli

import "testing"

func TestBaseFormatter(t *testing.T) {
	t.Run("FormatTransition contains Unicode right arrow", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatTransition("tick-a1b2", "open", "in_progress")
		if result != "tick-a1b2: open \u2192 in_progress" {
			t.Errorf("result = %q, want %q", result, "tick-a1b2: open \u2192 in_progress")
		}
	})

	t.Run("FormatTransition matches spec format id: old_status arrow new_status", func(t *testing.T) {
		f := &baseFormatter{}
		tests := []struct {
			name      string
			id        string
			oldStatus string
			newStatus string
			expected  string
		}{
			{
				name:      "open to in_progress",
				id:        "tick-a3f2b7",
				oldStatus: "open",
				newStatus: "in_progress",
				expected:  "tick-a3f2b7: open \u2192 in_progress",
			},
			{
				name:      "in_progress to done",
				id:        "tick-c3d4e5",
				oldStatus: "in_progress",
				newStatus: "done",
				expected:  "tick-c3d4e5: in_progress \u2192 done",
			},
			{
				name:      "done to open via reopen",
				id:        "tick-f6g7h8",
				oldStatus: "done",
				newStatus: "open",
				expected:  "tick-f6g7h8: done \u2192 open",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := f.FormatTransition(tt.id, tt.oldStatus, tt.newStatus)
				if result != tt.expected {
					t.Errorf("result = %q, want %q", result, tt.expected)
				}
			})
		}
	})

	t.Run("FormatDepChange renders add correctly", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatDepChange("added", "tick-c3d4", "tick-a1b2")
		expected := "Dependency added: tick-c3d4 blocked by tick-a1b2"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("FormatDepChange renders remove correctly", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatDepChange("removed", "tick-c3d4", "tick-a1b2")
		expected := "Dependency removed: tick-c3d4 no longer blocked by tick-a1b2"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})
}

func TestAllFormattersProduceConsistentTransitionOutput(t *testing.T) {
	toonFmt := &ToonFormatter{}
	prettyFmt := &PrettyFormatter{}

	id := "tick-a3f2b7"
	oldStatus := "open"
	newStatus := "in_progress"
	expectedArrow := "tick-a3f2b7: open \u2192 in_progress"

	toonResult := toonFmt.FormatTransition(id, oldStatus, newStatus)
	prettyResult := prettyFmt.FormatTransition(id, oldStatus, newStatus)

	if toonResult != expectedArrow {
		t.Errorf("ToonFormatter.FormatTransition = %q, want %q", toonResult, expectedArrow)
	}
	if prettyResult != expectedArrow {
		t.Errorf("PrettyFormatter.FormatTransition = %q, want %q", prettyResult, expectedArrow)
	}
	if toonResult != prettyResult {
		t.Errorf("ToonFormatter and PrettyFormatter produce different transition output: %q vs %q", toonResult, prettyResult)
	}
}
