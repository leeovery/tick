# Review Tracking: Same-Second Tasks Sort By ID - Gap Analysis

## Findings

### 1. The duplicate-sequence report tells the user their tasks lost authoring order when in the common case they lost nothing

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §5.2 (A duplicate-sequence doctor check), colliding with §4.2 (Creation date stays above the sequence) and §5.1 (Task ID is the absolute final sort term)

**Problem**:
A shared creation sequence arrives through a branch merge, and two branches are almost never worked in the same wall-clock second. Where the two tasks record different creation dates, those dates decide the order outright — the shared number is never consulted, and the tasks list in exactly the order they were authored. The diagnostic the user gets says otherwise, flatly: the tasks sharing a number have lost their authoring order relative to one another, and restoring it means editing the sequences in `tasks.jsonl`. So the routine merge — the scenario this whole section exists for — produces a warning that overstates a healthy file and sends the user to hand-edit the file tick treats as its source of truth, for nothing, with a live chance of getting that edit wrong. And the user who genuinely did lose ordering, because both tasks landed in one second, is handed the identical message with nothing in it to tell the two situations apart.

**Proposal**:
Make what the report claims conditional on the creation dates also tying, which is what the rest of the specification already decides: a recorded creation date outranks the sequence and by that ranking duplicate-sequence damage is confined to within a single second (§4.2), and the fallback to ID order among the group is what a duplicate does at worst (§5.1). The report sentence is the only place the loss is stated unconditionally. The alternative that also fits is teaching the check to separate groups that tie on the creation second from those that do not and word each differently; it was not preferred because it re-scopes a check built as a faithful mirror of the duplicate-identity check, for a distinction the user can read off the tasks the report already names.

**Current**:
"What the report tells the user is which tasks share a number, on which lines, and that those tasks have lost their authoring order relative to one another; restoring it means editing the sequences in `tasks.jsonl` so they differ."

**Proposed Text**:
"What the report tells the user is which tasks share a number and on which lines. Sharing a number costs the authoring order only where the tasks also record the same creation second — there the group falls back to ID order among themselves; where their recorded creation dates differ, those dates decide and the listed order is unaffected (§4.2). Where the order was lost, restoring it means editing the sequences in `tasks.jsonl` so they differ."

**Resolution**: Approved
**Notes**: Applied to §5.2 as staged. The derivation is the record's own — §4.2 puts `created` above `seq`, so a duplicate costs nothing unless the seconds tie too; the sentence was claiming more than the rule allows.

---

### 2. What limits the damage when a clock steps backwards between two creations

**Source**: Specification analysis
**Category**: Contradiction
**Move**: settled
**Priority**: Important
**Affects**: §4.2 (Creation date stays above the sequence), colliding with §7.2 (Accepted risks)

**Problem**:
When a machine's clock steps back a second or more between two task creations, those two tasks come back the wrong way round — their recorded dates differ, so the dates decide and nothing later in the sort is ever consulted. The specification states in one breath that the sequence never gets to speak for exactly this reason, and in the next that the final task-ID term bounds the damage when it happens; the ID term sits below the sequence, so if one is unreachable so is the other. The accepted-risk register states the same risk as surviving with nothing catching it. The two readings size the same failure differently, and whoever plans this work reads either that the misordering is already mitigated and needs no further thought, or that it is an open hole worth a guard the design deliberately does not carry. The user meets it as two tasks silently listed in the wrong order after a clock correction, with neither the sort nor any diagnostic mentioning it.

**Proposal**:
Say what is actually bounded. The ID term cannot bound a backwards step, because the pair's dates differ and the sort stops there; what is bounded is the blast radius — only the pair straddling the step is affected, every other pair is ordered on keys the step did not touch, and the result is still one defined order rather than a plan-dependent one. §7.2 already records the risk in those terms and §1.3's totality guarantee is what makes the surviving order defined, so §4.2 is the loose statement of the three.

**Current**:
"A backwards step of ≥1s between two task creations on an NTP-synced machine is rare enough to trade against an everyday import benefit, and the final ID term (§5.1) bounds the damage when it happens."

**Proposed Text**:
"A backwards step of ≥1s between two task creations on an NTP-synced machine is rare enough to trade against an everyday import benefit, and the misordering it produces is confined to the pair straddling the step: every other pair is ordered on keys the step did not touch, and the result is still one defined order rather than a plan-dependent one (§5.1)."

**Resolution**: Approved
**Notes**: Applied to §4.2 as staged. The derivation is the record's own — §4.2's own ranking puts the sort's stopping point at `created` when the dates differ, and §7.2 already records the risk as unmitigated.

---

## Observations

- §8.5 requires existing ordering tests to stay green unchanged, while §3.1 names an existing storage assertion (`migrate_test.go:702`) that changes with this work; the two sit in different sections with nothing linking them.
- §2.1 calls the sequence a monotonic number never renumbered afterwards, while §2.2 frees the highest number for reuse after a removal; the monotonicity holds within a file's current task set, not across the assignment history.
- §5.2 does not say where the new check falls in the order `tick doctor` prints its results.
- §3.3 adds an ordinal to the `dependencies` table without saying whether it joins the table's stated pair primary key, so a record listing the same blocker twice behaves as it does today.
