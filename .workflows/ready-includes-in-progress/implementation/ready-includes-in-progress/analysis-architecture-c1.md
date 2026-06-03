AGENT: architecture
STATUS: clean
FINDINGS_COUNT: 0
CYCLE: 1

FINDINGS: none

SUMMARY: Architecture is sound — clean seams, strong composition (blocked derived from ready), no unjustified public surface, and thorough end-to-end coverage of the cross-task seams.

ANALYSIS DETAIL:
- Seam 1 — widened status gate (query_helpers.go): BlockedConditions() derives its EXISTS clauses from the ReadyNo*() helpers via negateNotExists (logical-inverse composition, not a parallel reimplementation that could drift). The widened gate is picked up automatically by ReadyConditions(), BlockedConditions(), and ReadyWhereClause() — no caller had to change. Strong composition; the spec's "untouched inverse machinery" claim holds.
- Seam 2 — resume-first ORDER BY (list.go:311-319): correctly keyed on f.Ready alone, so it applies to tick ready and tick list --ready and to neither plain tick list nor --blocked. The `(t.status = 'in_progress') DESC` term is a no-op when zero in_progress rows exist (byte-identical to the neutral clause), and this no-regression property is directly tested (ready_test.go:427).
- Seam 3 — stats blocked arithmetic (stats.go:84-85): uses the spec's canonical (Open + InProgress) − Ready, reusing counts already gathered and adding no public query-helper surface. The spec explicitly rejected a direct BlockedWhereClause(); that decision was honored. Correctness rests on the partition invariant, guarded by the stats-vs-list consistency test (blocked_test.go:404) and the negative-prevention test (stats_test.go:115).

CONSIDERED BUT NOT FLAGGED: The `t.status IN ('open', 'in_progress')` literal is duplicated across ReadyConditions() (line 53) and BlockedConditions() (line 78), with the in_progress status string appearing a third time in the ORDER BY. This is the spec's deliberate "flip one shared literal per side" design and follows the package's pre-existing convention of inline SQL-fragment literals (the helpers never interpolate task.Status* constants). Drift is caught by the query_helpers_test equality assertions plus the partition/stats integration tests. Flagging it would re-litigate an accepted spec decision and a pre-existing pattern — out of scope.

INTEGRATION COVERAGE is thorough and end-to-end: ready/blocked partition assertion, list-vs-stats count consistency, list --ready float parity, Ready-exceeds-Open negative-prevention, --status composition both directions, and the in_progress-is-ready path in the full workflow test.
