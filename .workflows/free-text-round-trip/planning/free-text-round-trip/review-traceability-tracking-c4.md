# Review Tracking: Free Text Round Trip - Traceability

## Findings

### 1. The JSON conformance coverage is presented as a specification requirement the specification never states

**Type**: Hallucinated content
**Spec Reference**: §11 — its three parts are "Every structured command's output is decoded by a real TOON reader in the suite, and the test fails if it will not parse", "One deliberately awkward task becomes a permanent fixture", and "Rewritten assertions check decoded values, not output text", plus "**No byte-level pinning is kept in the machine formats.** … That trade is taken for toon and JSON". The sentence the plan quotes — "JSON output for each listed command is parsed and asserted by decoded value" — appears nowhere in the specification (`grep -n 'JSON output for each' specification.md` → no match).
**Plan Reference**: Phase 3 task `free-text-round-trip-3-5` (Context); Phase 4 task `free-text-round-trip-4-4` (Context); Phase 6 task `free-text-round-trip-6-5` (Problem, and two Context paragraphs)
**Move**: settled
**Change Type**: update-task

**Problem**:
Phase 6 builds a second conformance driver that runs every document in the inventory under `--json`, parses each one, and walks it for list-nullity, boolean `auto` and numeric `index`. That is real work with a real cost, and the plan tells whoever implements or trims it that the specification demanded it — quoting §11 for a sentence §11 does not contain, in three separate tasks. A reader who goes to the specification to check what is mandatory finds nothing there and cannot tell whether the JSON enumeration is a decision they may not touch or a choice the planner made. The plan names its own calls everywhere else — the marker's recognition point, the filtered-JSON map, the count-zero header retention — and this one is dressed as the specification's.

**Proposal**:
Keep the coverage and correct the attribution. The specification decides that JSON assertions check decoded values rather than pinned text ("That trade is taken for toon and JSON"), that a JSON consumer gets the same structured answer as a TOON one (§4.2), and that a command producing a record and status changes emits one document rather than two (§7.4) — a property nothing checks per document unless JSON is parsed per document. Running the inventory under `--json` follows from those three; what does not follow is that §11 said so. So the three citations are replaced with §11's actual words, and task `free-text-round-trip-6-5` gains one sentence naming the JSON enumeration as its own extension and what carries it. No scope changes: every Do step, acceptance criterion and test in `free-text-round-trip-6-5` stands, and Phase 6's acceptance criterion in `planning.md` stands as the plan's own commitment, now grounded in the task rather than in a quotation.

**Current**:

Site 1 — `phase-3-tasks.md`, task `free-text-round-trip-3-5`, Context:

```markdown
> §11: "Rewritten assertions check decoded values, not output text," and "JSON output for each listed command is parsed and asserted by decoded value" — the rewritten subtests unmarshal rather than matching strings.
```

Site 2 — `phase-4-tasks.md`, task `free-text-round-trip-4-4`, Context:

```markdown
> §11: "JSON output for each listed command is parsed and asserted by decoded value" and "No byte-level pinning is kept in the machine formats."
```

Site 3 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Problem:

```markdown
**Problem**: §11 requires JSON output for each listed command to be parsed and asserted by decoded value, and §4.2 requires a JSON consumer to get the same structured answer as a TOON one. Today each command's JSON assertions sit beside its toon ones, so the two coverages are maintained by hand and drift: a branch added to the toon table has no JSON counterpart unless someone remembers to write one. JSON also carries invariants toon does not have and that nothing checks across the board — a list unmarshalling to `null` instead of `[]`, an `auto` rendered as a quoted string rather than a boolean, and a second document concatenated after the first, which is exactly what `create` emitted before Phase 2 folded the transition output into the record.
```

Site 4 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Context, first paragraph:

```markdown
> §11: "JSON output for each listed command is parsed and asserted by decoded value." And: "Rewritten assertions check decoded values, not output text."
```

Site 5 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Context, final paragraph:

