---
id: migration-3-2
phase: 3
status: completed
created: 2026-02-15
---

# Surface beads provider parse/validation errors as failed results instead of silently dropping

**Problem**: The beads provider's `Tasks()` method in `internal/migrate/beads/beads.go:90-107` silently skips malformed JSON lines, empty-title entries, and validation failures. These skipped entries never reach the engine and never appear in the output. A user with 100 entries and 20 malformed ones sees "Done: 80 imported, 0 failed" with no indication that 20 entries were dropped. The spec requires "Continue on error, report failures at end."

**Solution**: Return entries with empty titles and invalid field values from the provider so the engine's validation can catch them and report them as failed Results with explanations in the output. Truly unparseable JSON lines (where no MigratedTask can be constructed) should be returned as sentinel MigratedTask values with a title indicating the parse error so they fail validation visibly.

**Outcome**: All entries in the JSONL file are accounted for in the migration output -- either as successful imports or as visible failures with explanations.

**Do**:
1. In `internal/migrate/beads/beads.go`, in the `Tasks()` method:
   - For malformed JSON lines (line 91-93): create a `MigratedTask` with a descriptive title like `"(malformed entry)"` and empty other fields. The engine's validation will reject it for the empty-title check or it will appear as a named failure. Alternatively, collect these as MigratedTask entries with a title describing the line number/error so the user sees them.
   - For entries with empty titles (line 96-99): remove the empty-title skip. Return the `MigratedTask` as-is with the empty title. The engine already handles empty-title validation at `engine.go:67-72` and produces a "(untitled)" result with the validation error.
   - For validation failures (line 102-104): remove the `Validate()` call from the provider. The engine already calls `Validate()` at `engine.go:67`. Double-validation in the provider pre-filters entries the engine should report.
2. Update beads provider tests to reflect that invalid entries are now returned rather than skipped.
3. Verify CLI integration tests show failures for invalid entries.

**Acceptance Criteria**:
- Empty-title entries from the JSONL file appear as failed results in migration output with a validation error message.
- Entries that fail validation (e.g., out-of-range priority) appear as failed results rather than being silently dropped.
- Malformed JSON lines produce visible failures in the output.
- Valid entries continue to import successfully.

**Tests**:
- Test beads provider with a JSONL file containing a mix of valid entries, empty-title entries, and malformed JSON lines; assert all are returned from `Tasks()`.
- Integration test: run migration against a fixture with invalid entries; assert the output shows the correct failed count and failure detail lines.
