# Review Tracking: Free Text Round Trip - Input Review

## Findings

### 1. The positional-selector example drops a field the caller asked for

**Source**: `discussion/free-text-round-trip.md` — "Field Extraction Flag" → Decision, the `--field description,notes.2` worked example (lines 597-604)
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §9.3 List fields are reachable by position

**Problem**:
The worked example for reaching a single note by position asks for two fields — a description and one note — and shows an answer containing only the note. Built as written, `tick show <id> --field description,notes.2` silently discards the description, and an agent that narrows one section loses every other section it named. The source's example for the same command shows both sections; the description line was dropped in transcription.

**Proposal**:
Restore the description section to the example so a positional selector visibly narrows only the section it names, and say so in a line, matching the source's rendering and §9.2's rule that a multi-field request returns the normal document minus the sections not asked for.

**Current**:
`notes.2` selects the second note. In a multi-field selection the section renders as normal, its count following the selection while each row carries its real position via the `index` column (§6.3) — `tick show tick-a1b2 --field description,notes.2`:

```
notes[1]{index,text,created}:
  2,"multi\nline\nnote","2026-09-16T08:30:00Z"
```

**Proposed Text**:
`notes.2` selects the second note. In a multi-field selection the section renders as normal, its count following the selection while each row carries its real position via the `index` column (§6.3) — `tick show tick-a1b2 --field description,notes.2`:

```
notes[1]{index,text,created}:
  2,"multi\nline\nnote","2026-09-16T08:30:00Z"

description: "Fix the parser.\n\nSteps:\n  - read the header\n  - validate"
```

A position narrows the section it names and nothing else: every other field in the selection comes back whole.

**Resolution**: Approved
**Notes**: Applied verbatim. User selected auto for remaining settled findings.

---

### 2. Which names `--field` accepts is never stated, while rejecting a name is an error

**Source**: `discussion/free-text-round-trip.md` — "Field Extraction Flag" → Decision and "Empty values and unrecognised names" (lines 573-658); the sources work entirely from examples (`description`, `notes`, `notes.2`, `title`, `status`) and never state the accepted set
**Category**: Gap/Ambiguity
**Move**: settled
**Affects**: §9.1 `--field` and `--fields` are the same flag; §9.6 Empty values, unrecognised names, and out-of-range positions

**Problem**:
An unrecognised field name fails the command with a non-zero exit, but nothing says what counts as recognised. Two builders land on two different vocabularies — one accepting `blockers` and `blocked-by`, another only `blocked_by` — and an agent that guesses a plausible name gets a hard failure with no way to know which spelling the tool wanted. The boundary the error fires on is load-bearing and currently undefined.

**Proposal**:
The accepted names are the names the output document itself uses — the top-level task fields and the section keys, exactly as they appear in a full `tick show`. That is the only answer the specification's own shape leaves standing: §9.2 defines the multi-field answer as the normal document minus what was not asked for, and §9.4 makes the task's own fields selectable "exactly like sections" to avoid a grammar that depends on which side of a boundary a name sits. A positional suffix is meaningful only on a section holding a list; applied anywhere else the name is unrecognised and takes §9.6's error.

**Proposed Text**:
**The names the flag accepts are the names the output document uses** — the task's own top-level fields (`id`, `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`) and the section keys (`description`, `notes`, `tags`, `refs`, `children`, `blocked_by`), spelled as a full `tick show` spells them. There is no second vocabulary to learn: what you read in the output is what you ask for. A positional suffix (`notes.2`, §9.3) attaches only to a section that holds a list; on anything else the whole name is unrecognised and takes §9.6's error.

Several of these are emitted only when set — `type`, `parent` and `closed` among the task's fields (`sed -n '265,285p' internal/cli/toon_formatter.go`), and `tags`, `refs` and `description` among the sections. **Recognition does not depend on presence**: a name on this list is always recognised, and asking for one the task does not carry is an empty field, which prints nothing and exits successfully (§9.6). A name absent from the list is unrecognised whatever the task holds.

