TASK: free-text-round-trip-2-1 (tick-0ca6ad) — Collapse A Command Status Movements Into One Changed Set

ACCEPTANCE CRITERIA:
- `CascadeResult.Changed` lists the requested change first, then cascaded changes in the order the state machine produced them
- The row for a task the caller named through `start`/`done`/`cancel`/`reopen` carries `Auto: false`; every other row carries `Auto: true`
- Every row of a block built with `primaryAuto: true` carries `Auto: true`
- A task appearing in more than one block collapses to one row whose `From` is its status before the command and whose `To` is its status after
- A task whose merged `From` equals its merged `To` produces no row
- `rows()` and `mergeStatusChanges()` return a non-nil empty slice, never nil, when no row survives
- `mergeStatusChanges` over a single block returns that block's `Changed` unchanged
- `RunTransition` builds a `CascadeResult` whether or not cascades fired
- Toon, pretty and JSON output for every command is byte-identical to before this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §7.2 requires one `changed[n]{id,title,from,to,auto}` table per status-moving command, in which a task appears at most once, reads from the status it held before the command to the status it holds after, and carries no row at all when it ends where it started. The `auto` column must reuse the transition history's existing vocabulary (false when a user or agent asked for the change, true when the system produced it) rather than invent a "requested" column. §7.5 records why `create`/`update` produce cascades at all (Rule 6 reopen of a done parent; Rule 3 auto-complete of the vacated parent), which is the case where two blocks can meet on a shared ancestor. §7.3/§7.4 place the table as a standalone document for the four status commands and as a section inside the record for `create`/`update`.

IMPLEMENTATION:
- Status: Implemented (shape refined by later tasks in the same phase)
- Location:
  - `internal/cli/format.go:186-195` — `StatusChange{ID, Title, From, To string; Auto bool}` beside `CascadeEntry`
  - `internal/cli/format.go:197-228` — `CascadeResult.PrimaryAuto` and `Changed()`, which seeds the set with the primary then each cascade entry with `Auto: true`
  - `internal/cli/format.go:142-151` — `StatusChanges.Blocks` / `Rows()`, the multi-block carrier for `create`/`update`
  - `internal/cli/transition.go:109-147` — `statusChangeSet` with `add`/`rows`
  - `internal/cli/transition.go:149-160` — `mergeStatusChanges`
  - `internal/cli/transition.go:33-58` — `RunTransition` builds the result unconditionally (the `if len(c) > 0` guard is gone) and dereferences it at line 58
  - Call sites: `internal/cli/transition.go:43`, `internal/cli/create.go:244`, `internal/cli/update.go:309`, `internal/cli/update.go:368`
- Notes:
  - Deliberate, sound divergence from the task's `Do` steps, and it belongs to later tasks of the same plan, not to drift: `Changed` is now a method on `CascadeResult` rather than a stored field (commit 8e579a68, task 2-7), and the `primaryAuto bool` parameter was replaced by `CascadeResult.PrimaryAuto` sourced from `task.TransitionResult.Auto` (commit c4b21fc6, task 2-8). The task's own Context section names that source as the right one: `RunTransition` calls `ApplyUserTransition` (`internal/task/apply_cascades.go:12-14`, auto=false) while `validateAndReopenParent` (`internal/cli/helpers.go:123`) and `autoCompleteParentIfTerminal` (`internal/cli/update.go:155`) call `ApplySystemTransition` (auto=true), so every `create`/`update` block carries `PrimaryAuto: true` without the caller having to assert it. Substance of every criterion is preserved; the flag can no longer disagree with the transition record it mirrors.
  - Merge semantics match §7.2 exactly: `add` keys on `task.NormalizeID` (`transition.go:119`), keeps the stored `From`, overwrites `To`, keeps the first non-empty `Title`, and takes `stored.Auto && incoming.Auto` so a change any block marks as requested stays requested. `rows()` walks first-seen order and drops `From == To`.
  - Blocks are appended in the order the mutations happen inside the `Mutate` closure (Rule 6 at `update.go:309`, before the field-update loop; Rule 3 at `update.go:368`, after it), so a shared ancestor's merged row reads from its pre-command status.
  - Byte-identity at this task's commit (4a946d19) holds by construction: `outputTransitionOrCascade` still branched on `len(cr.Cascaded) == 0` to `FormatTransition` at that commit, and the `create`/`update` edits were the `, true` argument only. In the delivered end state the toon and JSON output has deliberately changed — that is what tasks 2-2 onward exist to do — while pretty is unchanged (`internal/cli/pretty_formatter.go:293-302` still renders `id: old → new` from `TaskID`/`OldStatus`/`NewStatus` and the `ParentID`-linked tree from `Cascaded`, all of which this task left untouched, per §4.1).
  - `*cascadeResult` at `transition.go:58` cannot be nil: `Store.Mutate` (`internal/storage/store.go:187`) always invokes the closure, and the closure either assigns the result or returns the not-found error.
  - `create.go:244` passes `opts.parent` after `store.ResolveID` (`internal/cli/create.go:165-170`), so no partial ID reaches a row's `ID`.

