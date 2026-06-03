# Specification: Ready Includes In-Progress

## Specification

## Overview

### Problem

`tick ready` answers "what should I act on right now?" Today both `ready` and `blocked` gate on `status = 'open'`. An `in_progress` task is therefore in *neither* view — invisible to the very command used to resume interrupted work. Start a task, get interrupted, run `tick ready`, and it points you at *new* open work instead of the started-but-dangling task. The same `status = 'open'` gate also drops a started-but-stuck task out of `blocked`. The real defect is **double invisibility**: in-progress work falls off both actionability views.

### Goal

`ready` means "everything actionable right now" = unblocked `open` **+** `in_progress`. A started task is the most ready thing there is; resuming interrupted work becomes the natural default. `blocked` stays the strict complement so no live task ever falls through the cracks.

### Actor model (decided premise)

tick is single-actor with no ownership concept — a task carries no assignee. `in_progress` means "the task *I* started and got pulled off," not "someone else claimed it." Surfacing it in `ready` means "resume this." The "someone else took it, hands off" reading would require an assignee field tick doesn't have; excluding all `in_progress` to dodge collisions can't tell *my* work from *theirs*, so it would break the single-actor resumption case. Multi-actor claiming is a separate future feature (see Out of Scope).

### Governing invariant

`ready ⊎ blocked = all live tasks`, where live = `status IN ('open','in_progress')`. Every live task is in **exactly one** of `ready`/`blocked` — never both, never neither. `done` and `cancelled` are in neither. `ready` and `blocked` share one identical status gate; `blocked` is the literal De Morgan inverse of `ready`'s `NOT EXISTS` conditions. This invariant is what makes the change small (flip one shared literal per side) and guarantees nothing is dropped.

---

## Working Notes
