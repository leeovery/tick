## Attempt 1

ISSUES:
- `internal/cli/toon_refusal_test.go:92-95` — the ten-command loop asserts only `exitCode != 1 || stdout != ""`. It never asserts the failure *is* the TOON refusal, so six of the eleven commands the criteria cover (`update`, `note remove`, `start`, `done`, `reopen`, `cancel`) have no assertion anywhere in the suite that the diagnostic names the field/section or the task ID — the second acceptance criterion is unverified for them. Worse, the loop is self-sequencing: `note add` (line 85) is expected to fail while still storing, and `note remove … 1` (line 86) depends on that store. If the store half ever regresses, `note remove` fails with "note 1 not found" — exit 1, empty stdout — and the guard stays green while checking nothing. Any unrelated error (rejected transition, flag-validation change) passes it identically. This is also why the transition path's ID semantics (see NOTES) went unexamined.
  FIX: replace the inline check with the existing `assertRefused` helper and pass the refusal marker plus the ID, e.g. `assertRefused(t, stdout, stderr, exitCode, "cannot encode", id)` for the eight single-task-document commands and `assertRefused(t, stdout, stderr, exitCode, "cannot encode", "tasks")` for `{"list"}`. The reviewer verified every one of those diagnostics against a built binary: `update`/`note add`/`note remove` yield `cannot encode field title of task <id> as TOON: …`, the four transitions yield `cannot encode section changed of task <id> as TOON: …`, `list` yields `cannot encode section tasks as TOON: …`, so the assertions pass as written. Note `assertRefused` calls `t.Fatalf` on a wrong exit code, which aborts the subtest rather than the iteration — acceptable, or hoist the loop into `t.Run` per command if you want all failures reported.
  ALTERNATIVE: keep the loop's shape and add one line, `if !strings.Contains(stderr, "cannot encode") { t.Errorf(...) }`. Cheaper, but it leaves the ID half of the criterion unverified for the six commands, which is the half the cascade path actually makes a judgement about. The reviewer recommends routing through `assertRefused`.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- README.md:494 — the paragraph promises the diagnostic names the task, but `tick list` (and any multi-task document) names only the section; the sentence overstates what a reader will see.
  OLD: A command whose TOON document would carry such a value fails, naming the task and the field it could not encode, rather than printing a document without it.
  NEW: A command whose TOON document would carry such a value fails, naming the field or section it could not encode — and the task, where the document covers one — rather than printing a document without it.
- internal/cli/round_trip_test.go:26-28 — the second clause is the plan's rationale for choosing an escape character, which code-quality.md's comment discipline keeps out of comments; the part that earns its place is that `\x1b` is a codepoint TOON refuses, which the constant name alone does not say.
  OLD: // The refused carriers: the same three free-text fields carrying a C0 control
// character TOON cannot encode, an ANSI escape being the everyday way pasted
// terminal output brings one in.
  NEW: // TOON cannot encode a C0 control character other than tab, newline and
// carriage return.
- internal/cli/toon_refusal_test.go:16 — pure signature restatement (the body is a one-line `--toon` prepend and the results are named), which golang-documentation lists as an anti-pattern to remove on sight.
  OLD: // runToon runs a tick command in TOON format and returns stdout, stderr and the exit code.
  NEW:

NOTES:
- The transition path names the document's *subject*, not the task holding the refused value. The reviewer reproduced this: a parent with a clean title and a child whose title carries `\x1b`, then `tick done <parent>` → `cannot encode section changed of task <parent> as TOON: …`. The agent inspects the parent, finds nothing it cannot encode, and has no route to the child except switching format. Not raised as an issue: the semantics are consistent — a refused `blocked_by`, `children`, `notes` or `tags` section in a `show` document names the document's task the same way, and the task body only asked for an ID "wherever the document covers a single task", which is satisfied. But it is a live judgement, and the fix above is the right place to pin whichever reading is intended.
- `formatted(t).of(document, err)` looks unusual for Go but is the correct workaround: a plain `mustFormat(t, f.FormatX(...))` is illegal, since Go only expands a multi-value call when it is the sole argument.
- `show.go:84-90` deliberately does not route through `printDocument`, because it must keep the `document != ""` guard for the selection-keeps-no-field case. Correct, and the existing coverage at `show_fields_test.go:276-358` still pins it.
- `fieldRefusal`'s `"the named fields"` fallback (`toon_formatter.go:311`) is unreachable through any value the CLI can store — every field the encoder rejects in the aggregate also rejects alone — so it is untested defensive cover. Fine as written; nothing to change.
- The reviewer independently reached the same conclusion as the bank entry already recorded for this task (`baseFormatter`'s stubs now returning the successful-empty-document shape). Not re-banked.
