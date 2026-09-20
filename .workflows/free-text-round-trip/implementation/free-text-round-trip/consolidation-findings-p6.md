# Consolidation Findings: free-text-round-trip (Phase 6)

Every file this phase touched is a test file, so only failure-mode candidates
were admitted (finding-floor.md → Test Files). Duplicated test helpers the
phase created beside pre-existing ones — `decodeConformanceDoc` beside
`decodeToonDoc`, `assertConformanceKeys` beside `assertToonKeySet`,
`runTickConformance` beside `runTick` — were considered and dropped: reuse and
symmetry between test helpers are never findings.

## Findings

### F1: the coverage guard certifies commands off a self-declared `Command` nothing ties to the arguments the entry runs
- **Class**: behaviour
- **Failure**: `TestConformanceInventoryCoversEveryCommand` proves every key of `commandFlags` is claimed by exactly one of the must-parse, prose and out-of-scope sets. The must-parse set is built from `conformanceDoc.Command`, a free-text field written independently of the args the entry's `Setup` returns. An entry whose `Command` reads `ready` while its `Setup` runs `list --ready`, or a copy-pasted entry that keeps `Command: "list"` on a `blocked` setup, makes the guard green for a command no entry ever runs. The gap is silent for the object-shaped commands: `jsonShapeProblem` only distinguishes list-shaped from object-shaped documents, so swapping `show` for `stats`, or `create` for `update`, in a `Command` field raises nothing. Nobody notices until a malformed document ships from the uncovered command with the whole conformance suite passing.
- **Evidence**:
  - `internal/cli/conformance_test.go:23-33` — `conformanceDoc.Command` is documented as "the fully-qualified command name as commandFlags spells it", a contract the struct does not encode and no test enforces
  - `internal/cli/conformance_test.go:210-539` — 45 inventory entries, each writing `Command` and the `Setup` args separately
  - `internal/cli/conformance_test.go:762-822` — `TestConformanceInventoryWellFormed` checks duplicate names and `Setup`/`NotADocument` exclusivity, never `Command` against the args
  - `internal/cli/conformance_test.go:1486-1499` — `conformanceMustParseCommands` derives the covered set from `Command` alone
  - `internal/cli/conformance_test.go:1526-1533` — the guard that reports the coverage as complete
  - `internal/cli/conformance_test.go:673-684`, `internal/cli/conformance_test.go:743` — `jsonShapeProblem` keys off `Command` too, so a wrong value also picks the wrong top-level shape assertion
- **Proposed shape**: extend `inventoryProblems` (`conformance_test.go:763`) with a check that each entry's `Setup` args begin with `strings.Fields(entry.Command)`, reporting the entry name and both spellings when they disagree. Entries carrying `NotADocument` have no `Setup` and are skipped. Give it a negative case in `TestConformanceInventoryWellFormed` alongside the existing three, mirroring their fixture style. The current inventory already satisfies the rule, so no entry changes.
- **Bank**: reviewer entry on task 6-1 — "Nothing ties a conformanceDoc.Command to the arguments its Setup actually runs, so task 6-3's coverage guard can certify a command the suite never exercises." Confirmed against the final state: 6-3 landed the coverage guard and 6-4 through 6-6 added no tie.

### F2: the prose exemption is format-blind, so five commands that do emit JSON documents are excluded from the JSON conformance driver
- **Class**: behaviour
- **Failure**: `conformanceProseCommands` declares `dep add`, `dep remove`, `remove`, `init` and `rebuild` to emit "a confirmation message rather than a document". That holds for toon and pretty, where `baseFormatter` returns plain text — but under `--json` all five emit a JSON object: `FormatDepChange` returns `{"action","task_id","blocked_by"}`, `FormatMessage` returns `{"message"}`, and `FormatRemoval` returns `{"removed":[…],"deps_updated":[…]}`. `TestJSONOutputConformance` drives the same single inventory, so it never runs those five under `--json`, and the coverage guard — whose whole job is to catch exactly this gap — reports the inventory as complete because the prose set claims them. What goes wrong: an agent runs `tick remove tick-abc123 --json` or `tick dep add … --json` and its parser breaks, because nothing end-to-end asserts the command's stdout is one parseable JSON value and nothing asserts `removed` and `deps_updated` are arrays rather than `null`. The existing coverage is formatter-level only (`json_formatter_test.go:451`, `json_formatter_test.go:531`), so a handler printing anything alongside the object — the shape §4.2 records the dep-tree handler as having had, a bare `message` object emitted beside the formatter's output — ships with the suite green.
- **Evidence**:
  - `internal/cli/conformance_test.go:1478-1480` — `conformanceProseCommands` and the comment stating its rationale
  - `internal/cli/conformance_test.go:1503-1524`, `internal/cli/conformance_test.go:1526-1533` — the guard that accepts the prose claim as coverage
  - `internal/cli/conformance_test.go:1566-1570` — `TestJSONOutputConformance` drives `conformanceDocs` only
  - `internal/cli/conformance_test.go:639-642` — `jsonConformanceListKeys` omits `removed` and `deps_updated`, the two always-array keys of the removal document
  - `internal/cli/json_formatter.go:193` (`FormatDepChange`), `internal/cli/json_formatter.go:265` (`FormatMessage`), `internal/cli/json_formatter.go:283` (`FormatRemoval`) — the three JSON objects
  - `internal/cli/dep.go:125`, `internal/cli/dep.go:195`, `internal/cli/remove.go:194`, `internal/cli/init.go:37`, `internal/cli/rebuild.go:26` — the handlers that print them
