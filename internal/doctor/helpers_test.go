package doctor

import "testing"

func TestFileNotFoundResult(t *testing.T) {
	tests := []struct {
		name      string
		checkName string
	}{
		{
			name:      "returns correct result for JSONL syntax check",
			checkName: "JSONL syntax",
		},
		{
			name:      "returns correct result for ID uniqueness check",
			checkName: "ID uniqueness",
		},
		{
			name:      "returns correct result for arbitrary check name",
			checkName: "Custom check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := fileNotFoundResult(tt.checkName)

			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			r := results[0]

			if r.Name != tt.checkName {
				t.Errorf("Name = %q, want %q", r.Name, tt.checkName)
			}
			if r.Passed {
				t.Error("Passed = true, want false")
			}
			if r.Severity != SeverityError {
				t.Errorf("Severity = %q, want %q", r.Severity, SeverityError)
			}
			if r.Details != "tasks.jsonl not found" {
				t.Errorf("Details = %q, want %q", r.Details, "tasks.jsonl not found")
			}
			if r.Suggestion != "Run tick init or verify .tick directory" {
				t.Errorf("Suggestion = %q, want %q", r.Suggestion, "Run tick init or verify .tick directory")
			}
		})
	}
}
