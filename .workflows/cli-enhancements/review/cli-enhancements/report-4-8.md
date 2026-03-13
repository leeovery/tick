TASK: cli-enhancements-1-3 -- Integrate ResolveID into update and create ID-referencing flags

ACCEPTANCE CRITERIA:
- All commands accepting task IDs resolve through `ResolveID`: show, update, start, done, cancel, reopen, dep add/rm, remove, and ID-accepting flags (--parent, --blocked-by, --blocks)
- Edge case: partial ID resolving to self-reference in --parent or --blocked-by

STATUS: Complete

SPEC CONTEXT:
The specification states partial ID matching "applies everywhere an ID is accepted: positional args, --parent, --blocked-by, --blocks." This task specifically covers ID-referencing flags on create and update commands: --parent (create, update), --blocked-by (create), --blocks (create, update). The --blocked-by flag exists only on create (not update), and --blocks exists on both. The list --parent filter was addressed in Phase 6 (task 6-1) after an analysis finding.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:160-178` -- ResolveID called for --parent (line 162), --blocked-by (line 168), and --blocks (line 174)
  - `/Users/leeovery/Code/tick/internal/cli/update.go:208-225` -- ResolveID called for positional ID (line 209), --parent (line 214), and --blocks (line 221)
  - `/Users/leeovery/Code/tick/internal/cli/list.go:156-161` -- ResolveID called for --parent filter
- Notes: All ID-accepting flags correctly resolve partial IDs through `store.ResolveID()` before any mutation or validation logic. The resolution happens after the store is opened but before `store.Mutate()` is called, which is the correct ordering. The `--parent` empty-string guard on update (line 213: `*opts.parent != ""`) correctly skips resolution when --parent is used to clear the parent.

TESTS:
- Status: Adequate
- Coverage:
  - Partial ID in --parent on update: `partial_id_test.go:237` -- verifies parent resolves to full ID
  - Partial IDs in --blocks on update: `partial_id_test.go:261` -- verifies multiple comma-separated partial IDs in --blocks
  - Partial ID in --parent on create: `partial_id_test.go:289` -- verifies parent resolves to full ID
  - Partial IDs in --blocked-by on create: `partial_id_test.go:307` -- verifies multiple comma-separated partial IDs
  - Partial IDs in --blocks on create: `partial_id_test.go:329` -- verifies --blocks resolves and target task's blocked_by is updated
  - Self-reference edge case: `partial_id_test.go:355` -- partial --parent resolving to same task on update errors with "cannot be its own parent"
  - Ambiguous partial in --blocked-by: `partial_id_test.go:370` -- ambiguous prefix returns error listing matching IDs
  - Not-found partial in --parent: `partial_id_test.go:392` -- non-existent prefix returns "not found" error
- Notes: The edge case specifically called out in the plan (partial ID resolving to self-reference in --parent or --blocked-by) is tested. Test coverage is thorough without being redundant -- each flag type (--parent, --blocked-by, --blocks) has at least one happy path and the error paths (ambiguous, not-found, self-reference) are covered.

CODE QUALITY:
- Project conventions: Followed. Uses `store.ResolveID()` consistently, `fmt.Errorf` for error wrapping, pointer types for optional fields, `parseCommaSeparatedIDs` helper for comma-separated inputs.
- SOLID principles: Good. ResolveID is a single-responsibility method in the storage layer. The CLI handlers delegate to it cleanly. ID resolution is separated from validation and mutation.
- Complexity: Low. The resolution logic is linear: parse args, open store, resolve IDs, validate, mutate. No complex branching.
- Modern idioms: Yes. Idiomatic Go patterns throughout.
- Readability: Good. Clear comments at lines 160 and 208 ("Resolve partial IDs via store.ResolveID.") document the intent. The code flow is easy to follow.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
