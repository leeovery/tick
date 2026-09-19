## Attempt 1

ISSUES:
- `CLAUDE.md:48` — the project's Key Patterns entry still documents `Run<Command>(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error`, a signature nine `Run*` functions no longer have after this diff. CLAUDE.md is loaded into every session on this repo, so it is the reference a future contributor or agent follows when adding a command; following it produces a handler taking a single `args []string`, wired in `App.Run` with `flagArgs` alone, which compiles and silently drops post-marker text — precisely the opt-in trap this task exists to remove, noticed only when a user's `--`-quoted dash-leading text vanishes. This is the only place in the live tree (outside archived `.workflows/` material) still asserting the old shape.
  FIX: Replace line 48 with the new signature and a clause naming where the split happens, e.g. `- **Handler signature:** ` + "`Run<Command>(dir string, fc FormatConfig, fmtr Formatter, flagArgs, literals []string, stdout io.Writer) error`" + ` — ` + "`App.Run`" + ` splits arguments at the end-of-flags marker once and hands both halves to every handler; fully-positional commands concatenate them in order.`
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- `internal/cli/note.go:42` — restates the three lines below it (concat, `positional[0]` as ID, `positional[1:]` joined); the diff touched this comment, and the code says all of it.
  OLD: 	// Parse positional args: first is the ID, the rest are joined as text.
  NEW:
- `internal/cli/end_of_flags_test.go:29` — the comment sits as the doc for a two-constant group but names and describes only the second constant.
  OLD: // beadsClosedFixture carries the beads "closed" status, which maps to a done task.
  NEW: // Beads issue fixtures; the "closed" status maps to a done task.

NOTES:
- Deliberate behaviour change with no guard either way: `tick dep -- add a b` and `tick note -- add <id> text` previously routed (the sub-subcommand came off the unsplit slice) and now return "sub-command required", because routing reads `flagArgs[0]`. The Do list directs this and it matches the documented rule, and the failure mode of the alternative is thin (an error-message shape, no data loss), so the reviewer is not raising it — but nothing pins the chosen reading, and the spec's wording ("read as a flag") does not settle a positional command word.
- `internal/cli/app.go:65` — `handleHelp(subArgs)` is now the only consumer of the unsplit slice. Behaviour is unchanged (`tick help -- create` still prints create help), but it is the one line a reader copying the dispatch pattern could take the wrong shape from.
- `splitLiteralArgs`'s clamp is now unreachable from its single caller: `flags.literals` counts only arguments appended to `rest`, so `n <= len(subArgs)` always. Retained per the task's explicit instruction, and `TestSplitLiteralArgs` still pins it.
- `golang-code-style` puts a SHOULD at ≤4 parameters; the `Run*` functions are now at 6 (`RunTransition` at 7). Plan-directed, so not a finding — recorded only because the next signature change to these functions inherits it.
- `RunDepTree` computes `slices.Concat` before the `fc.Quiet` early return, and `parseDepArgs` still does `append([]string{}, args...)` on a slice its callers already built fresh. Both harmless; the second is pre-existing.
