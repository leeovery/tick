AGENT: architecture
FINDINGS:
- FINDING: list --parent does not resolve partial IDs, unlike every other ID-accepting parameter
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/cli/list.go:66, /Users/leeovery/Code/tick/internal/cli/list.go:169-181
  DESCRIPTION: The `--parent` flag in `parseListFlags` normalizes the ID to lowercase via `task.NormalizeID()` (line 66) but never resolves it through `store.ResolveID()`. The raw normalized value is passed directly to SQL (`WHERE id = ?` at line 172). This means `tick list --parent a3f` fails to find `tick-a3f1b2`, whereas `tick create --parent a3f`, `tick update --parent a3f`, `tick dep add a3f ...`, and every other ID-accepting parameter in the implementation correctly resolves partial IDs. The specification states "Applies everywhere an ID is accepted: positional args, --parent, --blocked-by, --blocks", and while `--parent` on create/update does resolve, `--parent` on list does not. This is a consistency gap in the API surface.
  RECOMMENDATION: In `RunList`, resolve `filter.Parent` through `store.ResolveID()` before entering the `store.Query()` closure. The pattern is already established in `RunUpdate` (lines 209-219) and `RunCreate` (lines 161-166). Add `filter.Parent, err = store.ResolveID(filter.Parent)` after opening the store and before calling `store.Query()`.
- FINDING: Cycle 1 fixes verified correct
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/show.go:72-86, /Users/leeovery/Code/tick/internal/storage/store.go:272-337
  DESCRIPTION: The two cycle 1 findings have been correctly addressed. (1) `queryShowData` now selects `type` in the SQL query, scans it into `typePtr`, and populates `data.taskType`. The `showData` struct has a `taskType` field, and `showDataToTaskDetail` correctly maps it to `task.Task.Type`. (2) `ResolveID` now performs both the exact-match and prefix-search within a single `s.Query()` call, acquiring the shared lock only once. Both fixes are sound.
  RECOMMENDATION: No action needed.
SUMMARY: One high-severity consistency gap: `list --parent` does not resolve partial IDs, unlike every other ID-accepting parameter in the codebase. Cycle 1 fixes for the missing type column in show queries and the double-lock in ResolveID are confirmed correct. Otherwise the architecture is clean -- domain/storage/CLI boundaries are well-separated, the Formatter interface composes cleanly across three implementations, and seam quality between task executors is good.
