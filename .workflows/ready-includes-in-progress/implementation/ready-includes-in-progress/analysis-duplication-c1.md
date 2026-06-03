AGENT: duplication
STATUS: findings
FINDINGS_COUNT: 1
CYCLE: 1

FINDINGS:

- FINDING: Status-gate literal duplicated across ReadyConditions and BlockedConditions
  - SEVERITY: medium
  - FILES: internal/cli/query_helpers.go:53, internal/cli/query_helpers.go:78
  - DESCRIPTION: The live-status gate `t.status IN ('open', 'in_progress')` is written as an independent string literal in two places — once in `ReadyConditions()` (line 53) and once in `BlockedConditions()` (line 78). The specification's governing invariant (`ready ⊎ blocked = all live tasks`) depends on these two gates being byte-identical: the spec states "ready and blocked share one identical status gate" and describes the whole change as "flip one shared literal per side." Yet the two literals are not derived from a single source — they are copy-paste twins that can silently drift (a future status addition, or a whitespace change to one IN-list, would break the partition with no compile error). The De Morgan inverse machinery around them (`negateNotExists`, the `ReadyNo*()` helpers) is already correctly shared, which makes this raw literal the lone un-deduplicated piece of the otherwise-shared gate.
  - RECOMMENDATION: Extract the gate into a single source of truth in `query_helpers.go` — e.g. a package-level `liveStatusGate` constant or a `LiveStatusGate() string` helper returning `t.status IN ('open', 'in_progress')` — and reference it from both `ReadyConditions()` and `BlockedConditions()`. This consolidates the two existing literals into one without adding behavior and structurally guarantees the "one shared status gate" the spec relies on. The three matching assertions in `query_helpers_test.go:41/60/115` can then assert against the same helper.

SUMMARY: One medium-impact duplication: the live-status gate literal is hand-copied into both `ReadyConditions()` and `BlockedConditions()` in `query_helpers.go`, defeating the spec's "single shared status gate" intent and leaving the partition invariant exposed to silent drift.

SCOPE NOTES (not findings): The implementation touched only three production files (`query_helpers.go`, `list.go`, `stats.go`); `list.go`'s conditional ORDER BY and `stats.go`'s arithmetic blocked-count derivation are each single-site and introduce no cross-file duplication. The per-command `run*` test harness helpers (`runReady`, `runBlocked`, `runStats`, `runList`) are near-identical but pre-date this implementation and follow an established project-wide convention — out of scope. The new ready/stats/blocked subtests share fixture-construction patterns but each asserts distinct semantics, below the extraction threshold.