```markdown
> The specification says each listed command's JSON is parsed and asserted by decoded value but does not fix the shape of the check. "Exactly one JSON value, then EOF" is chosen because the top-level shape is not uniform — `FormatTaskList` marshals an array while every other document marshals an object — and the property §7.4 actually requires is that the stream holds one document, not that it holds an object.
```

**Proposed Text**:

Site 1 — `phase-3-tasks.md`, task `free-text-round-trip-3-5`, Context:

```markdown
> §11: "Rewritten assertions check decoded values, not output text," and "**No byte-level pinning is kept in the machine formats.** … That trade is taken for toon and JSON" — the rewritten subtests unmarshal rather than matching strings.
```

Site 2 — `phase-4-tasks.md`, task `free-text-round-trip-4-4`, Context:

```markdown
> §11: "Rewritten assertions check decoded values, not output text" and "**No byte-level pinning is kept in the machine formats.** … That trade is taken for toon and JSON."
```

Site 3 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Problem:

```markdown
**Problem**: §11 takes its no-byte-level-pinning trade for toon and JSON alike and requires every rewritten assertion to check decoded values rather than output text, and §4.2 requires a JSON consumer to get the same structured answer as a TOON one. Today each command's JSON assertions sit beside its toon ones, so the two coverages are maintained by hand and drift: a branch added to the toon table has no JSON counterpart unless someone remembers to write one. JSON also carries invariants toon does not have and that nothing checks across the board — a list unmarshalling to `null` instead of `[]`, an `auto` rendered as a quoted string rather than a boolean, and a second document concatenated after the first, which is exactly what `create` emitted before Phase 2 folded the transition output into the record.
```

Site 4 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Context, first paragraph:

```markdown
> §11: "Rewritten assertions check decoded values, not output text." And: "**No byte-level pinning is kept in the machine formats.** … Decoded-value assertions survive harmless reformatting while still failing when a section goes missing or a value is wrong. That trade is taken for toon and JSON."
```

Site 5 — `phase-6-tasks.md`, task `free-text-round-trip-6-5`, Context, final paragraph:

```markdown
> §11's enumerated parse coverage is stated over a TOON reader — "Every structured command's output is decoded by a real TOON reader in the suite" — and the specification nowhere enumerates the same coverage for JSON. Running the whole inventory under `--json` is this task's extension of it, taken because §4.2 requires a JSON consumer to get the same structured answer as a TOON one, because §11's no-pinning trade is taken "for toon and JSON", and because §7.4's "the stream must be one document" is a property nothing else checks document by document. The specification does not fix the shape of the check either. "Exactly one JSON value, then EOF" is chosen because the top-level shape is not uniform — `FormatTaskList` marshals an array while every other document marshals an object — and the property §7.4 actually requires is that the stream holds one document, not that it holds an object.
```

**Resolution**: Pending
**Notes**:

---

### 2. The bare-value byte assertion is attributed to §11, which asks only that the filtered documents decode

**Type**: Hallucinated content
**Spec Reference**: §11 — "the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)". §3.1 exempts the bare value from the must-parse rule; §9.2 fixes its bytes — "its own bytes followed by a single newline". The sentence the plan quotes — "Both field-selection document forms … are decoded in the suite, and bare-value output is asserted as bytes" — appears nowhere in the specification (`grep -n 'asserted as bytes' specification.md` → no match).
**Plan Reference**: Phase 4 task `free-text-round-trip-4-8` (Context); Phase 6 task `free-text-round-trip-6-4` (Problem and Context)
**Move**: settled
**Change Type**: update-task

**Problem**:
Two tasks cite §11 for a sentence it does not carry, and the invented half is the one that matters: the specification asks that the filtered documents decode, and says nothing about asserting the bare value's bytes. The byte assertion is worth keeping — §9.2 fixes the bare value as its own bytes plus one newline, and no decoder will ever accept it, so bytes are the only check that reaches a decided behaviour — but a reader trimming Phase 6 cannot tell that from the plan, because the one part §11 genuinely mandates and the one part the plan added sit inside the same quotation marks. Task `free-text-round-trip-6-6` already draws the line correctly when it keeps those byte assertions out of its sweep; `free-text-round-trip-6-4` and `free-text-round-trip-4-8` contradict it by calling the same assertion a specification requirement.

