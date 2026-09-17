# Review Tracking: Free Text Round Trip - Claims Verification

## Findings

### 1. The README keeps agent-format samples this work replaces

**Source**: Tree measurement — `grep -n 'summary{chains,longest,blocked}:' README.md`, `grep -n 'tick-a1b2: open → in_progress\|tick-c3d4: open → done (auto)\|"from": "open"' README.md`
**Category**: Source defect
**Move**: route
**Affects**: §12.1 (the README correction), which scopes the README work to the `tick show` sample plus the new input surface

**Problem**:
After this work ships, the README still teaches output the tool no longer produces. Three samples change under this specification and none of them is named as owed a correction: the `dep tree` sample prints `summary{chains,longest,blocked}:` — the single-object header §5 replaces with top-level named fields — and it sits in the `### dep` command section, outside the Output Formats section the correction is scoped to; the Transition & Cascade samples print the arrow lines with `(auto)` that §7.2 replaces with the `changed` table; and the JSON transition sample prints the `{id, from, to}` object that §4.2 moves to the `changed` list. Someone learning the tool from the README writes a parser against shapes the tool stopped emitting — the defect §12.1 exists to prevent, left in place on three of the four samples that carry it.

**Evidence**:
Claim (§12.1): "Its Output Formats section prints worked `tick list` and `tick show` samples in the agent format. After this work the `tick show` sample shows output the tool no longer produces — its header, its tags and refs lists and its description block all change (§5, §6). The `tick list` table is library-written … and untouched by this work, so that sample stands."

```
$ grep -n 'summary{chains,longest,blocked}:' README.md
307:summary{chains,longest,blocked}:

$ sed -n '299,309p' README.md
**TOON** (flat edge list)
```
$ tick dep tree
dep_tree[2]{from,to}:
  tick-a1b2,tick-c3d4
  tick-c3d4,tick-f3e4

summary{chains,longest,blocked}:
  1,2,2
```

$ grep -n '^### \|^## ' README.md | sed -n '20,30p'
259:### `dep`
315:### `stats`
381:## Output Formats
419:### TOON (Token-Oriented Object Notation)
453:### Pretty
463:### Transition & Cascade Output
524:### JSON

$ grep -n 'tick-a1b2: open → in_progress\|tick-c3d4: open → done (auto)\|tick-f3e4: done (unchanged)\|"from": "open"' README.md
474:tick-a1b2: open → in_progress
484:  "from": "open",
503:tick-c3d4: open → done (auto)
504:tick-f3e4: done (unchanged)
```

Line 307 is inside `### dep` (line 259), not inside `## Output Formats` (line 381). Lines 474, 484, 503–504 are inside `### Transition & Cascade Output` (line 463), which is inside Output Formats and is not named by the claim.

Source carrying the same assertion: `.workflows/free-text-round-trip/discussion/free-text-round-trip.md`, "Published Documentation Owed a Correction" → Context, line 786: "**README's Output Formats section** — prints worked `tick list` and `tick show` samples in the agent format. After this work those samples show output the tool no longer produces." The enumeration of the README's stale agent-format output is the source's, and it names the same two samples.

**Proposed Text**:

**Resolution**: Routed
**Notes**: Measurement confirmed independently — the README's agent-format samples reach past `tick show` to the dep-tree summary (:307), the arrow transition (:473-475), the JSON transition (:481-486) and the cascade sample (:501-504). The discussion's inventory line repaired in place; the decision it sits under (the README is updated as part of this work) is unaffected. Specification §12.1 re-aligned with a measured table of every sample and why it changes.

---

### 2. The `tick list` sample declared sound prints a row the tool does not emit

**Source**: Tree measurement — `grep -n 'Update docs' README.md`, `grep -n 'Setup Sanctum,done,1' internal/cli/toon_formatter_test.go`
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §12.1 (the README correction), the sentence ruling the `tick list` sample correct as it stands

**Problem**:
The README's `tick list` sample ends an untyped task's row with a bare comma, where the tool writes `""` for the empty field — the library quotes an empty string rather than leaving a gap. The specification rules that sample sound and leaves it untouched, so a README updated by this work still shows a row no `tick list` invocation produces, on the one sample a reader is told to trust. The shape of the table is genuinely unchanged by this work; the sample's row content is wrong today and stays wrong.

**Proposal**:
Keep the ruling that the table's shape is untouched, and add the correction the sample's untyped row needs. The tool's own output for an empty type is `  tick-a1b2,Setup Sanctum,done,1,""` (`grep -n 'Setup Sanctum,done,1' internal/cli/toon_formatter_test.go` → `toon_formatter_test.go:31`), against the README's `  tick-d5c6,Update docs,open,3,` (`grep -n 'Update docs' README.md` → `README.md:400`).

**Evidence**:
Claim (§12.1): "The `tick list` table is library-written (`grep -n 'encodeToonSection("tasks"' internal/cli/toon_formatter.go` → `toon_formatter.go:75`) and untouched by this work, so that sample stands."

```
$ sed -n '396,401p' README.md
$ tick list
tasks[3]{id,title,status,priority,type}:
  tick-a1b2,Auth middleware,in_progress,1,feature
  tick-f3e4,Write tests,open,2,task
  tick-d5c6,Update docs,open,3,

$ grep -n 'Setup Sanctum,done,1' internal/cli/toon_formatter_test.go
31:		expectedRow1 := `  tick-a1b2,Setup Sanctum,done,1,""`

$ grep -n 'func NeedsQuoting' -A 4 ~/go/pkg/mod/github.com/toon-format/toon-go@v0.0.0-20251202084852-7ca0e27c4e8c/internal/format/format.go
27:func NeedsQuoting(s string, ctx Context) bool {
28-	if len(s) == 0 {
29-		return true
30-	}

$ grep -n 'toonTaskRow' -A 6 internal/cli/toon_formatter.go | head -8
22:type toonTaskRow struct {
23-	ID       string `toon:"id"`
24-	Title    string `toon:"title"`
25-	Status   string `toon:"status"`
26-	Priority int    `toon:"priority"`
27-	Type     string `toon:"type"`
```

The `type` field carries no `omitempty`, so an untyped task's field reaches the encoder as `""` and the library quotes it.

**Current**:
The `tick list` table is library-written (`grep -n 'encodeToonSection("tasks"' internal/cli/toon_formatter.go` → `toon_formatter.go:75`) and untouched by this work, so that sample stands.

**Proposed Text**:
The `tick list` table is library-written (`grep -n 'encodeToonSection("tasks"' internal/cli/toon_formatter.go` → `toon_formatter.go:75`), so its shape is untouched by this work and the sample keeps its form. Its untyped row is corrected: the sample ends that row with a bare comma (`grep -n 'Update docs' README.md` → `README.md:400`, `  tick-d5c6,Update docs,open,3,`) where the tool writes the library's quoted empty field (`grep -n 'Setup Sanctum,done,1' internal/cli/toon_formatter_test.go` → ``  tick-a1b2,Setup Sanctum,done,1,""``).

**Resolution**: Approved
**Notes**: Confirmed against the suite's own golden row (toon_formatter_test.go:31 → `  tick-a1b2,Setup Sanctum,done,1,""`); `toonTaskRow.Type` carries no `omitempty` and the library quotes an empty string. Landed in the same §12.1 rewrite as finding 1: the list table's shape is untouched by this work, but its printed row is wrong and is corrected while the section is rewritten.

---
