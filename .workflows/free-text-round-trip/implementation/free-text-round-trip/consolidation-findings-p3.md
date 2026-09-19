# Consolidation Findings: free-text-round-trip (Phase 3)

## Findings

### F1: the superseded full-graph guard now asserts nothing and documents a stub that does not exist
- **Class**: dead-code
- **Failure**: A handler-level regression on the populated full-graph branch under pretty — `runFullDepTree` printing an empty string, or routing to the wrong formatter method — passes `internal/cli/dep_tree_test.go:273-294`, because the subtest's only assertion is that the output is not the string `"No dependencies found."`. Its premise is gone: task 3-4 deleted the handler's zero-roots branch, so "the handler should NOT take the no-dependencies path" tests a branch that no longer exists. `TestPrettyFormatDepTree` covers the formatter, not the handler's routing, so a routing regression on this branch is caught by nothing. Separately, a maintainer reading the comment is told `FormatDepTree` is a stub returning empty and acts on that belief.
- **Evidence**:
  - `internal/cli/dep_tree_test.go:273-294` — subtest `"it outputs dep tree for project with dependencies"`; the false comment is `internal/cli/dep_tree_test.go:288-290` (`// FormatDepTree is currently a stub returning empty, so output will be empty,`)
  - `internal/cli/dep_tree_test.go:252-271` — subtest `"it still renders the populated graph"`, added this phase, already asserts the same branch by decoded edges and counts
  - `internal/cli/dep_tree.go:34-38` — `runFullDepTree` after task 3-4: no branch, one unconditional `FormatDepTree` call
  - `internal/cli/dep_tree_test.go:296-319` — sibling subtest `"it outputs focused view for task with dependencies"` with the same vacuous shape (`output == "No dependencies."`), pre-existing rather than caused by this phase, in the same test function
- **Proposed shape**: Delete the subtest at `dep_tree_test.go:273-294` with its comment — the decoded-edge subtest above it covers the branch, and `TestRunDepTree` already pins the pretty full-graph bytes at `dep_tree_test.go:183-207`. If the pretty handler branch is judged to want its own guard, replace it with a byte-exact pretty golden for the populated full graph rather than a negated-string check. The false comment dies with the subtest, so it is not listed under Comment Corrections. While the same edit is open, the focused sibling at `:296-319` is the identical shape and is worth taking with it — flagged as adjacent, not as caused by this phase.
- **Bank**: `free-text-round-trip-3-4` (reviewer) — confirmed against the final tree; line numbers shifted from the deposited `:251-272` / `:266-268` to `:273-294` / `:288-290`.

### F2: the dep-tree documents join TOON sections by hand while the rest of the formatter uses the file's helper
- **Class**: drift
- **Failure**: `encodeToonFields` returns `""` when the encoder rejects a value (`toon_formatter.go:268-276`), and `joinToonSections` exists to drop that empty section so the document degrades to a shorter but well-formed one. The two dep-tree documents opt out: a field-encoding failure there emits a document whose first or last separator is a bare blank line with nothing on one side of it — a malformed TOON document, which is the exact defect class §11's conformance work exists to catch, and no test exercises it because every dep-tree section is unconditionally non-empty today. The same divergence bites the next conditionally-present field added to a dep-tree document; the detail document already carries three (`type`, `parent`, `closed`), and whoever adds one to dep tree will append it to a `strings.Join` that does not drop empties. It is noticed as a consumer's parse failure on one command and not the others.
- **Evidence**:
  - `internal/cli/toon_formatter.go:187` — `formatFullDepTree` returns `strings.Join(sections, "\n\n")` (written by task 3-2)
  - `internal/cli/toon_formatter.go:203` — `formatFocusedDepTree` returns `strings.Join(sections, "\n\n")` (written by task 3-3)
  - `internal/cli/toon_formatter.go:119` — `FormatStats` returns `joinToonSections(sections)` (switched to the helper by task 3-1, same phase)
  - `internal/cli/toon_formatter.go:96` — `FormatTaskDetail` returns `joinToonSections(sections)`
  - `internal/cli/toon_formatter.go:278-287` — `joinToonSections`, the helper both dep-tree functions bypass
- **Proposed shape**: Replace both `strings.Join(sections, "\n\n")` calls with `joinToonSections(sections)`. Behaviour-preserving on every input the suite exercises — `buildEdgeSection` never returns `""` and `encodeToonFields` only does so on an encoder error — so it is a pure refactor: tests stay green, no test semantics change.

### F3: the emptied full dep-tree document contradicts its own counts when every participant is blocked
- **Class**: behaviour
- **Failure**: `BuildFullDepTree` derives the edge list by walking down from roots, and derives `chains`/`blocked` from every participant. When no participant is a root — every dependency edge sits inside a cycle, or every blocker is a dangling ID — the document reports `dep_tree[0]{from,to}:` beside `chains: 1` and `blocked: 2`. Before this phase that branch printed prose through `FormatMessage`, so no counts surfaced and there was nothing to contradict; §8's emptied document newly puts the two halves in one document that disagrees with itself. An agent parsing it reads "no dependency edges" and "two blocked tasks" and has no way to reconcile them: it concludes the project is unblocked and tries to start a task that is blocked, or reports a dependency graph that the counts say exists and the edges say does not. Both producing states are ones the project treats as real — `internal/doctor/dependency_cycle.go` and `internal/doctor/orphaned_dependency.go` exist for them, and `tick migrate` imports from external tools — though neither is reachable through `tick dep add`, which rejects cycles.
- **Evidence**:
  - `internal/cli/dep_tree_graph.go:132-149` — roots are only tasks with `len(BlockedBy) == 0`, so a cycle or an all-dangling graph yields none
  - `internal/cli/dep_tree_graph.go:151-157` — `blocked` counts every task with a blocker, root-reachable or not
  - `internal/cli/dep_tree_graph.go:160` and `:191-234` — `countChains` unions over all participants, including the ones no root reaches
  - `internal/cli/toon_formatter.go:172-188` — full-graph TOON document: edges from roots, counts from the result
  - `internal/cli/json_formatter.go:342-350` — the same split in `formatFullDepTreeJSON` (`roots: []` beside non-zero `chains`/`blocked`)
  - `internal/cli/dep_tree_test.go:235-250` — subtest `"it reports real counts when a cycle leaves no roots"`, added this phase, pins the contradictory document (`dep_tree` empty, `chains: 1`, `blocked: 2`) as correct
  - `internal/cli/dep_tree_graph_test.go:599-613` — the pre-existing cycle case, asserting `Roots == 0`
