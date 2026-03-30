# qualifyCommand Leaks "tree" Subcommand to Note Parent

The `qualifyCommand` function in `internal/cli/app.go` shares the `"tree"` case across both the `"dep"` and `"note"` parent commands in its switch statement. This means `tick note tree` gets qualified as `"note tree"`, which isn't registered in `commandFlags`.

The error path still technically works — `handleNote` eventually returns "unknown note sub-command 'tree'" — but if the user passes flags, the error message becomes confusing. For example, `tick note tree --foo` produces `unknown flag "--foo" for "note tree"`, which implies "note tree" is a real command that just doesn't accept that flag. The user would reasonably wonder what flags "note tree" does accept, when the real answer is that the command doesn't exist at all.

This was surfaced during the dep-tree-visualization review. The `qualifyCommand` switch groups `"add"`, `"remove"`, and `"tree"` together for both `"dep"` and `"note"` parents. The `"add"` and `"remove"` cases are legitimate for both parents (`dep add`, `dep remove`, `note add`, `note remove`), but `"tree"` only applies to `dep`. The fix would be to scope `"tree"` so it only qualifies under the `"dep"` parent.

Impact is low — this is an unlikely user path and the command ultimately fails regardless. But the error message is misleading when it does happen.
