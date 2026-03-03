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
- [ ] Edge cases with multiple children and partial completion
      - Some siblings done, some open — parent status?
      - All children done — auto-complete parent?

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
