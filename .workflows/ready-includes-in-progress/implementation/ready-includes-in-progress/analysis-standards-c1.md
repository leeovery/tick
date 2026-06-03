AGENT: standards
STATUS: clean
FINDINGS_COUNT: 0
CYCLE: 1

FINDINGS: none

SUMMARY: Implementation fully conforms to the specification and project conventions; no spec drift or standards violations found.

VERIFIED:
- Status gate (query_helpers.go:53,78): both ReadyConditions() and BlockedConditions() use `t.status IN ('open', 'in_progress')`; the negateNotExists/inverse machinery is untouched and BlockedConditions still derives its OR clause from the ReadyNo*() helpers. ReadyWhereClause() picks up the new gate automatically for stats. Matches spec "Affected Code Surface".
- Conditional ORDER BY (list.go:311-319): keyed solely on f.Ready, prepends `(t.status = 'in_progress') DESC, t.priority ASC, t.created ASC`; else branch keeps the neutral ordering. Applies to tick ready and tick list --ready, to neither plain list nor --blocked, with no special-case guard for narrowed browses. Matches.
- Stats blocked derivation (stats.go:85): `(stats.Open + stats.InProgress) - stats.Ready` — canonical arithmetic route. Both inline comments match the spec's prescribed text verbatim including the previously-omitted "no blocked ancestor" clause. Matches.
- No-change surfaces: state machine, flag registry/commandFlags, formatters, and cache schema all untouched, as the spec requires.

Test conformance: all MUST-change tests updated; all MUST-ADD tests present (resume-first ordering with discriminating worse-priority in_progress fixture, partition test, Ready-exceeds-Open non-negative derivation, --status open/in_progress composition, list --ready float parity, --count 1 top-in-flight). Acceptance criteria #1-#10 each exercised.

Conventions: stdlib testing only, t.Run subtests, "it does X" naming, t.Helper() on helpers, fmt.Errorf("...: %w", err) wrapping all followed. gofmt, go vet, and affected suites all pass.
