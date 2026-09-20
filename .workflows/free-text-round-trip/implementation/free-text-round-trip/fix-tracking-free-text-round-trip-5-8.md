## Attempt 1

ISSUES:
- `internal/cli/flags.go:160-163` — the "do not skip the following argument" half of the Do bullet is unpinned. Replacing `if def.TakesValue && !attached` with `if def.TakesValue` (i.e. keeping the old skip for the attached spelling) leaves the entire `./internal/cli` suite green. With that regression, `tick list --status=open --stauts=x` exits 0 and silently returns unfiltered results instead of `unknown flag "--stauts=x" for "list"` — the exact silent-typo failure the 0.2.0 unknown-flag rejection exists to prevent, re-opened for any command whose first flag uses the attached form, with no test to notice.
  FIX: add one subtest to `TestAttachedFlagValue` in `internal/cli/attached_flag_value_test.go`, beside `it rejects an unknown flag given in the attached form` (line 110), following that subtest's exact-stderr pattern: run `runList(t, dir, "--status=open", "--stauts=x")`, require exit code 1, and compare stderr byte-for-byte against `Error: unknown flag "--stauts=x" for "list". Run 'tick help list' for usage.\n`. The reviewer confirmed against the built binary that this is the current output byte-for-byte, and that the mutation above turns it red.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- `internal/cli/flags.go:160` — restates the condition beneath it (`def.TakesValue && !attached`); the `attached` name already carries the distinction, and code-quality.md forbids comments that say what the code says.
  OLD: 		// An attached value is already in hand; a separated one is the next argument.
  NEW:

ORCHESTRATOR ADDITION (landed on this task by the orchestrator, not from the review):
- The README documents the end-of-flags marker as the route for dash-leading free text but never mentions the attached form, which is now the only way to write a value of exactly `--` or one spelling a global flag. An agent holding such a value reads `README.md:641`'s "Flags come before `--`; everything after it is text", finds no spelling that re-attaches the value to `--title`/`--description`, and concludes the value cannot be written back — the CLI-abandonment failure §1 of the specification exists to close. Add one sentence to the free-text paragraph at `README.md:620-641` stating that a value-taking flag also accepts `--flag=value`, and that this is the spelling for a value that is exactly `--` or one that spells a global flag. The reviewer raised this as a cross-task item because a sibling task wrote that paragraph; the orchestrator is landing it here because documenting the attached form is this task's own ground, and the user has approved keeping the feature.

NOTES:
- The reviewer verified the inverted migrate test asserts the new behaviour (exit 0 plus the imported task's title) rather than merely passing, and did not treat the inversion or the CHANGELOG conflict as defects — both are user-approved.
- `CLAUDE.md`'s new bullet says "Each parser switches on the cut name" — `parseRemoveArgs` still switches on the whole argument, since `remove` has no value-taking flags. The sentence reads naturally as scoped to the parsers with value cases; no edit proposed, but a reader grepping `remove.go` will find the exception.
- `--description=` / `--title=` / `--field=` parity with the separated empty value is behaviour the Do list names but no test pins. Correct today by construction and verified by hand; a test would only guard against a future `attached && value != ""` shortcut.
- The `-` guard in `cutFlagValue` is unobservable today — the reviewer removed it and the full suite still passed — but it is cheap defence at the boundary rather than dead weight.
