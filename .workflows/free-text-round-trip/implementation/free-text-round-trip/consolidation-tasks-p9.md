# Consolidation Tasks: Free Text Round Trip (Phase 9)

## Task 1: A Task-List Refusal Names The Task That Caused It
placement: phase 9
severity: behaviour

**Problem**: One task carrying a C0 control byte kills `tick list`, `tick ready` and `tick blocked` with `cannot encode section tasks as TOON: toon: unsupported control character U+001B in string`. The library error names no row and the wrapper adds only the section name (`internal/cli/toon_formatter.go:342-349`, `:53-67`), so the agent is told a project-wide command failed and given nothing to act on: its routes are bisecting the project or reading `.tick/tasks.jsonl` directly — the §1 failure this work exists to delete, reached from the other side. The phase's own `fieldRefusal` (`internal/cli/toon_formatter.go:277-286`) already hunts the culprit field by re-marshalling each one alone, precisely "because the library error names none"; section encoding was left without the same rule, so a single-task document names its subject and a many-task document names nothing. All three commands reach the formatter through one call site (`internal/cli/list.go:228`), and the present diagnostic is pinned at `internal/cli/toon_refusal_test.go:121-129`, where the whole assertion for a two-task project is that stderr contains `"tasks"` — the section name the caller already knew.

**Solution**: Apply the rule the phase already established to the section path: when a section refuses, identify the offending row the way `fieldRefusal` identifies the offending field — by re-marshalling rows individually on the error path only — and name that task in the diagnostic, so `tick list` reports which task it could not encode. Where a row cannot be attributed the section name alone stands, as today. Settled rather than staged: §1's bar is output an agent can read without a rule learned outside it, and an error that forces bisection is the same defect as a document that lies, one step removed; the precedent is this phase's own field-level hunt, and the cost is an error-path-only re-marshal that never runs on a healthy project. §3.1's paragraph currently reads "and the task where the document covers one" — a sentence written to describe the code as built rather than the bar, which this task moves. The corrigendum extending it is the orchestrator's record, not a file this task touches.

**Outcome**: Every TOON refusal names something the agent can act on — the field and its task for a single-task document, the section and the offending task for a task list — so no refusal leaves bisection or reading the store as the only route forward.

