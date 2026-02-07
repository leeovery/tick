# Root Cause Analysis: Why V3 Chose String Timestamps

## What the Spec Says

The task 1-1 spec says:

> All timestamps use ISO 8601 UTC format (`YYYY-MM-DDTHH:MM:SSZ`)

And in the acceptance criteria:

> Task struct has all 10 fields with **correct Go types**

The specification (`tick-core.md`) is more detailed but similarly format-focused:

> All timestamps (`created`, `updated`, `closed`) use ISO 8601 format in UTC:
> - Format: `YYYY-MM-DDTHH:MM:SSZ`
> - Example: `2026-01-19T10:00:00Z`
> - Always UTC (Z suffix), never local time with offset

The SQLite schema in the spec stores timestamps as `TEXT`:

```sql
created TEXT NOT NULL,
updated TEXT NOT NULL,
closed TEXT
```

And every JSONL example in the spec shows timestamps as strings:

```jsonl
{"id":"tick-a1b2","title":"Task title","status":"open","priority":2,"created":"2026-01-19T10:00:00Z","updated":"2026-01-19T10:00:00Z"}
```

**Assessment**: The spec never mentions `time.Time`. It describes timestamps purely in terms of their serialized string format. The acceptance criterion "correct Go types" is the only thing that could steer toward `time.Time`, and it is ambiguous -- `string` is technically a "correct Go type" for ISO 8601 data, even if it is unconventional. The spec's emphasis on format, the SQLite TEXT columns, and the JSONL string examples could all be read as implying string storage. An experienced Go developer would know that `time.Time` is idiomatic and use it regardless, but the spec text alone does not mandate it.

## Prompt Differences That Could Have Influenced the Decision

The `workflow-skill-diff.md` analysis documents the exact V2-to-V3 executor changes. The relevant differences:

### 1. V3's expanded "Explore codebase" step

V2's step 5 was three brief bullets: "Read files and tests related to the task's domain / Identify patterns / Check for existing code."

V3's step 5 is dramatically expanded with 7 bullets, including:

> - **Skim the plan file** to understand the task landscape -- what's been built, what's coming, where your task fits. Use this for awareness, not to build ahead (YAGNI still applies)

This instruction gives V3 awareness of task 1-2 (JSONL storage), which stores tasks as JSON strings. An agent reasoning about "what's coming" could decide: "If timestamps will be serialized to JSON strings in task 1-2 anyway, why not store them as strings from the start?"

### 2. V3 receives the plan file as input

V2 received 5 inputs. V3 receives 7 inputs including the plan file path. V2 had no plan-level context -- it only knew about its own task.

### 3. V3's statelessness declaration

V3 adds: "You are stateless -- each invocation starts fresh." This might have subtly shifted the agent toward simpler, more self-contained design choices (strings are simpler than `time.Time` + parsing).

### 4. What did NOT differ

The code-quality.md file is identical in both versions. It is generic ("DRY," "SOLID," "YAGNI," "inject dependencies") and says nothing about concrete types, `time.Time`, or timestamp handling. The golang-pro project skill is the same for both runs. Neither the executor prompt nor the code-quality file explicitly instructs "use idiomatic Go types for domain concepts."

## Did Plan-Skimming Cause It?

**Plausible but not proven.** Here is the causal chain:

1. V3's executor was told to "skim the plan file to understand the task landscape."
2. The plan file shows task 1-2 is "JSONL storage with atomic writes" -- the next task after 1-1.
3. The task 1-2 spec says: "serialize each Task as a single JSON line" and "round-trips all task fields without loss."
4. If timestamps are `time.Time`, JSONL serialization requires either:
   - Custom `MarshalJSON`/`UnmarshalJSON` to produce `YYYY-MM-DDTHH:MM:SSZ` format, or
   - Relying on `time.Time`'s default RFC3339 marshaling (which adds fractional seconds and is slightly different from the spec format).
5. If timestamps are `string`, JSONL serialization is zero-effort -- `encoding/json` handles it natively.

An agent optimizing for "easy JSONL round-tripping" (informed by plan awareness) could rationally choose strings. V2's executor, which had no plan context and therefore no awareness that JSONL was next, had no such incentive.

However, V1 (sequential generation, which presumably had full plan awareness throughout) still chose `time.Time`. This weakens the argument that plan awareness alone causes the string choice.

**Counter-evidence**: The spec explicitly shows the SQLite schema using `TEXT` for timestamps and the JSONL format using string timestamps. Even without plan awareness, an agent reading the spec could reason toward strings. The spec's emphasis on ISO 8601 format strings may have been sufficient.

## Did the Code Quality File Prevent It in V2?

