---
status: in-progress
created: 2026-03-09
cycle: 1
phase: Gap Analysis
topic: unknown-flags-silently-ignored
---

# Review Tracking: unknown-flags-silently-ignored - Gap Analysis

## Findings

### 1. Central validator must know which flags consume a value argument

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Design > Flow (step 2), Command Flag Inventory

**Details**:
The spec says to "validate `rest` against [valid flags] -- error on any unknown flag" but does not address how the validator distinguishes flag values from flags. Many per-command flags consume the next argument as a value (e.g., `--priority 3`, `--status open`). The validator needs to know which flags are boolean (no value) vs value-taking (consume next arg) so it can skip the value position when iterating. Without this, the validator would either (a) misidentify values as unknown flags, or (b) fail to detect unknown flags that happen to follow value-taking flags.

The flag inventory table lists flag names but does not annotate which are boolean vs value-taking. An implementer must either infer this from the codebase or make assumptions about the export format from commands.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added Flag Metadata subsection to Design.

---

### 2. `ready` and `blocked` commands missing from Requirement 3 command list

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Requirements (item 3)

**Details**:
Requirement 3 enumerates "all commands" that must be covered but omits `ready` and `blocked`. These commands appear in the Command Flag Inventory table, so the intent is clear, but the requirements list is inconsistent with the inventory. An implementer following only the requirements section would miss them.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added ready and blocked to Requirements item 3.

---

### 3. `version` and `help` commands not addressed

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Requirements (item 3), Command Flag Inventory, Design > Dispatch Paths

**Details**:
`version` and `help` are dispatched in `app.go` before format resolution (same early-exit path as `doctor`/`migrate`). The spec doesn't mention them in either the requirements or the command flag inventory. An implementer might wonder whether `tick version --foo` or `tick help --unknown` should be validated. Currently `help` accepts `--all` as a sub-argument, which would need to be in the inventory if validation applies.

These are arguably edge cases since both are informational commands, but the spec should explicitly state they are excluded from validation (and why) so an implementer doesn't have to guess.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added Excluded Commands subsection to Design.

---

### 4. Validation mechanics for two-level commands (`dep add/rm`, `note add/remove`) underspecified

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Design > Flow, Command Flag Inventory, Error Behavior

**Details**:
The spec's flow describes: "Look up valid flags for the identified command, validate `rest` against them." For two-level commands like `dep add`, the `rest` array passed from `parseArgs` contains `["add", "tick-aaa", "tick-bbb"]` -- the sub-subcommand is embedded in `rest`, not in `subcmd`.

The spec doesn't clarify:
1. Does the central validator validate flags in `rest` for the top-level command (`dep`), or does it first extract the sub-subcommand and validate against `dep add`'s specific flag set?
2. What happens with flags between the top-level and sub-subcommand (e.g., `tick dep --blocks add tick-aaa tick-bbb`)? Currently `handleDep` takes `subArgs[0]` as the sub-subcommand and passes the rest to the handler.
3. The error message example shows `"dep add"` as the command name -- does the validator need to compose this from the two-level dispatch?

Since `dep add`, `dep rm`, `note add`, and `note remove` all accept no per-command flags, the validator's job is simpler (reject all flags in their args). But the spec should state how the lookup and validation work for these compound commands.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added Two-Level Commands subsection to Design.

---

### 5. How `parseArgs` communicates pre-subcommand unknown flag errors

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Design > Flow (step 1), Error Behavior

**Details**:
The spec says "Unknown flags before subcommand must also be rejected" and gives a specific error format. However, `parseArgs` currently returns `(globalFlags, string, []string)` with no error return value. The spec describes the desired behavior but doesn't indicate whether:
- `parseArgs` should gain an error return value
- A separate validation step after `parseArgs` should check for pre-subcommand unknown flags
- The unknown flags should be collected and returned alongside the other values

This is a design detail that an implementer would need to decide. Since `parseArgs` is called at the very start of `App.Run()`, changing its signature affects the top-level control flow.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added Pre-Subcommand Validation subsection to Design.

---

### 6. `dep add/remove` naming inconsistency in Requirement 3

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Requirements (item 3)

**Details**:
Requirement 3 lists `dep add/remove` but the actual sub-subcommand in the codebase is `rm` (not `remove`), and the Command Flag Inventory table correctly says `dep rm`. The requirements section should use `dep add/rm` to match the code and the inventory table.

**Proposed Addition**:

**Resolution**: Pending
**Notes**: Minor inconsistency. The inventory table is correct; only the requirements list uses the wrong name.
