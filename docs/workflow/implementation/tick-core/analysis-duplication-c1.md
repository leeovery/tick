AGENT: duplication
FINDINGS:
- FINDING: ReadTasks and ParseTasks near-duplicate JSONL parsing loop
  SEVERITY: medium
  FILES: internal/storage/jsonl.go:60-87, internal/storage/jsonl.go:91-112
  DESCRIPTION: ReadTasks (reads from file path) and ParseTasks (reads from []byte) contain identical scanner-based parsing logic: bufio.Scanner loop, empty-line skip, line-by-line json.Unmarshal, lineNum tracking, and error wrapping. The only difference is the io.Reader source (os.File vs bytes.NewReader). This is 20+ lines of duplicated logic.
  RECOMMENDATION: Have ReadTasks read the file into []byte then delegate to ParseTasks. This collapses 25 lines into ~5 and makes ParseTasks the single parsing implementation.

- FINDING: FormatTransition and FormatDepChange identical across ToonFormatter and PrettyFormatter
  SEVERITY: medium
  FILES: internal/cli/toon_formatter.go:165-191, internal/cli/pretty_formatter.go:129-155
  DESCRIPTION: ToonFormatter.FormatTransition and PrettyFormatter.FormatTransition produce byte-identical output (same type assertion, same fmt.Fprintf with arrow). ToonFormatter.FormatDepChange and PrettyFormatter.FormatDepChange are also identical (same type assertion, same switch on Action, same Fprintf calls). These are four methods (~50 lines total) where two formatters independently implemented the same plain-text rendering.
  RECOMMENDATION: Extract shared implementations into package-level functions (e.g. formatTransitionText, formatDepChangeText) that both ToonFormatter and PrettyFormatter call. Alternatively, embed a common base struct that provides these methods.

- FINDING: FormatMessage identical across ToonFormatter, PrettyFormatter, and StubFormatter
  SEVERITY: low
  FILES: internal/cli/toon_formatter.go:227-229, internal/cli/pretty_formatter.go:221-223, internal/cli/format.go:124-126
  DESCRIPTION: All three formatters implement FormatMessage as `fmt.Fprintln(w, msg)` -- three identical one-line implementations. While individually trivial, this is a pattern that will be copied into any future formatter.
  RECOMMENDATION: Embed a common baseFormatter struct with the FormatMessage method, or extract a standalone helper. Low priority given the method is one line, but prevents drift if the message format ever changes.

- FINDING: Stats ready/blocked SQL duplicates ReadyQuery/BlockedQuery WHERE clauses
  SEVERITY: medium
  FILES: internal/cli/stats.go:28-42, internal/cli/ready.go:7-23, internal/cli/stats.go:46-62, internal/cli/blocked.go:7-25
  DESCRIPTION: StatsReadyCountQuery duplicates the WHERE clause from ReadyQuery (same three conditions: status=open, NOT EXISTS unclosed blockers, NOT EXISTS open children). StatsBlockedCountQuery duplicates the WHERE clause from BlockedQuery. The only difference is SELECT COUNT(*) vs SELECT columns with ORDER BY. If the ready/blocked logic changes, both locations must be updated in sync.
  RECOMMENDATION: Extract the shared WHERE clause into a constant (e.g. readyWhereClause, blockedWhereClause), then compose both the list query and count query from it. For example: ReadyQuery = "SELECT ... FROM tasks t WHERE " + readyWhereClause + " ORDER BY ..."; StatsReadyCountQuery = "SELECT COUNT(*) FROM tasks t WHERE " + readyWhereClause.

- FINDING: buildReadyFilterQuery and buildBlockedFilterQuery are near-identical
  SEVERITY: low
  FILES: internal/cli/list.go:111-127, internal/cli/list.go:132-148
  DESCRIPTION: These two functions have identical structure: wrap an inner query in a SELECT, call appendDescendantFilter, conditionally append status and priority filters. The only difference is the inner query constant (ReadyQuery vs BlockedQuery) and the alias name. This is ~15 lines duplicated.
  RECOMMENDATION: Extract a shared buildWrappedFilterQuery(innerQuery, alias string, f listFilters, descendantIDs []string) function that both call with their respective inner query.

- FINDING: ID existence map construction repeated in mutation closures
  SEVERITY: low
  FILES: internal/cli/create.go:52-55, internal/cli/update.go:70-73, internal/cli/dep.go:55-58
  DESCRIPTION: The pattern `existing := make(map[string]int, len(tasks)); for i, t := range tasks { existing[t.ID] = i }` appears three times across create, update, and dep add. Each builds the same index for task lookup by ID.
  RECOMMENDATION: Extract a buildTaskIndex(tasks []task.Task) map[string]int helper. Minor but prevents subtle divergence (e.g. one using NormalizeID, another not). Currently all use raw t.ID which is consistent, but fragile.

SUMMARY: Six duplication findings (2 medium, 1 medium SQL, 3 low). The highest-impact items are the ReadTasks/ParseTasks parsing duplication in storage, the identical formatter methods across Toon/Pretty, and the ready/blocked SQL WHERE clause duplication in stats -- all of which risk logic drift when the duplicated code needs to change.