**No.** The code-quality.md is identical between V2 and V3 and contains nothing about timestamp types:

```
## Principles
### DRY: Don't Repeat Yourself
### SOLID
### Cyclomatic Complexity
### YAGNI
## Anti-Patterns to Avoid
- God classes
- Magic numbers/strings
- Deep nesting (3+)
- Long parameter lists (4+)
- Boolean parameters
```

No mention of "use concrete types," "prefer time.Time for timestamps," or anything about Go-specific type choices. The code quality file is language-agnostic.

The golang-pro project skill (`SKILL.md`) also does not mention timestamp types. Its reference files show `CreatedAt time.Time` in an example struct (in `project-structure.md`), but this is buried inside a general reference file, not prominently instructed. Whether the executor loaded this reference during task 1-1 execution depends on whether it judged the "Project Structure" topic relevant -- and for a first task creating a simple struct, it might not have.

## Evidence from the Code

V3's `task.go` has this comment on the struct:

```go
// Task represents a single task in Tick.
// All timestamps use ISO 8601 UTC format (YYYY-MM-DDTHH:MM:SSZ).
type Task struct {
    ...
    Created     string   `json:"created"`
    Updated     string   `json:"updated"`
    Closed      string   `json:"closed,omitempty"`
}
```

The comment directly echoes the spec language ("ISO 8601 UTC format (YYYY-MM-DDTHH:MM:SSZ)"). This suggests the agent read the spec's timestamp format description and interpreted it as the type definition -- "the spec says these are ISO 8601 strings, so I'll store them as strings."

V3 also provides `DefaultTimestamps()` which pre-formats timestamps:

```go
func DefaultTimestamps() (created, updated string) {
    now := time.Now().UTC().Format(time.RFC3339)
    return now, now
}
```

This is a "format once at creation, store as string" pattern. The agent converted `time.Time` to `string` at the earliest possible point and never looked back.

V2's code, by contrast, stores `time.Time` and defers formatting to serialization:

```go
now := time.Now().UTC().Truncate(time.Second)
return &Task{
    ...
    Created:     now,
    Updated:     now,
}
```

No comment explains V2's type choice -- it simply uses the Go-idiomatic type without justification, suggesting the V2 agent treated `time.Time` as the obvious default.

## Conclusion: Model Variance vs Prompt-Induced

**Primarily model variance, amplified by a prompt-level contributing factor.**

The evidence does not support a clean "the prompt caused it" narrative. Here is why:

1. **The spec is genuinely ambiguous.** It describes timestamps in terms of their ISO 8601 string format, shows them as strings in JSONL and TEXT in SQLite, and never mentions `time.Time`. The acceptance criterion "correct Go types" could mean either `time.Time` or `string`. Two reasonable developers could disagree.

2. **The prompt difference is suggestive but not deterministic.** V3's plan-skimming instruction could have nudged the agent toward thinking about serialization before type design, but V1 (which had full plan context) still chose `time.Time`. The plan-skimming instruction is not a smoking gun.

3. **The golang-pro skill has weak signal.** The `project-structure.md` reference shows `CreatedAt time.Time` in an example, but this is one line in a multi-page reference file. It is not prominently instructed, and the agent may not have loaded this reference at all for task 1-1.

4. **The code-quality file is irrelevant.** It is identical between versions and says nothing about types.

5. **The fundamental cause is how the agent interpreted the spec.** V3's agent read "All timestamps use ISO 8601 UTC format" and chose the literal representation (string). V2's agent read the same spec and chose the semantic representation (`time.Time`). This is a model-level interpretation difference -- the kind of variance you see between runs even with identical prompts.

**The prompt amplified the variance** in two ways:

- V3's plan-skimming gave the agent a reason to prefer strings (easier JSONL serialization), providing a rationalization for the non-idiomatic choice.
- V3's expanded exploration instructions added cognitive load, potentially reducing the attention budget for "what is the most idiomatic Go type here?" The agent spent more tokens on codebase exploration and plan awareness, leaving less for the design decision itself.

**If forced to assign percentages**: ~70% model variance (the spec is ambiguous enough that different runs produce different interpretations), ~30% prompt-induced (plan awareness provided a subtle nudge toward strings, and increased cognitive load reduced the chance of the agent reaching for the idiomatic default).

**The real failure is not in the prompt but in the absence of a guardrail.** Neither the spec, the code-quality file, nor the golang-pro skill explicitly says "use `time.Time` for timestamps in Go." Adding a single line to the golang-pro skill -- "Always use `time.Time` for temporal data, not strings" -- or to the spec -- "Use `time.Time` internally; format to ISO 8601 only at serialization boundaries" -- would have prevented this decision regardless of model variance.
