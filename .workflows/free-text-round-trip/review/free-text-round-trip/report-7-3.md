TASK: free-text-round-trip-7-3 (tick-401dc3) — A Dependency Participant That Names No Task Says So

ACCEPTANCE CRITERIA:
- `tick dep tree --pretty` in a project whose only blocker ID matches no task draws `tick-ghost1   (missing)` with the blocked task beneath it and an unchanged summary line.
- `tick dep tree --json` returns `{"id":"tick-ghost1","title":"","status":"missing"}` for that participant, its child keeping its own title and status.
- No participant that resolves to a task record carries the marker — a cycle's members render their real statuses.
- The toon edge list is byte-identical: it carries IDs only.
- `missing` is spelled once in production code and collides with none of the four task statuses.
- `go test ./...` green, with only the two dangling-blocker assertions changed.

STATUS: complete

SPEC CONTEXT: §8 ("Structured Output on Empty Branches") carries the paragraph "A participant that names no task says so" (specification.md:330): widening full-graph coverage puts a dangling blocker into the node-shaped renderings, where it renders with status `missing` and an empty title — pretty `tick-ghost   (missing)`, JSON `{"id":"tick-ghost","title":"","status":"missing"}` — the marker occupying the existing status slot so §4.1's no-new-visual-form bound holds, colliding with none of the four real statuses, and the toon edge list unaffected because it carries IDs only. The corrigendum of 2026-09-20 (specification.md:555) records the same decision and its derivation (dropping non-task participants would take the blocked real task out of pretty's tree with the ghost that seeds its walk; blank status would make an agent learn out-of-band that empty means "no such task"). The corrigendum's reference to `buildUnrootedTrees` was accurate when written (17:28) — the rename to `buildSeededTrees` landed later the same day in task 7-2 (777f67e3, 18:50), so the record is a dated statement of prior state, not a false claim.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/dep_tree_graph.go:237-238 (`// depTreeMissingStatus is the status carried by a dependency participant no task record matches.` / `const depTreeMissingStatus = "missing"`); internal/cli/dep_tree_graph.go:248-254 (seed builds `DepTreeTask{ID: id, Status: depTreeMissingStatus}` then replaces the whole struct with `toDepTreeTask(t)` when `taskIdx[id]` exists). Commit 7d46220f — 5 lines of production change, exactly the plan's Do list.
- Notes: The marker is confined to the seed by construction. `walkDownstream` (dep_tree_graph.go:53-58) and `walkUpstream` (dep_tree_graph.go:88-92) both `continue`/return on an ID absent from `taskIdx`, so no other builder path can emit a node without a record; the seed at :250 is the only `DepTreeTask` literal in production apart from `toDepTreeTask` (dep_tree_graph.go:33). `BuildFocusedDepTree` (dep_tree_graph.go:336-363) builds its target from a record it has already resolved and its trees from the two walks, so the marker cannot reach the focused view — consistent with §8, which scopes this to the full graph's node renderings.
- Consumption of the marker is inert: `writeDepTreeTaskLine` (internal/cli/pretty_formatter.go:421-424) formats the status without colouring or validating it, and `toJSONDepTreeNodes` (internal/cli/json_formatter.go:357-363) copies it through. Nothing in production parses a status back out of dep-tree output (`validStatuses` in internal/migrate/migrate.go:13 guards migration input only), so no enum is broken by a fifth value.
- The toon full-graph path reads only `result.Edges` (internal/cli/toon_formatter.go:182-193) and never touches `Task.Status`, so the edge list cannot move with this change.

TESTS:
- Status: Adequate
- Coverage: Builder level — internal/cli/dep_tree_graph_test.go:294 pins the seeded node as `DepTreeTask{ID: "tick-ghost1", Title: "", Status: "missing"}` with the real child intact; :395 walks every node of a two-task cycle and fails on any status that is not the record's own, which is the guard that the marker stays confined. CLI level — internal/cli/dep_tree_test.go:628 pins the JSON object (`"tick-ghost1", "", "missing"`) with the child keeping `Task A`/`open`, and :675 pins the whole pretty document byte-for-byte including the unchanged summary `1 chain, longest: 1, 1 blocked`. internal/cli/dep_tree_test.go:381 (the toon edge list) is unedited and still asserts a single `tick-ghost1 -> tick-aaa111` row with IDs only. The two-level chain variants (dep_tree_test.go:638, :696 and dep_tree_graph_test.go:315) carry `missing` consistently.
- Notes: Each assertion would fail if the marker were dropped or misapplied — removing it breaks three, widening it to recorded tasks breaks the cycle guard. Builder-level and CLI-level coverage of the same fixture is not redundant: one pins the struct, the other pins the two rendered documents that §8 names. No stale goldens survive — no test in internal/ still pins `   ()` or an empty dep-tree `status`.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` "it does X" subtest names, fixtures reused from the existing `danglingBlockerTasks`/`cycleTasks` helpers, no testify.
- SOLID principles: Good — the change is one constant and one field on an existing literal; no new type, branch or indirection.
- Complexity: Low — no branch added; the existing `if t, exists := taskIdx[id]; exists` upgrade already carried the distinction.
- Modern idioms: Yes.
- Readability: Good — the marker is named at its single declaration and the comment states the condition under which it survives.
- Issues: None. The const comment holds against the code, names no task id or spec section, and does not restate the literal.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...` green, with only the two dangling-blocker assertions changed." — the second half is settled by the diff of 7d46220f: exactly two assertions changed (`assertJSONDepTreeTask(..., "")` → `"missing"` at dep_tree_test.go:634 and the pretty golden `"tick-ghost1   ()\n"` → `"tick-ghost1   (missing)\n"` at :684), plus two subtest renames and the new builder subtests. The green run itself needs an executing pass over `go test ./...`.
