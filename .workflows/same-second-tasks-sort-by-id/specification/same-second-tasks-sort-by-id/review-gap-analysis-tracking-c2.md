# Review Tracking: Same-Second Tasks Sort By ID - Gap Analysis

## Findings

### 1. The guarantee says creation time no longer bears on ordering; the sort keeps it as the dominant term

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §1.3 (What tick guarantees after the fix), colliding with §4.2 (Creation date stays above the sequence) and §7.2 (Accepted risks)

**Problem**:
The contract section tells the reader that creation time stops carrying the ordering. The sort rule says the opposite: a recorded creation date outranks the sequence, two tasks whose dates differ are ordered by those dates, and the sequence speaks only on a tie. The accepted-risk register agrees with the sort rule and names the cost — a wall clock that steps backwards by a second or more between two creations still lists those two tasks in the wrong order, and the sequence never gets to speak because the dates differ.

Both readings are on the page, and they cost in two directions. Whoever writes the user-facing promise from the guarantee tells the user that timestamps no longer decide anything, which promises away the one way a list can still come back misordered after this work. And a reader planning the change from the guarantee alone reads it as licence to drop the date from the sort, which would hand a migrated project its tasks in the order the importer happened to walk its source rather than in the chronology the source recorded — the exact trade that was deliberately made the other way.

**Proposal**:
Correct the guarantee to say what the sort rule decides: creation time stops being the *last* word, not a load it no longer carries. Dates that differ still order the tasks, the sequence speaks only where they tie, and the backwards-clock-step misordering survives. The derivation is the record's own — §4.2 argues the created-above-sequence ranking explicitly and trades against it, and §7.2 registers the surviving failure it buys, so the guarantee is the loose statement of the three.

**Current**:
"Creation time stops being load-bearing for ordering. The timestamp format is unchanged, and no stored timestamp changes value."

**Proposed Text**:
"Creation time stops being the last word on ordering: where two tasks record the same second, the sequence decides. Dates that differ still order the tasks — recorded chronology outranks the sequence (§4.2) — so a wall-clock step backwards of a second or more between two creations still misorders them (§7.2). The timestamp format is unchanged, and no stored timestamp changes value."

**Resolution**: Pending
**Notes**:

---

### 2. Nothing requires the doctor check's passing verdict to be pinned, and the mirror it is built from fails the run

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Move**: settled
**Priority**: Important
**Affects**: §8.3 (Duplicate sequences), pinning the verdict decided in §5.2

**Problem**:
A duplicate sequence is decided to be a warning: the run still passes, because branch merges produce the condition routinely and tick offers no command that would clear it. The same check is also specified as a mirror of the duplicate-identity check, which is one of the nine that fail the run. The verification list asks only that the new check report a duplicate with its line numbers and stay quiet on a clean file; nothing there says the run must still pass.

Built as the faithful mirror it is told to be, the failing verdict ships silently. Every project that has merged a branch — this one first, since its task file is git-tracked and implementation runs in worktrees — goes red to any script, agent or CI gate that reads doctor's outcome, on a file that works perfectly, clearable only by hand-editing `tasks.jsonl`. No assertion in the suite notices, then or later.

**Proposal**:
Add the verdict to the doctor bullet in §8.3: the check reports at warning severity and `tick doctor` still exits zero on a file carrying a duplicate. §5.2 already decides it and states why; only the assertion is missing, and the exit code is the one property of the check a user meets without asking for it.

**Proposed Text**:
Append a bullet to §8.3, after "The new doctor check reports a duplicate with its line numbers, mirroring `DuplicateIdCheck`'s existing tests, and reports nothing on a clean file.":

"- A duplicate does not fail the run: the check reports at warning severity and `tick doctor` still exits zero on a file carrying one (§5.2)."

**Resolution**: Pending
**Notes**:

---

## Observations

- §2.1's "the list-family sort ends on it" reads as if the sequence were the final term; §4.1 and §5.1 put the task ID after it.
- §2.2's unqualified "Clashes cannot occur" sits beside §5's demonstrated merge duplicates; the sentence's own justification (the exclusive write lock) is what scopes it to a single working copy.
- §8.3's "reports nothing on a clean file" sits beside §5.2's "returns a single passing result on a clean file"; §5.2's own vocabulary separates reporting a group from returning a pass, so the two read together.
- §5.2 describes the report as naming which tasks share a number and on which lines, without saying whether the shared number itself is printed; §2.4 makes the doctor check the sole exception to the sequence never being printed.
- §8.2's bullet for a record stripped by a binary that did not know the field is the same fixture shape as the mixed post-merge file in the bullet above it, and §7.1 puts mixed-version reasoning out of scope.
- §6 gives the substance of the README tiebreak sentence but not its wording, which §8.6's prose assertion has to match exactly.
