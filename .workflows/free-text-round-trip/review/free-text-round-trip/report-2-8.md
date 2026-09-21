TASK: free-text-round-trip-2-8 (tick-8fcf09) — Read The Auto Flag From The Transition The Domain Applied

ACCEPTANCE CRITERIA:
- `task.TransitionResult` carries `Auto`, and `applyWithCascades` is the only place that sets it, from the same `auto` value it writes onto the primary target's `TransitionRecord`.
- `buildCascadeResult` takes five parameters and no caller passes a boolean literal for the primary's auto-ness.
- `autoCompleteParentIfTerminal` returns the `TransitionResult` the domain produced, and no production code builds a `task.TransitionResult` literal carrying statuses.
- Rendered auto still matches persisted auto on every existing path.
- No output changes anywhere — toon, JSON and pretty bytes are what they are today.
- `go build ./...`, `go vet ./...`, `go test ./...` clean and `gofmt -l internal cmd` empty.

STATUS: complete

SPEC CONTEXT: §7.2 of the specification replaces the arrow-and-`(auto)` cascade lines with a `changed[n]{id,title,from,to,auto}` table, and justifies the `auto` column on the grounds that it is not new vocabulary: "Every task's transition history already records each change with an `auto` flag — false when a user or agent asked for it, true when the system produced it as a consequence" (specification.md:281). The task closes the gap between that justification and the code — the CLI was re-asserting the flag with literals rather than reading the one the domain recorded.

IMPLEMENTATION:
- Status: Implemented (commit c4b21fc6, 9 files, +24/-24)
- Location:
  - internal/task/transition.go:5-12 — `Auto bool` added to `TransitionResult`, documented as the flag the same transition's `TransitionRecord` carries.
  - internal/task/apply_cascades.go:58 — `result.Auto = auto`, and the appended `TransitionRecord` at :59-64 now takes `Auto: result.Auto`, so the returned result and the persisted record read from one field.
  - internal/cli/transition.go:64 — `buildCascadeResult(id, title string, result task.TransitionResult, cascades []task.CascadeChange, tasks []task.Task)` — five parameters; :70 sets `PrimaryAuto: result.Auto`. The doc comment at :62-63 no longer describes the dropped parameter.
  - internal/cli/update.go:153 — `autoCompleteParentIfTerminal` captures the `TransitionResult` from `sm.ApplySystemTransition` and stores it at :161; the `oldStatus` local and the hand-built literal are gone.
  - Call sites: `grep -rn 'buildCascadeResult(' internal/cli` → 13 hits — the definition (transition.go:64), four production calls (transition.go:43, create.go:244, update.go:309, update.go:368) and eight in tests. None passes a sixth argument.
- Notes:
  - Behaviour equivalence verified per site: transition.go:43 takes `r` from `sm.ApplyUserTransition` (Auto=false, formerly the `false` literal); create.go:244 and update.go:309 take `r` from `validateAndReopenParent`, which only reports `reopened=true` after `sm.ApplySystemTransition` (helpers.go:128, Auto=true, formerly `true`); update.go:368 takes `r3.result` from `sm.ApplySystemTransition` (Auto=true, formerly `true`). Every substitution preserves the prior value, so no rendered byte moves.
  - `task.TransitionResult{` in production `internal/cli` is four zero-value returns at helpers.go:125, :130, :134, :136 — the criterion's substance, at line numbers shifted from the plan's (115/120/124/126) by later handler-signature work. No production literal carries statuses.
  - `buildCascadeResult` (transition.go:65) is the only production construction of `CascadeResult`, and `PrimaryAuto` is written only there; `format.go:216` reads it, `format.go:224` stamps `Auto: true` on cascade rows, matching the domain's own invariant at apply_cascades.go:104.
  - `sm.Transition`/`task.Transition` left as they were, per the plan. Confirmed no caller outside `internal/task`: `grep -rn '\.Transition(' --include='*.go' .` excluding `internal/task/` returns nothing, so the zero `Auto` those return never reaches the CLI.
  - The rule-3 substitution is safe against cascades re-moving the primary target: `applyWithCascades` pre-seeds the target in its seen-map (apply_cascades.go:76), so the parent's post-call `Status` still equals `result.NewStatus` — the value the dropped literal computed.

TESTS:
- Status: Adequate
- Coverage:
  - internal/cli/status_change_test.go:42-43 — the `Auto: true` fixture (`autoPrimaryResult`) replaces the dropped `true` literal for "it marks every row auto true when the primary was system-initiated" (:71-81); the auto-false and cascade-auto-true cases (:48-69) are untouched.
  - Rendered auto pinned end to end and unmodified by this commit: transition_test.go:685, :702, :705 (user row auto=false, cascade auto=true, toon and JSON agreeing row for row at :757); create_test.go:1402, :1438, :1441, :1468 (parent reopen, auto=true); update_test.go:1355, :1358, :1387 (re-parent onto a done parent plus rule-3 auto-completion, both auto=true).
  - Persisted auto pinned by internal/task: apply_cascades_test.go:302-305 and :458-461 (primary false, cascade true), create_test.go:1368 and update_test.go:1317 (persisted `Transitions[0].Auto == true` for the system paths).
  - Because the record is now written from `result.Auto` (apply_cascades.go:63), rendered and persisted cannot diverge by construction; the existing assertions on both sides would catch a wrong value at its single source.
- Notes: Exactly the mechanical conversion the plan specified — eight test call sites lose the boolean, one fixture line added, no assertion changed and no test name added. No redundancy introduced; nothing tests an implementation detail that the refactor removed.

CODE QUALITY:
- Project conventions: Followed — domain logic stays in `internal/task`, the CLI reads it; no new exported surface beyond the struct field; stdlib-only tests with "it does X" subtest naming.
- SOLID principles: Good — the change removes a duplicated fact (auto-ness asserted in two layers) and leaves one owner for it.
- Complexity: Low — parameter dropped, one local eliminated, no new branching.
- Modern idioms: Yes.
- Readability: Good — `rule3Result.result` now holds what the domain returned rather than a reconstruction, which is shorter and harder to get wrong.
- Comment accuracy: Holds. The `TransitionResult` doc (transition.go:5-7) describes the flag correctly for the path that sets it; `applyWithCascades`'s doc (apply_cascades.go:24-35) still accurately describes Auto on the primary and cascade records; `buildCascadeResult`'s doc no longer mentions the removed parameter.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go build ./...`, `go vet ./...`, `go test ./...` clean and `gofmt -l internal cmd` empty." — requires executing the build, vet, test and gofmt commands; reading shows a compile-consistent change (every `buildCascadeResult` call site converted, no stale references to the dropped parameter) but cannot confirm the toolchain is clean.
