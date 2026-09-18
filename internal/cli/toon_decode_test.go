package cli

import (
	"slices"
	"testing"

	toon "github.com/toon-format/toon-go"
)

// decodeToonDoc decodes a TOON document and fails the test if it does not
// decode to a single object.
func decodeToonDoc(t *testing.T, doc string) map[string]any {
	t.Helper()
	value, err := toon.DecodeString(doc)
	if err != nil {
		t.Fatalf("decode failed: %v\ndocument:\n%s", err, doc)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("decoded value is %T, want map[string]any\ndocument:\n%s", value, doc)
	}
	return obj
}

// decodeToonNotes decodes a task detail document and returns its notes rows.
func decodeToonNotes(t *testing.T, doc string) []map[string]any {
	t.Helper()
	raw, ok := decodeToonDoc(t, doc)["notes"]
	if !ok {
		t.Fatalf("key \"notes\" missing from decoded document:\n%s", doc)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("key \"notes\" = %#v, want a list", raw)
	}
	rows := make([]map[string]any, 0, len(items))
	for i, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("notes[%d] = %#v, want an object", i, item)
		}
		rows = append(rows, row)
	}
	return rows
}

func assertToonFields(t *testing.T, doc map[string]any, want map[string]any) {
	t.Helper()
	for key, wantValue := range want {
		got, ok := doc[key]
		if !ok {
			t.Errorf("key %q missing from decoded document", key)
			continue
		}
		if got != wantValue {
			t.Errorf("key %q = %#v, want %#v", key, got, wantValue)
		}
	}
}

func assertToonKeysAbsent(t *testing.T, doc map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := doc[key]; ok {
			t.Errorf("key %q should be absent from decoded document", key)
		}
	}
}

func assertToonStringList(t *testing.T, doc map[string]any, key string, want []string) {
	t.Helper()
	raw, ok := doc[key]
	if !ok {
		t.Fatalf("key %q missing from decoded document", key)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("key %q = %#v, want a list", key, raw)
	}
	got := make([]string, 0, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("key %q item %d = %#v, want a string", key, i, item)
		}
		got = append(got, s)
	}
	if !slices.Equal(got, want) {
		t.Errorf("key %q = %#v, want %#v", key, got, want)
	}
}
