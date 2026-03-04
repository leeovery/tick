---
topic: auto-cascade-parent-status
status: in-progress
work_type: feature
date: 2026-03-03
---

# Discussion: Auto-Cascade Parent Status

## Context

When a child task transitions to `in_progress`, its parent (and ancestors) remain `open` until manually started. This feels wrong — if you're working on a subtask, the parent is implicitly in progress. The question is whether Tick should automatically cascade status changes upward through the ancestor chain, and what the implications are.

The primary pain point: parent containers sit as "open" and never get closed out, especially when AI agents (Claude) are working through subtasks via the workflow system. Parents get neglected.

### References

- [Hierarchy & Dependency Model](hierarchy-dependency-model.md)
- [Data Schema Design](data-schema-design.md)

## Questions

- [x] Should starting a child auto-start its ancestors?
- [x] Should cascade behavior be unconditional or configurable?
- [x] Should completing/cancelling a parent cascade downward to children?
- [x] What happens on undo/reopen after a cascade?
- [x] Edge cases with multiple children and partial completion
- [x] How should cascaded transitions be tracked and displayed?
- [x] Should we validate parent/dependency state on mutations?
- [x] How should cascade and validation rules be architecturally organized?

---

*Each question above gets its own section below. Check off as concluded.*

---

## Should starting a child auto-start its ancestors?

### Context

The fundamental question: when a child transitions to `in_progress`, should the parent automatically follow? This touches on what "in progress" means for a parent task.

### Options Considered

**Option A: Parent as independent work item** — parent status is independently meaningful. "In progress" means *you* are working on it directly. No auto-cascade.

**Option B: Parent as container** — parent is an organizational grouping. Its status reflects aggregate child state. Auto-cascade makes sense because working on a subtask means working on the parent.

### Journey

Started by questioning whether parents are containers or work items. The user's mental model is clear: parents with children are containers/phases. The parent is a phase title. If work was independent, it should be a sibling with dependencies, not a child.

Explored what Linear does for comparison. Linear does NOT auto-start parents when children start — it's a gap users have built Zapier workarounds for. Linear only auto-closes parents when all children complete (opt-in, added Sept 2024). Their philosophy: automate low-judgment bookkeeping, keep high-judgment transitions manual.

However, Tick's existing design already treats parents as containers — `ReadyNoOpenChildren` prevents parents from being ready while children are open, and parent-child is explicitly grouping, not dependency. Auto-cascading is consistent with this existing philosophy.

Key insight: the real pain isn't "I can't see that a parent is in progress" — it's that parent containers sit around forever not getting closed. The upward start cascade is about visibility/correctness; the real fix is auto-closing parents when all children finish.

### Decision

**Yes, auto-cascade upward.** When a child transitions to `in_progress`, walk the ancestor chain and set any `open` ancestors to `in_progress`. This is consistent with Tick's existing parent-as-container model. Recursive — applies to grandparents etc.

---

## Should cascade behavior be unconditional or configurable?

### Context

Whether to add a configuration system to make cascade behavior opt-in/opt-out.

### Options Considered

**Option A: Configurable (Linear-style)** — per-team/per-project settings. Requires config file infrastructure.

**Option B: Unconditional** — always on, opinionated default.

### Journey

Considered the config route. Tick currently has zero configuration — no config file, no dotfile settings. Adding one is significant architectural cost: file format, loading, validation, defaults, documentation. It's permanent API surface.

Linear needed config because they serve thousands of teams. Tick is a personal CLI with one user currently. Opinionated defaults are a feature. The parent-as-container model is already baked into Tick's ready/blocked rules.

Decided to ship unconditional. If real user feedback demands config later, we'll have concrete requirements instead of guessing.

### Decision

**Unconditional.** No configuration. Two behaviors ship as default:
1. First child started → parent moves to `in_progress` (if parent is `open`)
2. All children reach terminal state → parent moves to `done`

If users complain, add config then with real feedback.

---

## Should completing/cancelling a parent cascade downward to children?

### Context

The inverse of upward cascade: when a parent is explicitly completed or cancelled, what happens to its children?

### Options Considered

**Option A: No downward cascade** — children are independent, user manages them.

**Option B: Cascade to all children** — parent done/cancelled means all children follow.

**Option C: Cascade only to non-terminal children** — done/cancelled children are left alone, only open/in_progress children are affected.

### Journey

Completing a parent means the phase is done — leaving orphaned open children under a done parent is messy, inconsistent state. Same logic applies even more strongly to cancellation: cancelling a phase means the work within it is cancelled.

Discussed whether done children should be affected. Walked through the scenario: parent in_progress, some children done, some open → cancel parent. The done children completed legitimately — that work happened. No reason to retroactively cancel it.

