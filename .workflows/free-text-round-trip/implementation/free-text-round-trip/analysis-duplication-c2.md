AGENT: duplication
FINDINGS: none
COMMENT_CORRECTIONS:
- internal/cli/flags.go:11 — describes only the separated spelling, false since a value-taking flag now also carries its value attached as "--flag=value"; the field name carries what survives
  OLD: // TakesValue indicates whether the flag consumes the next argument as its value.
  NEW:
- internal/cli/format.go:258 — restates the line above it ("for text-based formatters (Toon and Pretty)") and makes a cardinality claim about embedders that ordinary additive change falsifies
  OLD: // Embedded by ToonFormatter and PrettyFormatter.
  NEW:
- internal/migrate/migrate.go:28 — "normalized" is false now that MigratedTask carries a Normalize method the Provider does not call
  OLD: // MigratedTask represents a normalized task ready for insertion into tick.
  NEW: // MigratedTask represents a task ready for insertion into tick.
- internal/migrate/migrate.go:68 — tells a Provider implementer the tasks it returns are normalized, which is now the opposite of true: Engine.Run normalizes what the Provider hands it
  OLD: 	// Tasks returns all normalized tasks from the source, or an error if the source cannot be read.
  NEW: 	// Tasks returns every task from the source, or an error if the source cannot be read.
- internal/migrate/engine.go:54 — enumerates Run's steps and omits the one that changes what gets stored, so the import path's trim contract is written down nowhere
  OLD: // Run fetches tasks from the provider, validates each one, inserts valid tasks
  NEW: // Run fetches tasks from the provider, trims edge whitespace off each one's free
  NEW: // text, validates each one, inserts valid tasks
SUMMARY: No duplication candidate clears the floor this cycle; cycle 1's one finding (count-zero toon headers restating their row struct's columns) was closed by Tfree-text-round-trip-7-4, which now derives every such header from the struct tags via emptyToonSection. Seven further candidates were examined and dropped against the floor: the three formatters' parallel field projection and their fifteen selectedItems/sel.Positions pairings (guarded by TestRegisteredFieldRendering for presence, and a mis-paired constant is a typo rather than a rule that can silently diverge); the four build*Section shapes in toon_formatter.go (a dropped empty branch fails loudly in the decode and conformance suites); the StatusChange -> toonChangedRow / jsonStatusChange struct conversions (a new field breaks the conversion at compile time); the note index rendered once per machine format (pinned independently at list_show_test.go:1148 for toon and :1221 for JSON); the trim-before-store rule at create/update/note/migrate (an unrouted call site of a helper, which the floor excludes); the cascade from/to predicted by Cascades and applied by Transition (pre-existing in apply_cascades.go, untouched by this work); and the toon detail document appending its changed section ungated by the selection where JSON gates it (json_formatter.go:139 versus toon_formatter.go:100) — a real divergence, but unreachable today because only RunShow sets Fields and only outputMutationResult sets Changes, so no failure can be named for it. The conformance harness deserves note in the other direction: conformanceMustParseCommands derives its command set from the inventory rather than restating it, and conformanceCoverageProblems forces every registered command into exactly one of the must-parse, prose and out-of-scope sets, which is the composition that makes a duplicate list impossible there.
