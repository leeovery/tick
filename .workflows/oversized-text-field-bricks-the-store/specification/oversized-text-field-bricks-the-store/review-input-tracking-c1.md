# Review Tracking: Oversized Text Field Bricks The Store - Input Review

## Findings

### 1. The doctor check's name and help/README description are kept unchanged without a source decision

**Source**: No source decides this. Checked: investigation, Fix Direction → Chosen Approach ("Doctor judges each line the way tick loads it"), Fix Direction → Testing Recommendations ("If the JSONL check's name or description changes, the README's doctor 'Checks for:' list (`README.md:396`, read by `readme_samples_test.go:708`) and the `doctor` help text (`cli/help.go:217`) move with it"), Discussion, Risk Assessment.
**Category**: Unsourced decision
**Move**: route
**Affects**: 5.2 The check keeps its name

**Problem**:
After this work, doctor's JSONL check fails lines for reasons that are not syntax errors: a wrong-typed field such as `"priority":"high"`, `{}`, `null`, `[]`, or a whitespace-only line. Agents running doctor see the check's label in its output, and they see the check's one-line description in the README's doctor "Checks for:" list and in `tick help doctor`. The specification decides that the label stays `JSONL syntax` and that neither the README entry nor the help text changes. The investigation never made that call. It treats a rename or re-description as possible and records only what has to move with it if one happens. Keeping the name leaves a label and description that understate what the check now refuses. Renaming it changes user-visible output and a tested README sample. Neither side is decided in the record.

**Proposed Text**:

**Resolution**: Pending
**Notes**:

---

### 2. Description-cap boundary is not tested on the migrate route

**Source**: investigation, Fix Direction → Testing Recommendations: "Description cap: exactly-at-cap accepted, cap+1 refused on `create`, `update` and `migrate`"; Analysis H2 (the import validates through its own `MigratedTask.Validate`, separate from the CLI's checks)
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: 8.5 Description cap (§6)

**Problem**:
`tick migrate` checks issues through its own validation, which is separate from the `create` and `update` path. The tests pin the 50,000 / 50,001 boundary only on `create` and `update`. On migrate, the only test is that an over-cap issue is skipped. If the import path has an off-by-one, a legitimate issue with exactly 50,000 characters of description would be skipped during import. The user would lose that task from the migration, with a refusal naming a limit the description actually meets, and no test would catch it.

**Proposal**:
Add the boundary to the migrate test bullet: an issue with exactly 50,000 characters of description imports, and one with 50,001 is skipped with the refusal's elements in its reason. The investigation's testing recommendations settle this by naming `migrate` alongside `create` and `update` for the exactly-at-cap / cap+1 boundary.

**Current**:
- Migrate: an over-cap issue is skipped with §6.3's elements in its reason, the remaining issues import, and `--dry-run` reports the same refusal. An issue with an over-500-character or multi-line title is skipped likewise.

**Proposed Text**:
- Migrate: an issue whose description is exactly 50,000 characters imports, and one of 50,001 characters is skipped with §6.3's elements in its reason; the remaining issues import, and `--dry-run` reports the same refusal. An issue with an over-500-character or multi-line title is skipped likewise.

**Resolution**: Pending
**Notes**:

---

## Observations

- The investigation's sibling check lists four prior specifications this work contradicts: v1/tick-core's "No maximum length" for the description, increase-note-char-limit's "intentionally unbounded" exclusion, free-text-round-trip §10.3, and v1/doctor-validation's exclusion of schema checks together with its Errors #2 scope. The specification does not record that it supersedes them.
- Testing (doctor): the investigation asks for each bad-line fixture to be confirmed failing "every other command", but the specification confirms it against `tick list` only. Every command parses through the same loader, so `list` stands in for the rest.
- The relationship-check skip behaviour kept in §5.4 is already pinned by `orphaned_dependency_test.go:219` and `task_relationships_test.go:137,238`, per the investigation. Testing could name these as tests that must pass unchanged.
- The investigation's blast radius notes that `doctor.ParseTaskRelationships` (`task_relationships.go:82`) shares doctor's prefix semantics. It is test-only and has no production caller.
- §7 excludes the title and note refusals from the four-element refusal wording. The source implies that boundary only by leaving those refusals out of its fix direction. It never states it.
