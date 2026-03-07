---
topic: cli-enhancements
cycle: 1
total_findings: 7
deduplicated_findings: 6
proposed_tasks: 5
---
# Analysis Report: cli-enhancements (Cycle 1)

## Summary
One high-severity bug found by two agents independently: `queryShowData` in show.go omits the `type` column from its SQL query, so all detail output (show, create, update, note) renders task type as empty regardless of actual value. Four medium-severity duplication and architecture findings round out the report: parallel dedup/validate logic in the domain layer, duplicated TOON section builders, repeated validation blocks across CLI handlers, and double lock acquisition in ResolveID.

## Discarded Findings
- Test runner helpers copy-pasted across test files (duplication, low) -- low severity, isolated to test convenience wrappers with no drift risk; does not cluster with other findings to form a pattern warranting a task.