- **Proposed shape**: Make the full-graph edge list cover every participant. After walking from the roots, seed a further walk from any participant not yet emitted, so a cycle's or an orphan's edges appear and the edge list agrees with `chains`/`blocked`. Diamond duplication (`README.md:281`, pinned by `dep_tree_graph_test.go:615`) is preserved because the root walk is unchanged for root-reachable tasks. This changes what the command outputs on those two states, so it is a `behaviour` finding and must not be folded into F2's refactor: `dep_tree_test.go:235-250` and any JSON twin are re-pinned as part of it. Pretty is out of scope (§4.1) — it keeps `result.Message` on this branch. See S1: §8 assumes the empty branch implies zero counts, and does not decide this case.

### F4: no test checks any README sample against the output the tool actually produces
- **Class**: complexity
- **Failure**: A formatter change renames a field or shifts a value — `chains` → `chain_count`, a count that moves — and every README sample still decodes as valid TOON, so `TestREADMEToonSamplesDecode` stays green while the README teaches a shape the tool no longer emits. It is noticed when a user or agent writes a parser against the documented sample and it fails at runtime, which is the defect §12.1 exists to prevent ("leaving it describing output the tool does not produce is shipping a defect"). Each sample corrected in Phases 1-3 was hand-diffed against captured binary output once, at authoring time; nothing re-checks it afterwards. Three phases have each appended to the same file — an anchor per corrected sample plus a negative string marker per past defect — so the guard's final shape is a growing list of one-off markers, none of which encodes the general rule the samples are supposed to satisfy.
- **Evidence**:
  - `internal/cli/readme_samples_test.go:140-149` — `decodeAnchoredBlock` parses the block and asserts nothing about its content
  - `internal/cli/readme_samples_test.go:154-184` — five decode-only subtests, one per corrected sample; `:180-184` is this phase's dep-tree addition
  - `internal/cli/readme_samples_test.go:19-29` — the anchor/marker constant block, appended to in each of Phases 1, 2 and 3 (`readmeDepTreeAnchor` and `readmeObjectHeaderMarker` are this phase's)
  - `internal/cli/readme_samples_test.go:186-190` and `:265-269` — the two negative string markers (`"summary{"`, `"(unchanged)"`), each guarding one past defect
  - `README.md:302-309` — the dep-tree TOON sample this phase rewrote by hand
  - Phase 6 keeps this file decode-only (`phase-6-tasks.md:391`, `:410`), so no later task closes the gap
- **Proposed shape**: One fixture-driven test that seeds a project per documented sample, renders the command through the real formatter, and compares the result to the anchored README block. Every negative marker constant then falls out — a sample compared against real output cannot contain a section header the formatter no longer emits. Cost is the reason this is a phase-boundary judgement rather than an omission: each sample needs a seeded project matching its narrative and IDs (`tick-a1b2`, `tick-c3d4`, `tick-f3e4`), and the surface spans Phase 1 (`show`, `list`), Phase 2 (transition, cascade) and Phase 3 (dep tree) plus whatever Phases 4 and 5 add. It is a pure test addition: no production code and no existing test semantics change.
- **Bank**: `free-text-round-trip-3-6` (reviewer) — confirmed against the final tree.

## Comment Corrections

- `internal/cli/toon_formatter.go:171` — names the `summary` section this phase deleted; the function's name and the `FormatDepTree` doc two lines above (`:159-162`) already state the document's shape, so the line carries nothing the code cannot.
  OLD: `// formatFullDepTree renders the full graph as a dep_tree edge list with summary.`
  NEW:

## Spec Defects

### S1: §8 assumes the emptied full document implies zero counts
- **Claim**: §8 — "Both empty branches take that shape — nothing blocked anywhere, and a named task with no dependencies either way — so an agent parses one document whether or not anything is blocked, **and reads the counts to learn which it got**."
- **Observed**: A third state produces the emptied edge list: a project where no participant is a root (every dependency edge inside a cycle, or every blocker a dangling ID). `BuildFullDepTree` then returns zero roots with non-zero counts — `internal/cli/dep_tree_graph.go:132-149` against `:151-160` — so the document reads `dep_tree[0]{from,to}:` beside `chains: 1`, `blocked: 2`. This phase's own test pins it: `internal/cli/dep_tree_test.go:235-250`. Reading the counts therefore does not tell the agent which document it got; it tells it the two halves disagree. `internal/doctor/dependency_cycle.go` and `internal/doctor/orphaned_dependency.go` establish both producing states as ones the project expects to encounter.
- **Read**: genuinely open — §8 is right that the empty branch must be a structured document, and right about the two branches it names; it did not consider the all-participants-blocked case, and nothing in the specification decides whether the edge list or the counts should give way. F3 proposes covering every participant in the edge list, which makes §8's sentence true as written.