Also noted that Tick's transition table blocks done → cancelled directly. You'd have to reopen first, then cancel. So the "done parent gets cancelled, what about done children" scenario requires two steps, which gives the user a natural decision point.

### Decision

**Yes, cascade downward, but only to non-terminal children.** When a parent is marked done or cancelled:
- `open` and `in_progress` children copy the parent's terminal status
- `done` and `cancelled` children are left untouched
- Recursive — applies to grandchildren etc.

---

## What happens on undo/reopen after a cascade?

### Context

After a cascade has propagated, what happens when someone reopens a task that was part of that cascade? Three scenarios to consider.

### Options Considered

**Scenario 1: Upward cascade undo** — child started, parent auto-started. Child reopened. Should parent revert?

**Scenario 2: Downward cascade undo** — parent cancelled, children cascaded to cancelled. Parent reopened. Should children reopen?

**Scenario 3: Auto-done undo** — all children done, parent auto-closed. One child reopened. Should parent reopen?

### Journey

**Scenario 1:** No reverse cascade. Parent might have other in_progress children. The parent's state is its own now — reopening one child doesn't mean the parent isn't in progress.

**Scenario 2:** No reverse cascade. Reopening is deliberate and surgical — "I want to reconsider this specific task." Automatically resurrecting cancelled children could bring back work the user actually wanted gone.

**Scenario 3:** This one required more thought. The parent's done status was derived — conditional on all children being done. That premise is now broken. A done parent with an open child is inconsistent.

Considered auto-cascading parent to `in_progress` on child reopen, but that would require special-case logic ("reopen to in_progress if done siblings exist"). Instead, simpler to just reopen the parent to `open` using the existing reopen transition. Then normal rules apply: when the user starts the child, upward cascade kicks in and parent goes to `in_progress`. When the child finishes, parent auto-closes. The system self-corrects through existing rules.

The momentary "open parent with done children" state is transient and no different from manually creating a parent and adding done children to it. Not broken, just a brief intermediate state.

### Decision

Five clean rules covering all cascade behavior:
1. **Child starts → open ancestors go to in_progress** (upward start)
2. **All children terminal → parent goes to done** (upward completion)
3. **Parent done/cancelled → non-terminal children copy parent's status** (downward cascade)
4. **Child reopened under done parent → parent reopens** (undo auto-done)
5. **No reverse cascade on reopen otherwise** (reopening a child doesn't revert a started parent; reopening a parent doesn't reopen cancelled children)

---

## Edge cases with multiple children and partial completion

### Context

How cascade rules interact with multiple children in different states.

### Options Considered

**Mixed terminal states (some done, some cancelled):** Should parent become `done` or `cancelled`?

**All children cancelled:** Different from mixed case?

**New child added to auto-closed parent:** Should parent reopen?

**Child reparented away from in_progress parent:** Should parent revert?

### Journey

**Mixed terminal states:** If some children are done and some cancelled, all are terminal. Rule 2 fires. Parent should be `done` — the majority completed, and `done` is the positive terminal state. The parent's work was accomplished even if one subtask was abandoned.

**All cancelled:** If every child is cancelled, no work in the phase actually completed. Parent should be `cancelled` — different from the mixed case.

**New child added to done parent:** Adding a non-terminal child to a done parent breaks the "all children terminal" condition. Parent should reopen, same principle as the auto-done undo scenario (rule 4). The parent's done status was conditional.

**Child reparented away:** If a child is moved to a different parent (or made parentless), do not reverse the original parent's cascade. Same principle as rule 5 — keep rules simple, the parent's state is its own now.

### Decision

- **Mixed terminal → parent `done`** (at least one child completed)
- **All cancelled → parent `cancelled`** (no work completed)
- **New child added to done parent → parent reopens** (consistent with rule 4)
- **Child reparented away → no cascade** (consistent with rule 5)

Refined rule 2: **All children terminal → parent goes to `done`, unless all children are `cancelled`, in which case parent goes to `cancelled`.**

---

## How should cascaded transitions be tracked and displayed?

### Context

Cascades silently change multiple tasks. The user gets no feedback that something else changed, and there's no persistent record. This is especially important for downward cascades where `tick cancel <parent>` might silently cancel many children. Also relevant for debugging mistakes — without history, it's hard to trace what happened.

### Options Considered

**Persistent tracking:**

**Option A: No persistence** — just show cascade results in CLI output, no history.

**Option B: Transition history field** — add a `transitions` array to each task in JSONL, like notes. Each entry: `{from, to, at, auto}`.

**Option C: Separate log file** — dedicated audit log outside JSONL/SQLite.

**Display:**

**Option D: Flat output** — list all changed tasks, one line each.

**Option E: Tree output** — show cascade hierarchy visually.

### Journey

