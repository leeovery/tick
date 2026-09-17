# Review Tracking: Free Text Round Trip - Claims Verification

## Findings

### 1. `tick show` does not render task detail through the shared mutation helper

**Source**: Tree measurement — `grep -rn 'outputMutationResult' internal/cli/ | grep -v '_test.go'`
**Category**: Source defect
**Move**: route
**Affects**: §3.1 Output that must parse (task detail row of the must-parse table)

**Problem**:
The record tells whoever builds this that the five commands returning a task record — `show`, `create`, `update`, `note add`, `note remove` — all produce it through one shared helper, `outputMutationResult`. Four of them do; `tick show` does not. It queries and renders the detail itself, inline in its own handler. Work planned as a single change at that helper leaves the one command the whole effort is named for — an agent running `tick show` to read free text — untouched, and the `show`-only field selection has no home there at all: the helper is reached only after a mutation, so a field-selected read cannot flow through it.

**Evidence**:
Claim (§3.1): "| Task detail | `show`, `create`, `update`, `note add`, `note remove` (all via `outputMutationResult`, `grep -n 'func outputMutationResult' internal/cli/helpers.go` → `helpers.go:16`) |"

```
$ grep -rn 'outputMutationResult' internal/cli/ | grep -v '_test.go'
internal/cli/create.go:277:	if err := outputMutationResult(store, createdTask.ID, fc, fmtr, stdout); err != nil {
internal/cli/update.go:409:	if err := outputMutationResult(store, updatedID, fc, fmtr, stdout); err != nil {
internal/cli/note.go:86:	return outputMutationResult(store, id, fc, fmtr, stdout)
internal/cli/note.go:142:	return outputMutationResult(store, id, fc, fmtr, stdout)
internal/cli/helpers.go:13:// outputMutationResult handles post-mutation output for create and update commands.
internal/cli/helpers.go:16:func outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error {
```

No `show.go` call site. `tick show` renders detail directly:

```
$ sed -n '52,64p' internal/cli/show.go
	data, err := queryShowData(store, id)
	if err != nil {
		return err
	}

	if fc.Quiet {
		fmt.Fprintln(stdout, data.id)
		return nil
	}

	detail := showDataToTaskDetail(data)
	fmt.Fprintln(stdout, fmtr.FormatTaskDetail(detail))
	return nil
```

The helper's own comment records the same narrower scope (`internal/cli/helpers.go:13`): "outputMutationResult handles post-mutation output for create and update commands."

Source carrying the claim: `.workflows/free-text-round-trip/discussion/free-text-round-trip.md:166` — "| Task detail | `show`, `create`, `update`, `note add`, `note remove` (all via `outputMutationResult`, `internal/cli/helpers.go:16-30`) |".

**Resolution**: Routed
**Notes**: Measurement confirmed independently. The discussion's must-parse table (line 166) repaired in place — the four mutating commands share the helper, `show` renders inline. The decision itself (task detail must parse from all five commands) is untouched by the correction. Specification §3.1 re-aligned to the corrected fact, naming the consequence for §9's field selection.

---

### 2. Imported descriptions are stored untrimmed, so the round-trip guarantee has a second hole

**Source**: Tree measurement — `grep -rn 'Description' internal/migrate/*.go internal/migrate/beads/*.go | grep -v '_test'`
**Category**: Source defect
**Move**: route
**Affects**: §2.2 The fidelity bar is byte-identity (the derivation that existing whitespace trimming cannot stand in the way, and the conclusion that trimming is out of scope as a defect)

**Problem**:
The promise made to an agent is that a value read out of `tick show` and written straight back is stored byte-for-byte as it was, and the reason whitespace trimming is said not to threaten that promise is that no stored description can carry leading or trailing whitespace in the first place. That is true only of descriptions written through `tick create` and `tick update`. `tick migrate` stores the source tool's description exactly as it arrives, with no trim anywhere on that path, so a project migrated from beads can hold descriptions that begin or end with whitespace. Read one of those out and write it back and the stored value silently changes — the guarantee fails on precisely the tasks a user did not author by hand, and the exception is nowhere written down.

**Evidence**:
Claim (§2.2): "`create` and `update` both run a description through `TrimDescription` before storing … No stored description can therefore carry leading or trailing whitespace, and the trim is idempotent over anything that came out of storage. The trimming behaviour is out of scope as a defect."

```
$ grep -rn 'Description' internal/migrate/*.go internal/migrate/beads/*.go | grep -v '_test'
internal/migrate/migrate.go:34:	Description string
internal/migrate/store_creator.go:81:			Description: mt.Description,
internal/migrate/beads/beads.go:31:	Description  string `json:"description"`
internal/migrate/beads/beads.go:125:		Description: issue.Description,
```

```
$ grep -rn 'TrimDescription' internal/ --include='*.go' | grep -v '_test'
internal/cli/create.go:214:			Description: task.TrimDescription(opts.description),
internal/cli/update.go:193:		trimmed := task.TrimDescription(*opts.description)
internal/cli/update.go:342:				tasks[i].Description = task.TrimDescription(*opts.description)
internal/task/task.go:193:func TrimDescription(desc string) string {
```

The import write path builds the stored task with the provider's raw value:

```
$ sed -n '75,86p' internal/migrate/store_creator.go
		newTask := task.Task{
			ID:          id,
			Title:       mt.Title,
			Status:      status,
			Priority:    priority,
			Description: mt.Description,
			Created:     created,
			Updated:     updated,
			Closed:      closed,
		}
```

Source carrying the claim: `.workflows/free-text-round-trip/discussion/free-text-round-trip.md:112` — "Determined by the write paths' own behaviour: `create` and `update` both run the value through `TrimSpace` before storing … so no stored description can carry leading or trailing whitespace."

**Resolution**: Pending
**Notes**:
