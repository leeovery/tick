# Review Tracking: Free Text Round Trip - Claims Verification

## Findings

### 1. The empty dep-tree sentence comes from the command handler, not from a formatter

**Source**: Tree measurement — `sed -n '34,44p' internal/cli/dep_tree.go`
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §4.2 JSON moves with toon (the empty dep-tree form); bears on §8, which requires the nothing-blocked branch to come back as a structured document

**Problem**:
On the branch where nothing in the project is blocked, `tick dep tree` hands an agent the sentence `No dependencies found.` — the prose §8 replaces with an emptied document. The specification places that output inside the JSON formatter, at the line that wraps `result.Message`. That line never runs: the command handler returns before any dep-tree formatter is asked for anything, printing the sentence through the general-purpose message method, and the matching guard in the toon formatter is dead for the same reason. Repair carried out where the specification points changes nothing a caller sees — `tick dep tree` and `tick dep tree --json` keep answering in prose on the one branch an agent cannot predict. It also hides a second consequence: once the handler does route that branch through the dep-tree formatters, pretty stops printing its sentence unless it is given one, because pretty's full dep-tree rendering returns an empty string when there are no roots.

**Proposal**:
Point §4.2's citation at the code that actually emits the sentence — the `len(result.Roots) == 0` short-circuit in `runFullDepTree` (`internal/cli/dep_tree.go:38-41`), which calls `fmtr.FormatMessage` — and record that the dep-tree formatters' own message branches are unreachable and that pretty's full dep-tree rendering returns `""` for zero roots. Measurement settles it: `FormatDepTree` is only ever called with a non-empty root set or a named target, so the emptied document and pretty's retained sentence both hang off that handler branch.

**Evidence**:
Claim (specification §4.2): "A consumer parsing JSON gets the same structured answer as one parsing toon: … and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`)."

`sed -n '34,44p' internal/cli/dep_tree.go`:
```
34:// runFullDepTree builds and outputs the full dependency graph.
35:func runFullDepTree(tasks []task.Task, fmtr Formatter, stdout io.Writer) error {
36:	result := BuildFullDepTree(tasks)
37:
38:	if len(result.Roots) == 0 {
39:		fmt.Fprintln(stdout, fmtr.FormatMessage(result.Message))
40:		return nil
41:	}
42:
43:	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
44:	return nil
```

`sed -n '175,178p' internal/cli/dep_tree_graph.go` — the message exists only when the root set is empty, which is exactly the case the handler short-circuits:
```
	var message string
	if len(roots) == 0 {
		message = "No dependencies found."
	}
```

`grep -rn 'FormatDepTree(' internal/ --include='*.go' | grep -v _test` — the only call sites are the two in the handler (`dep_tree.go:43` with a non-empty root set, `dep_tree.go:61` with a named target):
```
internal/cli/json_formatter.go:360:func (f *JSONFormatter) FormatDepTree(result DepTreeResult) string {
internal/cli/toon_formatter.go:175:func (f *ToonFormatter) FormatDepTree(result DepTreeResult) string {
internal/cli/format.go:202:	FormatDepTree(result DepTreeResult) string
internal/cli/format.go:227:func (b *baseFormatter) FormatDepTree(_ DepTreeResult) string { return "" }
internal/cli/format.go:275:func (s *StubFormatter) FormatDepTree(_ DepTreeResult) string { return "" }
internal/cli/dep_tree.go:43:	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
internal/cli/dep_tree.go:61:	fmt.Fprintln(stdout, fmtr.FormatDepTree(result))
internal/cli/pretty_formatter.go:308:func (f *PrettyFormatter) FormatDepTree(result DepTreeResult) string {
```

`sed -n '360,367p' internal/cli/json_formatter.go` — the cited line 366 sits behind `result.Target == nil && result.Message != ""`, a combination the handler prevents:
```
func (f *JSONFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTreeJSON(result)
	}

	if result.Message != "" {
		return marshalIndentJSON(jsonMessage{Message: result.Message})
	}
```

`sed -n '175,182p' internal/cli/toon_formatter.go` — the toon side is dead for the same reason:
```
func (f *ToonFormatter) FormatDepTree(result DepTreeResult) string {
	if result.Target != nil {
		return f.formatFocusedDepTree(result)
	}

	if result.Message != "" {
		return result.Message
	}
```

`grep -n 'func (f \*JSONFormatter) FormatMessage' -A 2 internal/cli/json_formatter.go` — what the handler's call actually produces in JSON, the same `{"message": …}` shape the specification describes:
```
228:func (f *JSONFormatter) FormatMessage(msg string) string {
229-	return marshalIndentJSON(jsonMessage{Message: msg})
230-}
```

`sed -n '316,319p' internal/cli/pretty_formatter.go` — pretty has no sentence of its own on that branch:
```
func (f *PrettyFormatter) formatFullDepTree(result DepTreeResult) string {
	if len(result.Roots) == 0 {
		return ""
	}
```

Source check: the discussion states the shape only — "including the dep-tree empty case, where JSON currently hands back a `message` key carrying the English sentence" (`.workflows/free-text-round-trip/discussion/free-text-round-trip.md:200`), which measurement confirms. The `json_formatter.go:366` location is the specification's own; `grep -n 'json_formatter\|jsonMessage\|dep_tree.go' .workflows/free-text-round-trip/discussion/free-text-round-trip.md` returns only line 200.

**Current**:
> A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence (`grep -n 'jsonMessage{Message: result.Message}' internal/cli/json_formatter.go` → `json_formatter.go:366`).

**Proposed Text**:
A consumer parsing JSON gets the same structured answer as one parsing toon: the §7 `changed` list in place of the current `transition` object beside a `cascaded` list (`grep -n 'json:"transition"\|json:"cascaded"' internal/cli/json_formatter.go` → `json_formatter.go:276-277`), and the §8 structured empty dep-tree form in place of today's `message` key carrying the English sentence.

Where nothing in the project is blocked, that sentence reaches both machine formats without a dep-tree formatter being asked for anything: the command handler returns first, printing the general-purpose message (`sed -n '38,41p' internal/cli/dep_tree.go` → `if len(result.Roots) == 0 { fmt.Fprintln(stdout, fmtr.FormatMessage(result.Message)); return nil }`), which JSON renders as the `message` object and toon as the bare sentence. The message branches inside the dep-tree formatters themselves (`toon_formatter.go:180-181`, `json_formatter.go:365-366`) are never reached, so the emptied document has to be produced on the path the handler takes. Pretty's sentence has to survive that move: its own full dep-tree rendering returns an empty string when there are no roots (`sed -n '317,318p' internal/cli/pretty_formatter.go` → `if len(result.Roots) == 0 { return "" }`), so pretty keeps its prose on that branch only if it is handed the message there.

**Resolution**: Pending
**Notes**:
