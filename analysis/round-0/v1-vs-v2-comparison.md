# Implementation Comparison: V1 (Sequential) vs V2 (Agent-Based)

A comprehensive analysis comparing two implementations of the TIC core plan, built from the same specification and plan files using two different workflow approaches.

- **V1** (`implementation-v1` branch): Sequential single-agent approach. The main Claude session executed all tasks directly, one after another, within a continuous context window.
- **V2** (`implementation-v2` branch): Agent-based approach. An orchestrator dispatched fresh executor sub-agents per task, followed by independent reviewer sub-agents, with approval gates between tasks.

Both implementations follow the same 23 tasks across 5 phases for "Tick" — a task tracker CLI with JSONL/SQLite dual storage, dependency management, and multiple output formats.

---

## Hard Numbers

| Metric | V1 (Sequential) | V2 (Agent+Review) |
|---|---|---|
| Go files | 45 | 45 |
| Total lines | 8,343 | 15,035 |
| Implementation lines | ~2,892 | ~3,425 (+18%) |
| Test lines | ~5,451 | ~11,610 (+113%) |
| Test:impl ratio | 1.88:1 | 3.39:1 |
| Test cases (RUN count) | 410 | 602 |
| Packages tested | 3 | 6 |
| go vet | Clean | Clean |
| All tests pass | Yes | Yes |

The implementation code only grew 18%. Nearly all the additional 6,700 lines are tests.

### Code Volume by Package

| Package | V1 Impl | V1 Test | V2 Impl | V2 Test |
|---|---|---|---|---|
| `cmd/tick` | 18 | 0 | 17 | 168 |
| `internal/cli` | 1,940 | 3,533 | 2,157 | 8,453 |
| `internal/storage` | 647 | 1,016 | 708 | 2,077 |
| `internal/task` | 287 | 702 | 357 | 933 |
| **Total** | **2,892** | **5,251** | **3,239** | **11,631** |

---

## Layer-by-Layer Verdict

| Layer | Winner | Key Differentiator |
|---|---|---|
| **Task Model** | **V2** | V1 has a Unicode bug (`len()` vs `utf8.RuneCountInString()` for title validation). V2 has a cohesive constructor, consistent case normalization, better error messages. |
| **Storage** | **V2** | V2's `tryOpen()` probes table existence to catch schema corruption. Sub-packages enforce boundaries. Transaction rollback tested. V1 has better DRY lock helpers. |
| **CLI Commands** | **V2** | V1 uses SQL string interpolation (`fmt.Sprintf("t.status = '%s'", status)`) — a maintenance hazard. V2 uses parameterized queries. V2 also has `unwrapMutationError`, blocks dedup, unknown flag detection. |
| **Formatters** | **V2** | V2 has 3x the test code with exact string matching, systematic per-format coverage, JSON validity cross-checks. V1 has cleaner interface design (no `interface{}` stats param). |
| **Structure** | **V2** | Binary integration tests, sub-packages for storage backends, per-operation cache lifecycle. V1 has better file separation in the task package. |

---

## V2's Critical Advantages

These are areas where V2 is measurably better and the differences matter for correctness.

### 1. Real Bug in V1: Unicode Title Validation

```go
// V1 task.go:71 — counts bytes, not characters
if len(trimmed) > 500 {

// V2 task.go:99 — counts runes (characters)
if utf8.RuneCountInString(trimmed) > maxTitleLength {
```

A title with 500 CJK characters = 1,500 bytes. V1 wrongly rejects it. V2 correctly accepts it.

### 2. SQL Safety

```go
// V1 list.go:234 — string interpolation (maintenance hazard)
fmt.Sprintf("t.status = '%s'", status)

// V2 list.go:128-129 — parameterized queries (safe by construction)
where = append(where, "status = ?")
queryArgs = append(queryArgs, flags.status)
```

V1's string interpolation is protected by whitelist validation, but this pattern is a maintenance hazard — if validation is ever relaxed or bypassed, it becomes a direct SQL injection vector.

### 3. Schema Corruption Detection

V2's SQLite layer probes each table with `SELECT 1 FROM X LIMIT 0` before use (`sqlite.go:213-238`). V1 would silently fail on a structurally corrupt (but openable) database. V2 even tests this with a valid-SQLite-wrong-schema scenario (`sqlite_test.go:839-874`).

### 4. Case Normalization

V2 normalizes IDs in dependency validation, dep add/rm, and transition lookups. V1 does raw string comparison in several places, meaning mixed-case IDs would cause failures.

### 5. Blocks Deduplication

V2 prevents duplicate `blocked_by` entries in both create and update. V1 blindly appends, allowing the same dependency to appear multiple times.

### 6. Test Thoroughness

V2 has 602 test cases vs V1's 410, a 3.39:1 test-to-code ratio vs 1.88:1. Specific areas where V2 tests what V1 doesn't:

