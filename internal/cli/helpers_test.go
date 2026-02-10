package cli

import (
	"testing"
	"time"

	"github.com/leeovery/tick/internal/task"
)

func TestParseCommaSeparatedIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"single ID",
			"tick-aaa111",
			[]string{"tick-aaa111"},
		},
		{
			"multiple IDs",
			"tick-aaa111,tick-bbb222,tick-ccc333",
			[]string{"tick-aaa111", "tick-bbb222", "tick-ccc333"},
		},
		{
			"whitespace around IDs",
			"tick-aaa111 , tick-bbb222 , tick-ccc333",
			[]string{"tick-aaa111", "tick-bbb222", "tick-ccc333"},
		},
		{
			"empty string",
			"",
			nil,
		},
		{
			"only commas and whitespace",
			" , , ",
			nil,
		},
		{
			"normalizes to lowercase",
			"TICK-AAA111,Tick-BBB222",
			[]string{"tick-aaa111", "tick-bbb222"},
		},
		{
			"filters empty segments from trailing comma",
			"tick-aaa111,",
			[]string{"tick-aaa111"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommaSeparatedIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseCommaSeparatedIDs(%q) returned %d items %v, want %d items %v",
					tt.input, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCommaSeparatedIDs(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestApplyBlocks(t *testing.T) {
	t.Run("it appends sourceID to matching tasks BlockedBy", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 || tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001]", tasks[0].BlockedBy)
		}
		// Unmatched task should not be modified
		if len(tasks[1].BlockedBy) != 0 {
			t.Errorf("tasks[1].BlockedBy = %v, want empty", tasks[1].BlockedBy)
		}
	})

	t.Run("it sets Updated timestamp on modified tasks", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111"}, applyNow)

		if !tasks[0].Updated.Equal(applyNow) {
			t.Errorf("tasks[0].Updated = %v, want %v", tasks[0].Updated, applyNow)
		}
		// Unmatched task should retain original Updated
		if !tasks[1].Updated.Equal(now) {
			t.Errorf("tasks[1].Updated = %v, want %v (unchanged)", tasks[1].Updated, now)
		}
	})

	t.Run("it is a no-op with non-existent blockIDs", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-nonexist"}, applyNow)

		if len(tasks[0].BlockedBy) != 0 {
			t.Errorf("tasks[0].BlockedBy = %v, want empty", tasks[0].BlockedBy)
		}
		if !tasks[0].Updated.Equal(now) {
			t.Errorf("tasks[0].Updated = %v, want %v (unchanged)", tasks[0].Updated, now)
		}
	})

	t.Run("it handles multiple blockIDs", func(t *testing.T) {
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
		tasks := []task.Task{
			{ID: "tick-aaa111", Title: "Task A", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-bbb222", Title: "Task B", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
			{ID: "tick-ccc333", Title: "Task C", Status: task.StatusOpen, Priority: 2, Created: now, Updated: now},
		}

		applyNow := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
		applyBlocks(tasks, "tick-src001", []string{"tick-aaa111", "tick-ccc333"}, applyNow)

		if len(tasks[0].BlockedBy) != 1 || tasks[0].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[0].BlockedBy = %v, want [tick-src001]", tasks[0].BlockedBy)
		}
		if len(tasks[1].BlockedBy) != 0 {
			t.Errorf("tasks[1].BlockedBy = %v, want empty", tasks[1].BlockedBy)
		}
		if len(tasks[2].BlockedBy) != 1 || tasks[2].BlockedBy[0] != "tick-src001" {
			t.Errorf("tasks[2].BlockedBy = %v, want [tick-src001]", tasks[2].BlockedBy)
		}
	})
}