**Persistent tracking:** Option C (separate log) adds significant complexity — new file, new format, separate from the JSONL source of truth. Option A is too little — if a cascade makes a mistake, there's no way to trace what happened and bring tasks back to the right state.

Option B (transitions field) fits naturally. JSONL already handles arrays (tags, refs, notes). Each task carries its own history. The cache schema would need a `task_transitions` junction table like `task_notes`, but that's mechanical. Growth concern is minimal — tasks don't transition many times, and JSONL already does full rewrites on every mutation.

**Display:** Reviewed existing output. Today both toon and pretty use the same `baseFormatter.FormatTransition()` which outputs: `tick-abc123: open → in_progress`. One line, no cascade awareness.

For cascades, the key principle is: **both formats show the same information, just formatted differently.** Toon and pretty should never show different content — only different presentation.

**Upward cascade** is always a linear chain (each task has one parent), so display is straightforward — list each ancestor that changed.

**Downward cascade** is a tree (one parent, multiple children, each with their own children). Tree display makes sense for pretty. For toon, flat lines with an `(auto)` marker.

Both formats should show unchanged terminal children too, so the user can see what was *not* affected.

**Pretty format example (downward cancel):**
```
tick-parent1: in_progress → cancelled

Cascaded:
├─ tick-child1 "Login": in_progress → cancelled
├─ tick-child2 "Signup": open → cancelled
│  ├─ tick-grand1 "Form": open → cancelled
│  └─ tick-grand2 "Validation": open → cancelled
└─ tick-child3 "Logout": done (unchanged)
```

**Pretty format example (upward start):**
```
tick-child1: open → in_progress

Cascaded:
├─ tick-parent1 "Auth phase": open → in_progress
└─ tick-grand1 "Sprint 3": open → in_progress
```

**Toon format example (downward cancel):**
```
tick-parent1: in_progress → cancelled
tick-child1: in_progress → cancelled (auto)
tick-child2: open → cancelled (auto)
tick-grand1: open → cancelled (auto)
tick-grand2: open → cancelled (auto)
tick-child3: done (unchanged)
```

### Decision

**Persistent tracking:** Add a `transitions` array field to the Task struct. Each entry records `{from, to, at, auto}`. Follows the same pattern as notes. Cache gets a `task_transitions` junction table.

**CLI display:** Both formats show the same information — the primary transition plus all cascaded changes and unchanged terminal children. Pretty uses tree formatting with box-drawing characters. Toon uses flat lines with `(auto)` and `(unchanged)` markers for machine parsing.

---

## Should we validate parent/dependency state on mutations?

### Context

The cascade rules assume parents are in sensible states, but currently nothing prevents adding a child to a cancelled parent or a dependency on a cancelled task. The Socratic exploration of "what about tasks with no children that gain their first child?" revealed this gap.

### Journey

**Adding a child to a cancelled parent:** Almost certainly a mistake. The user probably meant to add it somewhere else, or forgot the parent was cancelled. Cancelled means "abandoned" — adding new work to abandoned work is contradictory.

**Adding a child to a done parent:** Already decided — parent reopens (rule 4 / new child edge case). Done means "completed but could be revisited." Adding a child is a form of revisiting.

**Adding a dependency on a cancelled task:** A cancelled blocker is already treated as resolved by ready/blocked queries (cancelled counts as closed). So it's harmless but pointless — you're saying "I'm blocked by something that's already been abandoned." Block it for cleanliness and consistency.

Discussed whether to block or warn. Blocking is cleaner — one extra step (reopen first) if you really mean it. This makes the explicit reopen a deliberate choice, not an accidental state.

This led to a broader observation: the rules are accumulating across transition validation, cascade logic, dependency validation, and now parent state validation. Currently scattered across `transition.go`, `dependency.go`, and CLI handlers. Enough rules to justify consolidating.

### Decision

**Block mutations against cancelled tasks:**
- Cannot add a child to a cancelled parent → error: "cannot add child to cancelled task, reopen it first"
- Cannot add a dependency on a cancelled task → error: "cannot add dependency on cancelled task, reopen it first"

**Adding a child to a done parent** remains allowed — triggers parent reopen (existing rule).

Consistent principle: `cancelled` is a hard stop; `done` is soft and revisitable.

---

## How should cascade and validation rules be architecturally organized?

### Context

With cascade rules, parent state validation, dependency state validation, and existing transition validation, the rule count is growing. Currently scattered across `task/transition.go` (transition table), `task/dependency.go` (cycle detection, child-blocked-by-parent), and CLI handlers (various checks). Consolidating into a single architectural unit would prevent further scattering and make the rules discoverable.

### Options Considered

**Option A: Single `cascade.go` file** — co-locate all cascade and validation logic in one file in `internal/task/`. Functions like `ApplyCascade(tasks []Task, changedID string, action string) []Change`. Not a framework, just co-location.

