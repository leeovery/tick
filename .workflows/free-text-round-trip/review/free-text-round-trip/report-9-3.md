TASK: free-text-round-trip-9-3 (tick-88e344) — Corrections: delete the dead `FormatCascadeTransition` and `FormatDepTree` stubs from `baseFormatter`

ACCEPTANCE CRITERIA:
1. `internal/cli/format.go` declares no `FormatCascadeTransition` or `FormatDepTree` on `baseFormatter`.
2. `go build ./...` succeeds: the four `var _ Formatter` assertions and the test-side assertions in `format_test.go` still hold.
3. `go test ./...` passes with no test file changed; the whole diff is confined to `internal/cli/format.go`.
4. Behaviour is unchanged: `tick dep tree` and the cascading commands (`start`, `done`, `cancel`, `reopen`) produce byte-identical output under `--toon`, `--pretty` and `--json`.
5. `go vet ./...`, `gofmt` and `golangci-lint run ./...` are clean.

STATUS: complete

SPEC CONTEXT: Specification §8 ("Structured Output on Empty Branches") removes the prose/empty answers from the machine formats and replaces them with the non-empty document emptied — the same fields with zero counts. §3 requires every command to emit a decodable document on every branch, the empty one included. A `("", nil)` formatter return is the one shape that defeats both: `printDocument` (internal/cli/helpers.go:36-42) writes the document with `fmt.Fprintln` and returns nil, so an empty string becomes a bare newline at exit 0 — success reporting nothing. This task removes the last two places that shape could be inherited by accident.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/format.go:300-325 (`baseFormatter` and its two remaining methods); deletion commit 716dc0eb, 1 file changed, 6 deletions.
- Notes:
  - `baseFormatter` now declares exactly two methods: `FormatDepChange` (internal/cli/format.go:305) and `FormatRemoval` (internal/cli/format.go:315). A grep for `baseFormatter` across all `*.go` returns only the type/doc/method declarations, the two embeds, and test constructions — no `FormatCascadeTransition` or `FormatDepTree` on the type anywhere (criterion 1 met).
  - Interface satisfaction verified by enumerating every method set against the 8-method `Formatter` (internal/cli/format.go:276-298). `ToonFormatter` (internal/cli/toon_formatter.go:21) declares 6 of its own including the two overrides at :159 and :174, plus the two promoted from `baseFormatter`. `PrettyFormatter` (internal/cli/pretty_formatter.go:55) mirrors that, overrides at :286 and :366. `JSONFormatter` (internal/cli/json_formatter.go:15) declares all 8 itself (:28, :93, :189, :228, :261, :279, :317, :372) and embeds nothing. `StubFormatter` (internal/cli/format.go:331) declares all 8 itself (:334-:355). All four `var _ Formatter` assertions hold on reading (criterion 2 met).
  - Behaviour-preservation is settled by reading, not inference: Go resolves a method at the shallowest depth, and both embedders declare the two overrides at depth 0, so the deleted stubs were unreachable through the interface. No call site selects the embedded field explicitly — no `.baseFormatter.` selector exists in the tree. The deletion therefore cannot move a byte of `dep tree` or cascade output in any of the three formats (criterion 4 met).
  - The type doc at internal/cli/format.go:300-301 ("provides shared implementations of FormatDepChange and FormatRemoval for text-based formatters (Toon and Pretty)") was left untouched as the task directed, and the deletion is what makes it true. No stale comment remains.
  - The guard the task claims is installed is real: a new formatter embedding `baseFormatter` and omitting either method now fails its `var _ Formatter` assertion at compile time instead of inheriting a command that exits 0 printing a bare newline.

TESTS:
- Status: Adequate (pure refactor — correctly no new test)
- Coverage: The commit changed no test file; `git show --stat 716dc0eb` is `internal/cli/format.go | 6 ------`, so criterion 3's diff-confinement half is met exactly. `internal/cli/base_formatter_test.go` was re-read in full: `TestBaseFormatter` (:5) asserts on `FormatDepChange` only and `TestBaseFormatterFormatRemoval` (:25) on `FormatRemoval` only — four subtests, none of which constructs a `baseFormatter` and calls either deleted stub. The bank entry's claim that two assertions pinned the stubs is wrong on the final state, as the task concluded: :40 is `"it formats multiple task removal"` and :54 is `"it formats removal with dependency updates"`, both `FormatRemoval`. The compile-time surface is held by `TestFormatterInterfaceCompileCheck` (internal/cli/format_test.go:353) and `TestCascadeTypes` (:358), whose four `var _ Formatter` assertions sit at :360-:363 and which then calls `FormatCascadeTransition` on all four concrete formatters through the interface.
- Notes: Declining to add a test here is the right call — the deletion's guard is a compile error, which no runtime assertion can express. Behaviour of the two overridden methods is already covered by `TestToonFormatterCascadeTransition`/`TestPrettyFormatterCascadeTransition` (internal/cli/cascade_formatter_test.go:13, :66), `TestAllFormattersCascadeEmptyArrays` (:264), `TestToonFormatDepTree` (internal/cli/toon_formatter_test.go:896) and `TestPrettyFormatDepTree` (internal/cli/pretty_formatter_test.go:745), none of which needed changing.

CODE QUALITY:
- Project conventions: Followed. Dead code removed rather than left behind a comment; the shared-behaviour embed keeps only what Toon and Pretty genuinely share, matching the CLAUDE.md formatter-interface description.
- SOLID principles: Good. Interface segregation improved — `baseFormatter` no longer supplies a default that silently satisfies a contract it cannot honour, so the Liskov hazard (an embedder inheriting a substitutable-but-wrong implementation) is gone.
- Complexity: Low. Net -6 lines, no control flow touched.
- Modern idioms: Yes.
- Readability: Good. The type's doc and its method set now agree.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...` passes with no test file changed; the whole diff is confined to `internal/cli/format.go`." — The diff half is settled (commit 716dc0eb touches only internal/cli/format.go, no test file). The suite-pass half needs `go test ./...` executed; reading establishes only that no test references the deleted methods.
- "`go vet ./...`, `gofmt` and `golangci-lint run ./...` are clean." — Needs the three commands run. Reading shows the deletion removed the two methods with their doc comments and surrounding blank lines cleanly, leaving no stray whitespace, but tool cleanliness across the module is an execution result.