- Multi-byte Unicode titles
- Transaction rollback verification
- Lock-held verification from inside callbacks
- Schema corruption with valid SQLite
- Exact error message assertions on every invalid transition
- Post-mutation persistence verification
- Field ordering in JSONL output
- Newlines in descriptions
- Indentation format verification for JSON output
- Cross-method JSON validity checks

---

## V1's Genuine Advantages

### 1. DRY Lock Acquisition Helpers

V1 extracts lock acquisition into `acquireExclusiveLock()`/`acquireSharedLock()` helpers. V2 duplicates ~45 lines of lock code across 3 methods (`Mutate`, `Query`, `ForceRebuild`).

```go
// V1 store.go:80-106 — extracted helper
func (s *Store) acquireExclusiveLock() (*flock.Flock, error) {
    fl := flock.New(s.lockPath)
    ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
    defer cancel()
    locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
    if err != nil || !locked {
        return nil, fmt.Errorf("could not acquire lock...")
    }
    return fl, nil
}

// V2 store.go:90-104, 163-177, 216-230 — same block copy-pasted 3x
```

### 2. Fully Typed Formatter Interface

V1's `Formatter` uses dedicated structs for every method parameter with compile-time type safety and no domain coupling. V2 uses `interface{}` for stats (losing compile-time safety) and `task.Status` in `FormatTransition` (coupling presentation to domain).

```go
// V1 — fully typed, decoupled
type Formatter interface {
    FormatStats(w io.Writer, data StatsData) error
    FormatTransition(w io.Writer, data TransitionData) error // plain strings
}

// V2 — interface{} and domain coupling
type Formatter interface {
    FormatStats(w io.Writer, stats interface{}) error                    // runtime assertion
    FormatTransition(w io.Writer, id string, old, new task.Status) error // domain type
}
```

### 3. Consistent Error String Convention

V1 uses lowercase verb-phrase error strings per Go convention throughout. V2 mixes styles including `"failed to open cache db"` and `"Could not acquire lock"` (capitalized — violates Go convention).

### 4. Task Domain File Organization

V1 separates the task package into three files by concern: `task.go` (model), `dependency.go` (cycle detection), `transition.go` (state machine). V2 puts everything in a single 358-line `task.go`.

### 5. Printf-Style VerboseLogger

V1's `Log(format string, args ...any)` is more ergonomic than V2's `Log(msg string)` which forces callers to `fmt.Sprintf` before every call.

### 6. TransitionResult Struct Return Type

V1 returns a `TransitionResult` struct from `Transition()`. V2 returns naked multiple values `(oldStatus, newStatus, err)`. The struct is more extensible — adding a field requires no caller changes.

### 7. Co-located Presentation Data Types

V1 defines all presentation types (`TaskListItem`, `TaskDetail`, `TransitionData`, `DepChangeData`, `StatsData`) alongside the `Formatter` interface in `format.go`. V2 scatters them: `showData` in `show.go`, `StatsData` in `toon_formatter.go`, `TaskRow` in `list.go`.

### 8. Init Cleanup on Failure

V1 removes the `.tick/` directory if `WriteFile` fails during init. V2 leaves an orphaned directory.

---

## Root Cause Analysis: Why V1 Produced Better Code in Specific Areas

### Root Cause Distribution

| Root Cause | Count | % of V1 Advantages |
|---|---|---|
| **B. Holistic Design** — cross-layer visibility | 8 | 57% |
| **A. Context Continuity** — seeing previous code | 2 | 14% |
| **C. Incremental Refinement** — conventions evolving | 1 | 7% |
| **D. Coincidental** — not approach-dependent | 3 | 22% |

### The Dominant Failure Mode: Holistic Design (57%)

The single agent could design an interface knowing its callers, design return types knowing their consumers, and structure packages knowing what would go in each file. When agent A designs an API and agent B consumes it in the multi-agent approach, neither has full context about the other's constraints. This produces:

- `interface{}` instead of concrete types (stats parameter)
- Domain types leaking into presentation layers (`task.Status` in formatter)
- Bare parameters instead of extensible structs (transition, dep change)
- Scattered type definitions (presentation types spread across files)

### Detailed Attribution