**Resolution**: Approved
**Notes**: Applied verbatim under auto. Disposal — move held as `settled`; the derivation from §9.2 and §9.4 stands. Proposed Text amended before presentation: the staged enumeration omitted `parent` and `closed`, which `buildTaskSection` emits conditionally (`sed -n '265,285p' internal/cli/toon_formatter.go`), and said nothing about whether a conditionally-emitted name is recognised when absent — load-bearing, since §9.6 makes an unrecognised name a hard error.

---

### 3. Nothing commits to documenting the new input surface

**Source**: `discussion/free-text-round-trip.md` — "Write Side Input" → Decision, "`--` … becomes the canonical documented way to pass free text that may begin with a dash" (line 700); "Published Documentation Owed a Correction" → Decision (lines 792-798)
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §12.1 The README is updated as part of this work

**Problem**:
The work ships two new ways to talk to the tool — `--field`/`--fields` on `show`, and `--` as the marker for free text beginning with a dash — and commits to documenting neither. The README work is scoped to correcting stale output samples. A marker described as "the canonical documented way" that appears in no documentation is a marker no caller reaches for, which leaves `tick create "- some title"` failing for exactly the users who never find out the answer, and leaves the flag the whole work exists to serve undiscoverable.

**Proposal**:
Extend §12.1's README scope to the new input surface. The source calls `--` "the canonical documented way", which only holds if it is written down somewhere, and §12.1's own reason — live documentation someone reads to learn the tool — applies identically to a flag that did not exist before.

**Current**:
Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work those samples show output the tool no longer produces. It is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect.

**Proposed Text**:
Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work those samples show output the tool no longer produces. It is live documentation someone reads to learn the tool, not a record of a past decision, so leaving it describing output the tool does not produce is shipping a defect.

The same section gains the new input surface, for the same reason: `--field`/`--fields` on `show` (§9), and `--` as the way to pass free text that may begin with a dash (§10.2). Calling `--` the canonical form only means something if a caller can find it written down, and a flag documented nowhere is a flag nobody uses. The command's own help text carries both alongside.

**Resolution**: Approved
**Notes**: Applied under auto, with the help-text clause grounded: `TestCommandFlagsMatchHelp` (flag_validation_test.go:310) makes a help entry mandatory for any registered long flag, so documenting `--field` in help is a project constraint rather than an added preference.

---

### 4. The two paths already available for bare free text, and why neither is the answer

**Source**: `discussion/free-text-round-trip.md` — "Context" (lines 7, 16): the `tick show --json` escape hatch "works but is the wrong shape", and "A fourth output format was rejected. `--raw` was pressed on and dropped"
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §9 Field Selection (opening)

**Problem**:
The specification introduces field selection without recording that two cheaper-looking routes to the same end were examined and dropped: a fourth output format (`--raw`), and the `--json` output that already hands back the string. Both are the obvious first suggestions, and with the reasoning missing they come back — either during the build, as "why not just use `--json`", or later as a `--raw` proposal that has to answer for every command in the CLI. The spec records declined alternatives everywhere else it made a choice; this one is the choice the work was named for.

**Proposal**:
Carry the two declines into §9's opening, with the reasons the sources gave: `--json` costs tokens and hands back a string the caller must parse JSON syntax off, and `--raw` was dropped because a format has to answer for every command in the CLI and carries an ongoing consistency burden across all of them — field extraction being the framing that avoids designing a format at all.

**Current**:
The companion to the format repair: a way to ask for one field's value and get it with nothing around it — no header, no indentation, no quoting. This is the case that started the work, where an agent needed a task's description as a plain string and went to the raw data file instead.

**Proposed Text**:
The companion to the format repair: a way to ask for one field's value and get it with nothing around it — no header, no indentation, no quoting. This is the case that started the work, where an agent needed a task's description as a plain string and went to the raw data file instead.

Two routes to the same end were declined. `tick show --json` already returns the string and is the wrong shape for it: it costs tokens and hands back a value the caller has to parse JSON syntax off — the agent wanted the text, not a document containing it. A fourth output format, `--raw`, was pressed on and dropped: a format has to answer for every command in the CLI — `tick list --raw`, `tick stats --raw` — and carries that consistency burden forever. A flag that selects fields avoids designing a format at all.

**Resolution**: Pending
**Notes**:

---
