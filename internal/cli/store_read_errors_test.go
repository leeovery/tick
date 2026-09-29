package cli

import (
	"testing"
)

const wrongTypedTaskLine = `{"id":"tick-a1b2c3","title":"T","status":"open","priority":"high","created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`

func assertStoreReadError(t *testing.T, dir string, want string, args ...string) {
	t.Helper()
	stdout, stderr, code := runTick(t, dir, args...)
	if code != 1 {
		t.Fatalf("%v exit code = %d, want 1; stdout = %q", args, code, stdout)
	}
	if stderr != "Error: failed to parse tasks.jsonl: "+want+"\n" {
		t.Errorf("%v stderr = %q, want %q", args, stderr, "Error: failed to parse tasks.jsonl: "+want+"\n")
	}
}

func TestStoreReadErrors(t *testing.T) {
	wrongTyped := "line 2 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int"

	t.Run("it names the line, task and reason of a wrong-typed field on list", func(t *testing.T) {
		dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), wrongTypedTaskLine)
		assertStoreReadError(t, dir, wrongTyped, "list")
	})

	t.Run("it names the same line, task and reason on create and rebuild", func(t *testing.T) {
		dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), wrongTypedTaskLine)
		assertStoreReadError(t, dir, wrongTyped, "create", "New task")
		assertStoreReadError(t, dir, wrongTyped, "rebuild")
	})

	t.Run("it names the line, task and timestamp reason of an unparseable created", func(t *testing.T) {
		dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"),
			`{"id":"tick-a1b2c3","title":"T","status":"open","priority":2,"created":"yesterday","updated":"2026-01-19T10:00:00Z"}`)
		assertStoreReadError(t, dir,
			`line 2 (tick-a1b2c3): invalid created timestamp "yesterday": parsing time "yesterday" as "2006-01-02T15:04:05Z": cannot parse "yesterday" as "2006"`,
			"list")
	})

	t.Run("it names a malformed-JSON or whitespace-only line by number alone", func(t *testing.T) {
		for bad, reason := range map[string]string{
			`{"id":"tick-a1b2c3","title":`: "unexpected end of JSON input",
			"   ":                          "unexpected end of JSON input",
			"not valid json":               "invalid character 'o' in literal null (expecting 'u')",
		} {
			dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), bad)
			assertStoreReadError(t, dir, "line 2: "+reason, "list")
		}
	})

	t.Run("it names a failing line with no string id by number alone", func(t *testing.T) {
		emptyCreated := `invalid created timestamp "": parsing time "" as "2006-01-02T15:04:05Z": cannot parse "" as "2006"`
		for bad, reason := range map[string]string{
			`{"title":"T"}`: emptyCreated,
			`{"id":42,"title":"T","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}`: "json: cannot unmarshal number into Go struct field taskJSON.id of type string",
			`null`: emptyCreated,
			`[]`:   "json: cannot unmarshal array into Go value of type task.taskJSON",
		} {
			dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), bad)
			assertStoreReadError(t, dir, "line 2: "+reason, "list")
		}
	})

	t.Run("it counts a skipped empty line 2 when naming a wrong-typed task on line 3", func(t *testing.T) {
		dir, _ := setupRawProject(t, plainTaskLine("tick-aaa111"), "", wrongTypedTaskLine)
		assertStoreReadError(t, dir,
			"line 3 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int",
			"list")
	})
}
