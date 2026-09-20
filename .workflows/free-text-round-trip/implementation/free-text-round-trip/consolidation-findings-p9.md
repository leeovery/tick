# Consolidation Findings: free-text-round-trip (Phase 9)

## Findings

### F1: baseFormatter's two inherited stubs still return a successful empty document
- **Class**: dead-code
- **Failure**: `baseFormatter` promotes `FormatCascadeTransition` and `FormatDepTree` implementations that return `("", nil)` — the successful-empty-document shape this phase deleted everywhere else (`encodeToonSection`'s `name[0]:` fallback, `encodeToonFields`' `""` return). Both current embedders override both methods, so nothing calls the stubs today. A formatter added later that embeds `baseFormatter` and forgets one override still satisfies `Formatter` at its `var _ Formatter = (*XFormatter)(nil)` line, and the promoted stub feeds `printDocument` a document of `""` with a nil error: `tick dep tree` writes a bare newline and exits 0. The agent reads a command that succeeded and reported no dependencies. Nothing notices — the error channel the phase just added says the document is correct. Deleting the stubs turns that omission into a compile failure at the interface assertion instead, which is the only place it can be caught.
- **Evidence**:
  - `internal/cli/format.go:280-284` — the two stubs and their doc comments
  - `internal/cli/format.go:268-270` — `baseFormatter`'s doc already names only `FormatDepChange` and `FormatRemoval` as what it provides
  - `internal/cli/toon_formatter.go:17` embeds it; `internal/cli/toon_formatter.go:162` and `:177` override both
  - `internal/cli/pretty_formatter.go:51` embeds it; `internal/cli/pretty_formatter.go:289` and `:369` override both
  - `internal/cli/helpers.go:36-42` — `printDocument` prints a bare newline and returns nil for `("", nil)`
  - `internal/cli/base_formatter_test.go` carries no assertion on either stub (it tests `FormatDepChange` and `FormatRemoval` only), so nothing needs deleting on the test side
- **Proposed shape**: delete both stub methods from `baseFormatter` with their comments. `ToonFormatter` and `PrettyFormatter` compile unchanged; `baseFormatter` stops being a partial `Formatter` and its existing doc comment becomes accurate. Behaviour-preserving: no live call site reaches either stub.
- **Bank**: executor entry on Tfree-text-round-trip-9-1 ("baseFormatter's FormatCascadeTransition/FormatDepTree stubs are dead and now return `("", nil)`"). Confirmed against the final state for the code half. The entry's test half is stale — `base_formatter_test.go:40` and `:54` are `FormatRemoval` subtests, not assertions that the stubs return `""`; the remedy is code-only.

### F2: the task-list refusal names a section the agent already knows and no task it can act on
- **Class**: behaviour
- **Failure**: one task whose title carries a C0 control byte (§3.1: "an escape sequence inside pasted terminal output is the everyday way") kills `tick list`, `tick ready` and `tick blocked` for the whole project, and the diagnostic is `cannot encode section tasks as TOON: toon: unsupported control character U+001B in string`. The library error names no row and `encodeToonSection` adds only the section name, which for a task list is information the caller already had. The agent has no handle on which task to fix: its routes are re-running under `--json`/`--quiet` and bisecting, or reading `.tick/tasks.jsonl` — the exact abandonment of the CLI that §1 names as the failure that triggered this work. Within this same diff, `fieldRefusal` already refuses to accept that: it re-encodes each field singly to name the culprit *because* "the library error names none" (its own comment). Section encoding does not follow the rule the phase wrote next to it.
- **Evidence**:
  - `internal/cli/toon_formatter.go:342-349` — `encodeToonSection` wraps with `part: "section " + name` and nothing row-level
  - `internal/cli/toon_formatter.go:53-67` — `FormatTaskList` returns that error straight out; no `toonDoc`, no task identity available to `refusalForTask`
  - `internal/cli/toon_formatter.go:277-286` — `fieldRefusal`, the culprit-hunt already written for the named-field case
  - `internal/cli/list.go:228-229` — the single call site, shared by `list`, `ready` and `blocked`
  - `internal/cli/toon_refusal_test.go:121-129` — pins the current diagnostic: the whole assertion for a two-task project is that stderr contains `"tasks"`
  - `github.com/toon-format/toon-go@v0.0.0-20251202084852/internal/format/format.go:94-95` — the library message is `toon: unsupported control character U+%04X in string`, with no positional information
- **Proposed shape**: on a section refusal, identify the offending row the way `fieldRefusal` identifies the offending field — re-encode rows singly and name the first that fails — and carry that row's ID into the diagnostic where the row type has one, so `tick list` reports which task it could not encode. Cheapest shape that keeps `encodeToonSection` generic: an optional row-identity function supplied by the callers whose rows carry an ID (`tasks`, `changed`, `blocked_by`, `children`, `notes` by index), defaulting to today's section-only message.
- **Spec tension the orchestrator must rule on**: §3.1 as amended requires the diagnostic to name "the field or section it could not encode, and the task where the document covers one". The landed code satisfies that literally — a task list covers many tasks, so no task is named. This finding argues the multi-task document is where naming matters most and the phase's own `fieldRefusal` sets the precedent; accepting it likely needs a §3.1 corrigendum rather than a bare task.

