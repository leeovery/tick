# Review Tracking: Free Text Round Trip - Claims Verification

## Findings

### 1. Dash-leading note text is not a shape this project's notes take

**Source**: Tree measurement — `grep -c '"notes"' .tick/tasks.jsonl`
**Category**: Source defect
**Move**: route
**Affects**: §10.1 (the defect), and the scope call in §10.1 that folds the dash-leading fix into this work unit

**Problem**:
The specification tells the reader that free text beginning with a dash is "the shape a large share of this project's own notes take" — that the write-side refusal is already biting tick's own notes routinely, and that this frequency is why the fix belongs in this work unit rather than in a follow-up. Neither part of that is true of the tree. Tick's own task store holds 165 tasks and zero notes, so no note of this project's takes any shape at all. Across every tick store on this machine — 14 of them, 178 notes — not one note begins with a dash. The refusal itself is real and is correctly described (`ValidateFlags` rejects any dash-leading bare argument), but the evidence the specification offers for how often it bites is false, and a reader weighing the scope of §10 against its cost is weighing it against a frequency that does not exist.

**Evidence**:
Claim (§10.1, verbatim): "An agent reads a note off a task, corrects a typo, and writes it back. If the note begins with a dash — `- read the header`, the shape a large share of this project's own notes take — the command refuses it:"

```
$ grep -c '"notes"' .tick/tasks.jsonl
0

$ wc -l < .tick/tasks.jsonl
165
```

Cross-store scan of every tick store on this machine:

```
$ python3 -c "
import json,glob
tot=0;dash=0;stores=0
for p in sorted(glob.glob('/Users/leeovery/Code/*/.tick/tasks.jsonl')):
    stores+=1
    for line in open(p):
        line=line.strip()
        if not line: continue
        try: o=json.loads(line)
        except Exception: continue
        for n in (o.get('notes') or []):
            tot+=1
            if (n.get('text') or '').startswith('-'): dash+=1
print('stores:',stores,'notes:',tot,'dash-leading:',dash)
"
stores: 14 notes: 178 dash-leading: 0
```

The refusal mechanism itself measures as described, so only the frequency claim fails:

```
$ grep -n 'func ValidateFlags' -A 30 internal/cli/flags.go
117:func ValidateFlags(command string, args []string, flags CommandFlags) error {
...
122-		if !strings.HasPrefix(arg, "-") {
123-			continue
...
136-		def, ok := cmdFlags[arg]
137-		if !ok {
138-			return fmt.Errorf("unknown flag %q for %q. Run 'tick help %s' for usage.", arg, command, helpCommand(command))
```

Source carrying the claim: `.workflows/free-text-round-trip/discussion/free-text-round-trip.md`, section `## Write Side Input` → `### Context`, line 680, verbatim:

```
$ sed -n '680p' .workflows/free-text-round-trip/discussion/free-text-round-trip.md
An agent reads a note off a task, corrects a typo, and writes it back. If the note begins with a dash — `- read the header`, the shape a large share of this project's own notes take — the command refuses it:
```

**Proposed Text**:

**Resolution**: Routed
**Notes**: Measurement confirmed independently — tick's own store holds zero notes, so the frequency claim is unsupported. The claim is colour rather than ground: the refusal mechanism it illustrates measures exactly as described, and the decision to fix the free-text argument problem rests on that mechanism plus the round-trip hole, neither of which moves. Repaired in place in the discussion (line 680) and re-aligned in specification §10.1 — the dash-leading shape is now described as what a bulleted note naturally takes, which needs no frequency evidence.
