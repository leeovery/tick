# Consolidation Findings: oversized-text-field-bricks-the-store (Phase 3)

## Findings

None. The one bank entry (route the migrate priority check through `task.ValidatePriority`) was dropped: the duplicated priority range and message in `internal/migrate/migrate.go:22-25` and `:50-52` date from 172262a2 (2026-02-15). This phase did not create or change them, so the duplication is pre-existing code the phase sits next to.

## Comment Corrections

- internal/migrate/migrate.go:40-42 — the second sentence repeats the four checks in the body directly below it, including the priority range that the constants already hold. Both migrate tasks in this phase (3-3, 3-4) had to rewrite it, so it goes stale whenever a check is added. The first line (`// Validate checks that a MigratedTask satisfies tick's constraints.`) stays.
  OLD:
  // It returns an error if the title breaks the CLI's title rules, the status
  // is unrecognized, the priority is outside the 0-4 range, or the description
  // exceeds the description cap.
  NEW:

- internal/task/task.go:202-204 — the second sentence repeats the error text built by the `fmt.Errorf` two lines below it. It says what the message contains, not a contract the code cannot state. The first sentence carries the after-trimming contract and stays.
  OLD:
  // ValidateDescription checks that a description, after trimming, is at most
  // 50,000 characters. The refusal carries the counted length and states that
  // nothing was saved.
  NEW:
  // ValidateDescription checks that a description, after trimming, is at most
  // 50,000 characters.
