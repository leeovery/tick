# Plan: Same Second Tasks Sort By Id

## Phases

### Phase 1: Creation sequence and total list-family ordering
status: draft

**Goal**: Record an explicit creation sequence on every task (assigned on create, backfilled on read by running maximum, carried into a schema-v3 cache column) and end the list/ready/blocked sort on created, sequence, then task ID. Tied tasks then read back in authoring order, and a duplicate sequence falls back to one fixed order.

**Why this order**: This is the root-cause fix and the reproduction. Nothing else in the spec means anything until the sequence exists and the sort reads it. The show children clause, the doctor check and the documented contract all depend on it.

**Acceptance**:
- [ ] Five same-second tasks created in order step-1..step-5, with IDs chosen so ascending-ID order contradicts authoring order: `tick list` and `tick list --parent P` both return step-1..step-5
- [ ] Same fixture: `tick ready --parent P --count 1` returns step-1, not the ID-lowest task
- [ ] Tied open tasks plus an in_progress task authored after them: `tick ready` lists the in_progress task first, then the open tasks in authoring order
- [ ] Same-second blocked tasks: `tick blocked` returns them in authoring order
- [ ] A migration import whose source carries no creation times writes a batch with identical timestamps: listing it returns import order
- [ ] A tasks.jsonl with no seq field on any record and one shared creation second: listing returns line order, and after a create/update/remove it still does. The rewritten file now carries seq values in record order
- [ ] A file where some records carry seq and some do not (post-merge shape), all in one second: backfill assigns by running maximum and the newest record orders last
- [ ] A new task created in a project whose highest seq is N is written with seq N+1; the value survives update, remove of another task and `tick rebuild` unchanged, and round-trips through tasks.jsonl identically
- [ ] A project with a v2 cache: the next command rebuilds the cache at schema v3, and the tasks table's seq column holds each task's sequence (asserted directly, not only through order)
- [ ] Two same-second tasks sharing one seq: filtered and unfiltered listings both return them in task-ID order, the same every run
- [ ] Existing ordering tests pass unchanged. The migrate "values already in storage untouched" test is updated for the added field

### Phase 2: tick show sub-lists in creation and declaration order
status: draft

**Goal**: Order a task's children by created, sequence, task ID (no priority term) and its blockers by declaration order, using a new dependency ordinal in the cache. Apply this across every detail document that renders these sections, and across positional --field addressing.

**Why this order**: The children clause needs the sequence built in Phase 1. The blocker ordinal changes the cache schema that Phase 1 already moved to v3. The change is intended and has no existing test coverage, and it affects five rendering sites plus the conformance inventory, so it gets its own checkpoint.

**Acceptance**:
- [ ] Parent with same-second children authored c1..c3, whose IDs contradict that order: `tick show P` lists children c1, c2, c3
- [ ] A later-authored child with better priority: `tick show P` still lists it after its earlier-created sibling
- [ ] Children sharing a seq and a creation second: `tick show P` lists them in task-ID order, the same every run
- [ ] Task whose blocker created later was declared first in blocked_by: `tick show` lists that blocker first, matching `tick dep tree` edge order and the record's blocked_by array
- [ ] `create`, `update`, `note add` and `note remove` output detail documents whose children and blocked_by sections follow the same order as `tick show`. The conformance inventory still decodes each one
- [ ] `tick show P --field children.0` returns the first-authored child and `--field blocked_by.0` returns the first-declared blocker

### Phase 3: Duplicate-sequence visibility and the documented contract
status: draft

**Goal**: Add a read-only doctor check that reports each duplicate-sequence group with its line numbers at warning severity. Document the tiebreak and the new diagnostic in the README, each pinned by a prose assertion.

**Why this order**: The check and the documentation report on and describe the behaviour Phases 1–2 built. The check needs the seq field, and the README sentence states a guarantee that only holds once the sort is fixed.

**Acceptance**:
- [ ] tasks.jsonl with two records sharing a seq: `tick doctor` reports the duplicate group with both line numbers as a warning and exits zero
- [ ] A clean file, or a file with no seq on any record (predating the field): `tick doctor` reports the check passing with a single result
- [ ] Running the check never modifies tasks.jsonl
- [ ] The check is registered in RunDoctor and the doc comment's check count and names reflect it
- [ ] README's sort-contract sentence states that tasks tied on creation date within a priority band come back in creation order, and a prose test fails if it is removed
- [ ] README's doctor enumeration names duplicate creation sequences as its own entry, and a prose test fails if it is folded back into "duplicates"
