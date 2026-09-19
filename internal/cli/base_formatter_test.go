package cli

import "testing"

func TestBaseFormatter(t *testing.T) {
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

func TestBaseFormatterFormatRemoval(t *testing.T) {
	t.Run("it formats single task removal", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
		})
		expected := `Removed tick-a1b2 "My task"`
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats multiple task removal", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "First task"},
				{ID: "tick-c3d4", Title: "Second task"},
			},
		})
		expected := "Removed tick-a1b2 \"First task\"\nRemoved tick-c3d4 \"Second task\""
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats removal with dependency updates", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed: []RemovedTask{
				{ID: "tick-a1b2", Title: "My task"},
			},
			DepsUpdated: []string{"tick-e5f6", "tick-g7h8"},
		})
		expected := "Removed tick-a1b2 \"My task\"\nUpdated dependencies on tick-e5f6, tick-g7h8"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("it formats removal with empty removed list", func(t *testing.T) {
		f := &baseFormatter{}
		result := f.FormatRemoval(RemovalResult{
			Removed:     []RemovedTask{},
			DepsUpdated: []string{},
		})
		if result != "" {
			t.Errorf("result = %q, want empty string", result)
		}
	})
}