TESTS:
- Status: Adequate
- Coverage: `internal/cli/status_change_test.go:30-183` carries all ten named micro-acceptance cases — requested change auto=false (48), cascades auto=true (60), system-initiated block all auto (72), no-cascade block yields one row (82), collapse across blocks (93), no-op dropped (105), first-seen order across blocks (116), first non-empty title kept (142), empty non-nil slice for both `mergeStatusChanges()` and a no-op block (156), single block unchanged (174). The eleventh ("it leaves transition output unchanged") is carried end-to-end by the existing suite: `internal/cli/readme_samples_test.go:374-378` runs the real CLI for `start`/`done` in toon, pretty and JSON and `readme_samples_test.go:177-192` asserts the `auto` column is false only on the requested row, which also exercises `RunTransition`'s no-cascade path producing `changed[1]` (`readme_samples_test.go:28`).
- Notes: No redundancy — each subtest fixes one behaviour and would fail if that behaviour broke (drop the `stored.Title == ""` guard and 142 fails; drop the `From != To` filter and 105 fails; use a map walk instead of `order` and 116 fails). Setup is the minimum two-task fixture plus a small block builder (`statusChangeBlock`, line 11); no mocking. Stdlib `testing` with "it does X" subtests, matching the project's convention.

CODE QUALITY:
- Project conventions: Followed — unexported accumulator kept in the package, `task.NormalizeID` used for every ID comparison, doc comments on each exported and unexported declaration, stdlib-only tests with `t.Run` subtests.
- SOLID principles: Good — `statusChangeSet` does one thing (merge rows); `CascadeResult` keeps its rendering fields and derives the flat view rather than storing a second copy that could drift.
- Complexity: Low — `add` is one branch, `rows` one loop with one filter, `mergeStatusChanges` two loops.
- Modern idioms: Yes — lazy map init on first insert, `make([]T, 0, n)` for the guaranteed non-nil empty slice, value-type rows so blocks survive the `Mutate` closure.
- Readability: Good — the merge rule is stated in the doc comment at `transition.go:116-117` and reads off the four lines that implement it.
- Issues: None. Comments in the changed code hold: `transition.go:137-138` ("Never nil, so JSON renders [] rather than null") is true of the consumer at `internal/cli/json_formatter.go:301-309`; `transition.go:150-151` ("Rows hold only copied values") is true — `StatusChange` and `CascadeEntry` hold only strings and a bool; `format.go:197-198` on `PrimaryAuto` matches `transition.go:70`. No process-artifact references survive in the code.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — needs the three commands run over the repository; reading confirms the call sites all match the current five-argument `buildCascadeResult` signature and that no orphaned reference to a `Changed` field remains, but a clean suite, vet and format pass can only be observed by executing them.