| # | V1 Advantage | Root Cause | Explanation |
|---|---|---|---|
| 1 | DRY lock helpers | A. Context Continuity | Saw duplication opportunity because prior method was in context |
| 2 | Consistent error conventions | C. Incremental Refinement | Convention established early, carried forward consistently |
| 3 | Printf-style VerboseLogger | B. Holistic Design | Designed logger API knowing storage layer's formatting needs |
| 4 | Fully typed Formatter interface | B. Holistic Design | Designed interface, data types, and implementations as cohesive unit |
| 5 | TransitionResult struct return | B. Holistic Design | Knew how CLI would consume the return value |
| 6 | 3-file task domain separation | B. Holistic Design | Incremental file creation as concerns emerged one-by-one |
| 7 | Co-located presentation structs | A. Context Continuity | All types defined while interface was fresh in context |
| 8 | Init cleanup on failure | D. Coincidental | General coding discipline, not approach-dependent |
| 9 | Struct params for formatter methods | B. Holistic Design | Cross-layer visibility: interface + callers designed together |
| 10 | Value-type formatter parameters | B. Holistic Design | Designed interface knowing callers would construct complete structs |
| 11 | Exit-code return from Run() | B. Holistic Design | Designed main.go and Run() together (debatable advantage) |
| 12 | Explicit writer injection | B. Holistic Design | Constructor designed knowing testing requirements |
| 13 | MarshalJSONL utility | D. Coincidental | Minor utility, not structurally significant |
| 14 | Integration-style CLI tests | D. Coincidental | Testing philosophy choice, not approach-dependent |

---

## Bridging the Gap: Applying V1's Strengths to the Multi-Agent Approach

The sequential approach's speed advantage is real but comes with a critical cost: context compaction. V1 hit compaction limits 3-4 times during implementation, creating risk of session pollution — where summarization of previous work introduces confusion and context loss. The agent-based approach eliminates this by giving each task a fresh context window, but loses the holistic design visibility that produced V1's better code in specific areas.

The challenge is clear: **how do we get the benefits of fresh-context execution (V2) while preserving the cross-cutting architectural awareness that continuous context (V1) provides naturally?**

The answer lies in explicitly passing forward what the sequential agent got for free.

### What the Current Workflow Skills Are Missing

Analysis of the executor agent definition (`implementation-task-executor.md`), reviewer agent definition (`implementation-task-reviewer.md`), and orchestrator skill (`technical-implementation/SKILL.md`) reveals these gaps:

**1. No "Codebase Integration" instruction in the executor**

The executor is told to "Explore codebase — understand what exists" but is NOT told to:
- Search for existing helper functions that could be reused
- Look for patterns established by previous tasks
- Ensure new code integrates with existing abstractions
- Avoid duplicating logic that already exists elsewhere
- Match error message conventions established in existing code

**2. No "Cross-boundary awareness" for interface design tasks**

When a task involves creating an interface, the executor has no instruction to:
- Read the callers that will consume the interface
- Read the implementers that will fulfill it
- Design for extensibility across those boundaries
- Use concrete types over `interface{}`

**3. The reviewer doesn't check for DRYness or codebase cohesion**

The reviewer's 5 dimensions (spec conformance, acceptance criteria, test adequacy, convention adherence, architectural quality) do not include:
- "Is there duplicated code that should be extracted?"
- "Are existing helpers/patterns being reused?"
- "Is new code consistent with conventions established by previous tasks?"
- "Does the interface design maintain proper layer boundaries?"

**4. No "convention handoff" between tasks**

The orchestrator commits after each task but does NOT pass any "here's what conventions were established in previous tasks" context to the next executor. Each agent starts cold.

**5. The reviewer is scoped too narrowly for cross-cutting concerns**

V2's own failures (the `interface{}` stats parameter, the capitalized error string) weren't caught by the reviewer because it was scoped to "task scope only" — it only checks what's in the current task, not cross-cutting concerns.

---

## Recommendations

### 1. Add "Codebase Integration" Section to the Executor Agent

In `implementation-task-executor.md`, after step 5 ("Explore codebase"), add explicit instructions:

```
5b. **Integrate with existing code** — before writing new code:
   - Search for existing helper functions, utilities, or patterns that solve
     similar problems. Reuse them instead of creating duplicates.
   - If you need to create a new helper/abstraction, check whether similar
     logic already exists elsewhere. If it does, extract and reuse.
   - When implementing an interface or API boundary, read BOTH sides: the
     callers that will use it AND the implementers that will fulfill it.
   - Match error message conventions (casing, prefix style) established
     in existing code.
   - Match naming conventions (file names, function names, type names)
     established in existing code.
```

**Addresses:** Context Continuity failures (DRY helpers, convention matching). This is the equivalent of what the sequential agent got for free by having previously-written code in context.

### 2. Add "Codebase Cohesion" as 6th Review Dimension

In `implementation-task-reviewer.md`, add:

```
### 6. Codebase Cohesion
Does the new code integrate well with the existing codebase?
- Is there duplicated logic that should be extracted into a shared helper?
- Are existing helpers/patterns being reused where applicable?
- Are naming conventions consistent with previously-implemented code?
- Are error message conventions consistent (casing, wrapping style)?
- If an interface was created, does it use concrete types (not interface{})?
  Does it avoid importing domain types into presentation layers?
- Are shared data types co-located with the interfaces they serve?
```