**Proposal**:
Keep the assertion and correct the attribution, in the words `free-text-round-trip-6-6` already uses for the same boundary. §11's real sentence covers the two filtered documents; §3.1 exempts the bare value because it is not a document; §9.2 fixes its exact bytes. That split is what the record settles, and stating it costs the plan nothing: every Do step, acceptance criterion and test in `free-text-round-trip-6-4` stands, and Phase 6's acceptance criterion in `planning.md` stands as the plan's own commitment.

**Current**:

Site 1 — `phase-4-tasks.md`, task `free-text-round-trip-4-8`, Context:

```markdown
> §11: "Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes." That coverage is Phase 6's; this task only anchors the README's own sample.
```

Site 2 — `phase-6-tasks.md`, task `free-text-round-trip-6-4`, Problem:

```markdown
**Problem**: §11 requires both field-selection document forms — a multi-field selection and one narrowed by position — to be decoded in the suite, and requires bare-value output to be asserted as bytes. Phase 4 built those outputs and tested each behaviour where it landed, but the inventory of documents stops at full `show`. §3.1 also carves out the only two things in the whole CLI that are exempt from the must-parse rule — a bare value from `show --field`, and a selection that prints no bytes at all — and an exemption that exists only as an absence is indistinguishable from an oversight: a reader of the table cannot tell whether the bare value was considered and excluded or simply forgotten.
```

Site 3 — `phase-6-tasks.md`, task `free-text-round-trip-6-4`, Context, first paragraph:

```markdown
> §11: "the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)." And: "Both field-selection document forms — a multi-field selection and one narrowed by position — are decoded in the suite, and bare-value output is asserted as bytes."
```

**Proposed Text**:

Site 1 — `phase-4-tasks.md`, task `free-text-round-trip-4-8`, Context:

```markdown
> §11: "the coverage is counted in documents rather than commands: … every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)." That coverage is Phase 6's, along with the byte assertion Phase 6 adds for the bare value §3.1 exempts from it; this task only anchors the README's own sample.
```

Site 2 — `phase-6-tasks.md`, task `free-text-round-trip-6-4`, Problem:

```markdown
**Problem**: §11 requires every document `show` produces to be decoded in the suite, "a multi-field selection and one narrowed by position included". Phase 4 built those outputs and tested each behaviour where it landed, but the inventory of documents stops at full `show`. §3.1 also carves out the only two things in the whole CLI that are exempt from the must-parse rule — a bare value from `show --field`, and a selection that prints no bytes at all — and an exemption that exists only as an absence is indistinguishable from an oversight: a reader of the table cannot tell whether the bare value was considered and excluded or simply forgotten. The exempt bare value is not thereby unchecked: §9.2 fixes its bytes exactly, and bytes are the only assertion that reaches output no decoder will accept.
```

Site 3 — `phase-6-tasks.md`, task `free-text-round-trip-6-4`, Context, first paragraph:

```markdown
> §11: "the coverage is counted in documents rather than commands: every branch a listed command can take, the emptied forms of §8 included, and every document `show` produces, a multi-field selection and one narrowed by position included (§9.2, §9.3)."
>
> The bare-value byte assertion is this task's own rather than §11's: §11 asks for decoded-value assertions over documents, §3.1 exempts the bare value from the must-parse rule because it is not a document, and §9.2 fixes its exact bytes — "its own bytes followed by a single newline" — so a byte assertion is the only check that reaches it. Task `free-text-round-trip-6-6` states the same boundary when it keeps the bare-value byte assertions out of its sweep.
```

**Resolution**: Pending
**Notes**:

---

## Coverage Notes (no finding)

Recorded so the gaps that were checked and found closed are visible, and so deliberate departures the plan already names are not re-raised in a later cycle.