**Do**:
1. In `internal/cli/toon_formatter.go`, add a section-level culprit hunt beside `fieldRefusal` (`:277-286`), reachable only from the section encoder's error branch: re-marshal each row alone (`toon.MarshalString(toon.NewObject(toon.Field{Key: name, Value: []T{row}}))`) and return a `*toonEncodeError` that keeps `part: "section " + name` and sets `taskID` to the identity of the first row that fails on its own; when no single row fails, return today's section-only error unchanged.
2. Keep `encodeToonSection(name, rows)`'s signature and message exactly as they are and have it delegate to an identity-taking variant with no identity function, so the eight existing call sites (`grep -n 'encodeToonSection(' internal/cli/toon_formatter.go` → `:67`, `:89`, `:94`, `:130`, `:158`, `:225`, `:308`, `:325`) and the unit test at `internal/cli/toon_formatter_test.go:857-868` compile and behave unchanged.
3. Point `FormatTaskList` (`internal/cli/toon_formatter.go:52-68`) at the identity-taking variant, supplying the row's ID (`func(r toonTaskRow) string { return r.ID }`); `list`, `ready` and `blocked` all inherit it through the single call site at `internal/cli/list.go:228`.
4. Carry the offending row's identity in `toonEncodeError.taskID`, never folded into `part`, so `refusalForTask` (`internal/cli/toon_formatter.go:368-375`) stays the overriding rule — a document covering a single task still names its own subject, and the cascade diagnostic still names the parent rather than the cascaded child.
5. Extend `internal/cli/toon_refusal_test.go`: the list case at `:121-129` asserts the offending task's ID alongside `"tasks"` and asserts the encodable sibling's ID is absent, and the `list` row of `refusedDocumentCommands()` (`:58`) expects the ID as well as the section name.

**Acceptance Criteria**:
- [ ] `tick list` over a project where one task's title carries `\x1b` exits 1, writes nothing to stdout, and its stderr names both the `tasks` section and that task's ID; `tick ready` and `tick blocked` behave identically through the shared call site.
- [ ] The diagnostic names the offending task alone — the ID of an encodable task in the same list never appears in stderr.
- [ ] Two refused tasks in one list name the first in row order, matching `fieldRefusal`'s first-failure rule.
- [ ] Where the section refuses but no single row refuses on its own, the message is byte-identical to today's `cannot encode section tasks as TOON: …`.
- [ ] The per-row re-marshal is reachable only from the section encoder's error branch: a list of encodable tasks marshals the section exactly once, as today.
- [ ] Every other refusal diagnostic is unchanged — `show`, `dep tree`, `note add`, `note remove`, `start`, `done`, `cancel` and `reopen` still name the section or field plus the document's subject, and `internal/cli/toon_refusal_test.go:170-181` still passes with the parent named and the cascaded child absent.
- [ ] An empty task list and `list --quiet` are untouched: neither reaches the section encoder (`internal/cli/toon_formatter.go:54-56`, `internal/cli/list.go:221-226`).
- [ ] `go test ./...`, `go vet ./...` and `golangci-lint run ./...` are clean, and `gofmt` has been applied.

**Tests**:
- `"it names the offending task when one task in the list cannot be encoded"`
- `"it names only the offending task when an encodable task sits beside it"`
- `"it names the first offending task when two tasks in the list carry refused values"`
- `"it falls back to the section name when the section refuses but no single row does"`
- `"it names the offending task for ready and blocked as well as list"`
- The existing refusal suite stays green unchanged, in particular `"it names the document's subject when the refused value sits on a cascaded task"` (`internal/cli/toon_refusal_test.go:170-181`) and `"it returns the encoder's error from encodeToonSection"` (`internal/cli/toon_formatter_test.go:857-868`).

## Task 2: Corrections
placement: phase 9
severity: corrections

**Problem**: `baseFormatter.FormatCascadeTransition` and `FormatDepTree` (`internal/cli/format.go:280-284`) return `("", nil)` — the successful-empty-document shape this phase deleted everywhere else. Both embedders override both (`internal/cli/toon_formatter.go:17`, `internal/cli/pretty_formatter.go:51`), so the stubs are dead today; a formatter added later that embeds `baseFormatter` and forgets an override inherits one, satisfies the `Formatter` interface at compile time, and makes `printDocument` (`internal/cli/helpers.go:36-42`) write a bare newline at exit 0 — a command that reports success and says nothing. The bank entry recorded for this cited two test assertions as pinning the stubs; those line numbers are `FormatRemoval` subtests in the final state, so the remedy is code-only.

**Solution**: Delete both stubs. Behaviour-preserving — every embedder already overrides them — and it makes `baseFormatter`'s existing doc comment accurate, since the type then carries only the methods it genuinely shares.

**Outcome**: `baseFormatter` provides only `FormatDepChange` and `FormatRemoval`, so a formatter that embeds it and omits `FormatCascadeTransition` or `FormatDepTree` fails to compile at its `var _ Formatter` assertion rather than inheriting a command that exits 0 printing a bare newline.

**Do**:
1. Delete the two stub methods and their doc comments from `internal/cli/format.go:280-284`, leaving `FormatDepChange` (`:272-278`) and `FormatRemoval` (`:286-298`) in place.
2. Leave `baseFormatter`'s type doc at `internal/cli/format.go:268-269` exactly as it stands — it already names `FormatDepChange` and `FormatRemoval` alone, and the deletion is what makes it true.
3. Edit no test file. `internal/cli/base_formatter_test.go` asserts on `FormatDepChange` (`:5-23`) and `FormatRemoval` (`:25-77`) only — the two line numbers the bank entry cited, `:40` and `:54`, are the `"it formats multiple task removal"` and `"it formats removal with dependency updates"` subtests — so no assertion pins either stub.
4. Confirm the two embedders still satisfy `Formatter` on their own methods: `ToonFormatter` (`internal/cli/toon_formatter.go:17`, overrides at `:162` and `:177`) and `PrettyFormatter` (`internal/cli/pretty_formatter.go:51`, overrides at `:289` and `:369`). `StubFormatter` (`internal/cli/format.go:302`, own methods at `:326` and `:329`) and `JSONFormatter` do not embed `baseFormatter` and are unaffected.
5. Run `go build ./... && go test ./... && go vet ./...`, apply `gofmt`, and run `golangci-lint run ./...`.

**Acceptance Criteria**:
- [ ] `internal/cli/format.go` declares no `FormatCascadeTransition` or `FormatDepTree` on `baseFormatter`.
- [ ] `go build ./...` succeeds: the four `var _ Formatter` assertions (`toon_formatter.go:21`, `pretty_formatter.go:55`, `json_formatter.go:15`, `format.go:305`) and the test-side assertions at `format_test.go:355-364` still hold.
- [ ] `go test ./...` passes with no test file changed; the whole diff is confined to `internal/cli/format.go`.
- [ ] Behaviour is unchanged: `tick dep tree` and the cascading commands (`start`, `done`, `cancel`, `reopen`) produce byte-identical output under `--toon`, `--pretty` and `--json`.
- [ ] `go vet ./...`, `gofmt` and `golangci-lint run ./...` are clean.

**Tests**: A pure refactor — no new test and no change to any test's semantics; the existing suite is the check. `TestToonFormatterCascadeTransition` and `TestPrettyFormatterCascadeTransition` (`internal/cli/cascade_formatter_test.go:13`, `:66`), `TestAllFormattersCascadeEmptyArrays` (`:264`), `TestToonFormatDepTree` (`internal/cli/toon_formatter_test.go:896`), `TestPrettyFormatDepTree` (`internal/cli/pretty_formatter_test.go:745`) and `TestBaseFormatter`/`TestBaseFormatterFormatRemoval` (`internal/cli/base_formatter_test.go:5`, `:25`) all stay green unchanged. The guard the deletion installs is compile-time at the `var _ Formatter` assertions, not a runtime assertion.