**Addresses:** The reviewer failing to catch duplication, convention drift, and interface design anti-patterns.

### 3. Add "Architecture Context" to Task Normalisation

In `task-normalisation.md`, extend the template:

```
TASK: {id} — {name}
PHASE: {N} — {phase name}

ARCHITECTURE CONTEXT:
{Summary of key interfaces, types, and patterns established by
previously completed tasks in this phase. Include: interface
signatures, data type locations, helper functions, error conventions,
file organization patterns.}

INSTRUCTIONS:
{all instructional content from the task}
```

The orchestrator populates "ARCHITECTURE CONTEXT" by scanning the codebase after each completed task and noting key patterns. This simulates the "context continuity" that the sequential agent had naturally.

**Addresses:** Holistic Design failures (the dominant 57% category). Gives each fresh executor the "what exists and how it's structured" picture.

### 4. Add Orchestrator "Architecture Snapshot" Step

In the task loop (`task-loop.md`), after a task is approved and committed, add:

```
## E. Update Architecture Context (after commit)

After committing an approved task, briefly scan the committed code to note:
- New interfaces or type definitions created
- New helper functions or shared utilities created
- Error message convention used (casing, wrapping style)
- File organization patterns chosen

Carry this context forward to the next executor invocation as
ARCHITECTURE CONTEXT in the task normalisation template.
```

**Addresses:** The mechanism that feeds Recommendation 3. Without this step, the architecture context has no source of truth.

### 5. Add "Convention Consistency" to Code Quality Reference

In `code-quality.md`, add:

```
## Convention Consistency

When adding to an existing codebase:
- **Error strings**: Match the casing and wrapping style of existing errors.
  Go convention: lowercase, no "failed to" prefix, verb-phrase
  (e.g., "opening database: %w").
- **Interface parameters**: Use concrete types, not interface{}. Use structs
  for 3+ parameters. Use value types unless pointer semantics are needed.
- **File organization**: Follow the project's existing pattern for splitting
  files. If the project uses one-concern-per-file, don't consolidate.
- **Helpers**: Before writing a new helper, search for existing ones.
  After writing a new helper, consider if other code could use it.
```

**Addresses:** Incremental Refinement failures (error conventions drifting) and provides concrete guardrails.

### 6. Phase-Boundary Review

Add a lightweight review step in the orchestrator that triggers after all tasks in a phase complete:

```
## Phase Boundary Review

After completing all tasks in a phase, before proceeding to the next:
1. Scan for code duplication across all files changed in this phase
2. Check for convention inconsistencies (error messages, naming, file org)
3. Identify shared helpers that should be extracted
4. If issues found, create a small refactoring commit before proceeding
```

**Addresses:** The periodic "step back and look at the whole picture" that the sequential agent did naturally as code accumulated in its context window.

---

## Addressing V2's Own Failures

### The `interface{}` stats parameter

**Root cause:** The stats command agent and the formatter interface agent didn't coordinate. The stats agent needed to pass data to a formatter but didn't know what type the formatter expected.

**Fix:** Recommendation 2 (reviewer checks for `interface{}`). Recommendation 3 (architecture context tells the executor what types exist).

### The capitalized error string

**Root cause:** No convention guidance. The executor didn't search for existing error messages to match.

**Fix:** Recommendation 1 (match error conventions) and Recommendation 5 (explicit error string guidance in code-quality.md).

### The duplicated lock code

**Root cause:** The agent wrote each method independently without extracting the common pattern. The code-quality.md mentions "Rule of Three" for DRY but the agent didn't apply it.

**Fix:** Recommendation 1 (search for existing helpers before creating duplicates). Making the instruction explicit rather than relying on general principles.

### The init cleanup omission

**Root cause:** The agent didn't think about the failure path.

**Fix:** Adding "verify cleanup on failure paths" to code-quality.md would help. The reviewer could catch it under "architectural quality" if told to check error/cleanup paths specifically.

---

## Overall Verdict

**V2 is the better implementation.** The margin is meaningful but not overwhelming. V2's advantages are structural and correctness-related: a real Unicode bug fixed, SQL injection-safe queries, better corruption recovery, more robust ID handling, and dramatically more thorough tests. V1's advantages are organizational and ergonomic: cleaner file decomposition, DRY-er code in a few spots, a more type-safe formatter interface. V1's advantages are fixable in a single refactor session; V2's test coverage gap would be systemic to close.

The agent-based approach produced measurably better code despite being slower. Fresh context per task meant deeper focus. The review stage caught issues that a streaming sequential approach would miss. The cost was real (slower execution, ~2x wall-clock time), but the quality improvement is tangible.

The path forward is clear: keep the agent-based approach but close the holistic design gap by explicitly passing architectural context forward between tasks and checking for integration quality in reviews. The 6 recommendations above target the specific failure modes identified in this analysis.