- **Proposed shape**: split the prose claim by format. Keep the five commands out of the toon driver, and give each a must-parse entry the JSON driver runs: `dep add`, `dep remove`, `remove`, `init`, `rebuild`. The cleanest shape is a second field on `conformanceDoc` — `ToonNotADocument` beside the existing exemption, or a `Formats` field naming which drivers an entry feeds — so `driveConformanceInventory` can skip per driver rather than globally, and `conformanceCoverageProblems` keeps counting each command exactly once. Add `removed` and `deps_updated` to `jsonConformanceListKeys`. Correct the comment at `conformance_test.go:1478-1479` in the same edit: the commands are prose in toon and pretty, not in JSON.

## Comment Corrections

- `internal/cli/conformance_test.go:79-80` — restates the fixture the code builds and claims what the tests cover
  OLD: `// conformanceDepGraph returns a three-task chain beside a task carrying no`
  `// dependencies, covering every branch of the focused dependency view.`
  NEW:

- `internal/cli/conformance_test.go:130-131` — restates the two lines below it
  OLD: `// conformanceStatusTask builds a task, stamping the closed time a terminal`
  `// status carries.`
  NEW:

- `internal/cli/conformance_test.go:141-142` — restates the fixture and claims which branch a test reaches
  OLD: `// conformanceLoneTask returns a childless task, the branch where a status`
  `// command moves nothing but its target.`
  NEW:

- `internal/cli/conformance_test.go:147-148` — restates the fixture and claims which branch a test reaches
  OLD: `// conformanceStatusFamily returns a parent and its only child, the branch`
  `// where a status command cascades along the hierarchy.`
  NEW:

- `internal/cli/conformance_test.go:174-176` — restates the fixture and makes a cardinality claim ("every field-selection document") that ordinary additive change falsifies
  OLD: `// conformanceSelectedTask returns an open, parentless task carrying a`
  `// description and two notes: the seed of every field-selection document, and`
  `// of the selection that names only fields it does not carry.`
  NEW:

## Spec Defects

### S1: §3.2's prose exemption reads format-independent but holds only for toon and pretty
- **Claim**: §3.2 "Output that stays prose" — "`dep add`, `dep remove`, `remove`, `init`, and the general-purpose messages. These are confirmations of a command the caller issued: the caller already knows what it asked for and the exit code says whether it worked."
- **Observed**: under `--json` none of these is prose today and none becomes prose under this work. `JSONFormatter.FormatDepChange` (`internal/cli/json_formatter.go:193`) emits `{"action","task_id","blocked_by"}`, `JSONFormatter.FormatMessage` (`internal/cli/json_formatter.go:265`) emits `{"message":…}` for `init` and `rebuild`, and `JSONFormatter.FormatRemoval` (`internal/cli/json_formatter.go:283`) emits `{"removed":[…],"deps_updated":[…]}`. The section sits beside §3.1, which is explicitly scoped to "a standard TOON reader", and §4.2, which says JSON moves with toon — neither states what §3.2 means for JSON. This phase's implementer read it as format-independent and excluded all five from the JSON conformance driver (`internal/cli/conformance_test.go:1478-1480`), which is F2 above.
- **Read**: spec stale — the code is right and predates this work; the section is TOON/pretty-scoped in fact and needs a scoping clause saying so, since JSON already returns objects for these commands and continues to.
