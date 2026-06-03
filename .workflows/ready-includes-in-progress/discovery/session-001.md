# Discovery Session 001

Date: 2026-06-03
Work unit: ready-includes-in-progress

## Description (as of session)

Include in-progress tasks in `tick ready` output, keeping `blocked` consistent as its inverse; semantics and presentation to be settled in discussion.

## Seed

- seeds/2026-06-02-ready-includes-in-progress.md (inbox:idea)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

Work originated from an inbox idea: `tick ready` currently surfaces only the next actionable *open* tasks and skips tasks already in `in_progress`, leaving started-but-interrupted work invisible to the command you'd use to ask "what should I be doing right now?".

The user was initially unsure whether this should be a quick-fix or something larger, and questioned the underlying semantics — whether `ready` should include `in_progress` at all, and (a noted confusion) whether terminal states belong (clarified: `done` is terminal, nothing to act on, so it does not belong in `ready`).

Shaping surfaced a genuine behaviour debate rather than a purely mechanical change: (1) the semantics of what "ready" means — an in-progress task is arguably the *most* actionable thing, but `ready` also serves the "what new work can I pull?" question, and mixing started + unstarted work muddies that; (2) the presentation question flagged in the seed — whether in-progress items appear inline, are sorted to the top, or are visually distinguished. A related constraint: `blocked` is defined as the De Morgan inverse of `ready` (`query_helpers.go`, `ReadyConditions()` / `BlockedConditions()`), so any change to "ready" must stay consistent with how "blocked" is derived.

The user confirmed the change makes sense to add but "probably requires a bit of discussion" — which ruled out quick-fix (no behaviour debate) and settled the shape as a single, coherent, ship-able feature whose discussion phase exists to resolve the semantics and presentation before spec and code.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

(none)