**Option B: Rule-based engine** — define rules as data structures with conditions and actions. More extensible but heavier.

**Option C: `StateMachine` struct** — a type that encapsulates all transition logic, validation, and cascade rules. Methods like `Transition()`, `ValidateAddChild()`, `ValidateAddDep()`, `Cascades()`.

### Journey

Counted the distinct rules that would be consolidated:

1. Transition validation (open → in_progress, etc.) — currently in `transition.go`
2. Upward start cascade (new)
3. Upward completion cascade with done-vs-cancelled logic (new)
4. Downward done/cancel cascade (new)
5. Auto-done undo — child reopened under done parent (new)
6. New child added to done parent → reopen (new)
7. Block adding child to cancelled parent (new)
8. Block adding dependency to cancelled task (new)
9. Cycle detection — currently in `dependency.go`
10. Child-blocked-by-parent rejection — currently in `dependency.go`

10 rules total. 2 existing (transition validation, cycle detection + child-blocked-by-parent), 8 new. Enough to justify a proper home.

Option B (rule engine) feels like over-engineering for 10 rules — building a framework where methods would do.

Option A (single file) is pragmatic but doesn't give a clean API surface. Callers still need to know which functions to call and in what order.

Option C (`StateMachine` struct) gives a clean API — callers interact with one type. Sketched a rough shape:

```go
type StateMachine struct{}

// Core transition
func (sm *StateMachine) Transition(t *Task, action string) (TransitionResult, error)

// Validation (before mutations)
func (sm *StateMachine) ValidateAddChild(parent *Task) error
func (sm *StateMachine) ValidateAddDep(blocker *Task) error

// Cascades (after a transition, returns additional changes)
func (sm *StateMachine) Cascades(tasks []Task, changed *Task, action string) []CascadeResult
```

`Cascades()` returns a list of additional tasks that need changing. The caller (inside `Mutate()`) applies them in a loop — each application could trigger more cascades (recursive bubbling for upward completion). Loop until no more cascades. This keeps cascade logic in one place while the storage layer handles atomicity.

Discussed migration strategy. Since only 2 rules currently exist and we're adding 8, it makes sense to migrate the existing rules into the `StateMachine` at the same time rather than having them live in two places. The migration is mechanical — `task.Transition()` becomes `sm.Transition()`, callers update accordingly.

User noted unfamiliarity with Go patterns for this kind of thing and suggested researching whether state machines in Go are a solved problem before committing to an approach.

### Decision

**`StateMachine` struct in `internal/task/`** that consolidates all transition, validation, and cascade rules. Migrate existing `transition.go` and `dependency.go` logic into it. Research Go state machine patterns before finalizing the internal design — the external API shape (Transition, Validate*, Cascades) is settled, but the internal implementation may benefit from established patterns.

---

## Summary

### Key Insights

1. Tick's existing parent-child model already treats parents as containers (ReadyNoOpenChildren, no dependency semantics). Auto-cascading is consistent with this philosophy, not a departure from it.
2. Linear's approach (opt-in auto-close only, no upward start cascade) solves a narrower problem. Tick can be more opinionated as a personal CLI with a single user.
3. Five cascade rules plus edge case refinements cover the full state space without special-case logic. The rules compose cleanly — each addresses a specific scenario without conflicting with others.
4. Reopen behavior is deliberately conservative (no reverse cascade) except when a derived state's premise is broken (auto-done undo, new child added).
5. Transition history on each task (like notes) provides audit trail without new infrastructure. Same pattern as existing array fields.
6. CLI output shows same content in both formats — pretty uses tree display, toon uses flat tagged lines.
7. Cancelled tasks are a hard stop — cannot add children or dependencies. Done tasks are soft — adding a child triggers reopen. Consistent principle across mutations.
8. A `StateMachine` struct consolidates 10 rules (2 existing + 8 new) into a single architectural unit. Clean API surface, discoverable rules, prevents further scattering.

### Current State

All questions resolved:
- Upward start cascade: unconditional, recursive
- Upward completion cascade: unconditional, recursive, with done-vs-cancelled distinction
- Downward cascade: non-terminal children only, recursive
- Reopen behavior: conservative, auto-done undo only
- No configuration system
- Edge cases handled consistently
- Transition history: array field on Task, like notes
- CLI display: tree (pretty), flat tagged (toon), same content both formats
- Cancelled-task mutation blocking: children and dependencies
- Architecture: `StateMachine` struct consolidating all rules

### Next Steps

- [ ] Research Go state machine patterns before finalizing internal design
- [ ] Specification: formalize all rules, transition tracking, display, and StateMachine architecture
- [ ] Implementation: build StateMachine, migrate existing rules, add cascade rules
