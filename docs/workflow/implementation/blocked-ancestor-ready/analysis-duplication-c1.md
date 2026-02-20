AGENT: duplication
FINDINGS:
- FINDING: BlockedConditions() duplicates SQL from ReadyNo*() helpers instead of composing from them
  SEVERITY: high
  FILES: internal/cli/query_helpers.go:64-94
  DESCRIPTION: BlockedConditions() contains three independently-written EXISTS subqueries (unclosed blockers at lines 69-73, open children at lines 74-77, blocked ancestor CTE at lines 79-91) whose SQL bodies are character-for-character identical to the NOT EXISTS subqueries already returned by ReadyNoUnclosedBlockers(), ReadyNoOpenChildren(), and ReadyNoBlockedAncestor(). Each pair shares the same inner SELECT, JOINs, and WHERE clause -- only the EXISTS vs NOT EXISTS wrapper differs. This is exactly the anti-pattern called out in code-quality.md under "Compose, Don't Duplicate": the blocked query should derive from the ready helpers rather than being independently authored SQL that can drift.
  RECOMMENDATION: Refactor each ReadyNo*() helper to return just the inner subquery body (or add companion helpers that do), then have both ReadyConditions() and BlockedConditions() compose from them -- wrapping in NOT EXISTS for ready, EXISTS for blocked. Alternatively, since BlockedConditions() is the De Morgan inverse of the ready non-status conditions, derive it mechanically: for each ReadyNo*() string, strip the "NOT EXISTS" prefix and replace with "EXISTS". This eliminates ~25 lines of duplicated SQL and guarantees the two stay in sync when the subqueries change.
- FINDING: runReady() and runBlocked() test helpers are near-identical
  SEVERITY: low
  FILES: internal/cli/ready_test.go:14-26, internal/cli/blocked_test.go:15-27
  DESCRIPTION: Both helpers construct an App with identical Stdout/Stderr/Getwd/IsTTY wiring, build a fullArgs slice, call app.Run, and return the same triple. The only difference is the command string ("ready" vs "blocked"). This is a 13-line block duplicated with a one-word change.
  RECOMMENDATION: Extract a shared helper like runCommand(t, dir, command string, args ...string) into the test helpers file. Both runReady and runBlocked become one-line delegations. This is a minor improvement -- the duplication is small and test-only -- but it would simplify any future command test files that follow the same pattern.
SUMMARY: BlockedConditions() contains three SQL subquery bodies copy-pasted from the ReadyNo*() helpers (high severity, violates Compose Don't Duplicate principle). A secondary low-severity finding is the near-identical runReady/runBlocked test helpers.