- **Every specification quotation in the plan was verified mechanically against the specification text** — all blockquoted fragments of 40 characters or more in the six task files, split on ellipses and compared after whitespace and emphasis normalisation. Two fabricated sentences came back, both attributed to §11, and both are the findings above. Every other quotation resolves to real specification text; the remaining near-misses are punctuation drift (a closing colon rendered as a full stop), quote-character conversion (the specification's double quotes rendered as single quotes inside a quotation), a fenced example collapsed into the sentence that introduces it, or a quotation of source code rather than the specification (`toJSONRelated`'s "Always returns a non-nil empty slice to ensure JSON `[]` instead of `null`", which is `internal/cli/json_formatter.go:123` and is labelled as such).
- **Every non-quoted "§N requires …" claim in a task's Problem or Solution was checked against the named section**: §7.2's merge and marking rules, §7.4's one-document and always-present rules, §4.2's JSON parity, §5.2's focused-identity fields, §8's emptied document, §9.2's filtered document, §9.6's out-of-range failure, §9.7's format-flag application in JSON and pretty, §10.2's post-marker rule, §11's fixture and no-pinning requirements, and §8's count-zero schema all state what their sections state. The two overclaims are the findings above.
- **Direction 1 (spec → plan)**: every bolded decision in §2.1–§2.2, §3.1–§3.3, §4.1–§4.3, §5.1–§5.4, §6.1–§6.4, §7.1–§7.6, §8, §9.1–§9.8, §10.1–§10.2, §11 and §12.1–§12.2 traces to at least one task with matching acceptance criteria. §9.1's accepted-name list matches `free-text-round-trip-4-1`'s registry and `free-text-round-trip-4-8`'s README list name for name. §10.3's "no stdin or file input path" and §6.4's "no `note edit`" are honoured by absence, as are the three alternatives §4.1 declines.
- **§12.1's five README samples plus the `tick list` row** are each allocated to exactly one task, and the live README carries no other agent-format sample: `$ tick` prompts sit at lines 289, 302, 396, 408, 473, 501, 512 and 579, the unprompted toon blocks at 421-424 and 429-450, and the JSON blocks at 481-486 and 527-535 — the last being a task-list sample this work does not touch.
- **Phase 6's inventory partition remains satisfiable**: the live `commandFlags` registry carries 21 keys (19 literal plus `ready` and `blocked` from `init()`), and the plan's three declared sets — 14 must-parse, 5 prose, 2 out of scope — account for all of them with none doubled.
- **Cross-phase deferrals** each land in a receiving phase's acceptance criteria: task 1-5's create/update decode deferral → Phase 2's single-document criterion; task 1-4's and task 4-2's dash-leading deferrals → Phase 5's writability criteria; task 1-6's dep-tree sample deferral → Phase 3's README criterion; task 4-8's `--` deferral → Phase 5's README/help criterion; task 4-8's suite-coverage deferral → Phase 6's field-selection criterion.
- **Departures the plan already names, raised and settled in earlier cycles; not re-raised**: the retained count-zero header text assertions against §11's no-pinning sentence (cycle 1), and `tick create --description -- x` against §10.2's "Everything that works today works identically" (cycle 1).
- **§8's emptied full dep tree versus a dependency cycle** (Phase 3 task 4, Phase 6 task 2): the specification never identified the cycle branch, and §8's "reads the counts to learn which it got" only works if the counts are true. Settled by the record; not raised.
- **The focused dep tree carrying both directions on the populated branch** (Phase 3 task 3): derived from §8's "the document the non-empty branch produces, emptied: the same fields", which cannot hold if the populated branch's key set varies with the data. Named in the task as a change rather than left implicit; not raised.
- **Spec-silent mechanisms the plan settles and labels as such** — the notes query ordering that makes §6.3's index truthful, the `changed` section's position between `notes` and `description`, the dep-tree summary fields sitting after the edge section, the filtered-JSON `map[string]any` and its key order, the `--field` and out-of-range error wording, `parseArgs` as the marker's recognition point and its rulings on a leading `--` and a `--` in a value slot, the migrate Engine as the single normalisation point, `flagScanLimit` as the shape of the `note add` exemption, `rebuild` declared with §3.2's prose commands, the `CLAUDE.md` Formatter-interface line, and the inventory-partition guard — are each how the plan builds a decided requirement, not new requirements of the product. Not traced.
