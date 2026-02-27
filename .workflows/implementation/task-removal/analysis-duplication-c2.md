AGENT: duplication
FINDINGS:
- FINDING: Double parseRemoveArgs + empty-ID validation between handleRemove and RunRemove
  SEVERITY: medium
  FILES: internal/cli/app.go:221-225, internal/cli/remove.go:151-155
  DESCRIPTION: Both handleRemove (app.go:221-225) and RunRemove (remove.go:151-155) call parseRemoveArgs on the same args and perform the identical len(ids)==0 check with the same error string "task ID is required. Usage: tick remove <id> [<id>...]". In the non-force path, parseRemoveArgs runs twice and the empty-ID guard runs twice. The error message string is duplicated verbatim. If the usage text changes, both locations must be updated.
  RECOMMENDATION: Have handleRemove pass pre-parsed IDs to RunRemove (or a lower-level function) so parsing and validation happen once. Alternatively, extract the error message into a package-level variable or const so it cannot drift between the two call sites.
- FINDING: Double store open and executeRemoval in non-force path
  SEVERITY: medium
  FILES: internal/cli/app.go:228-238, internal/cli/remove.go:157-173
  DESCRIPTION: When --force is not set, handleRemove opens the store, calls store.Mutate with executeRemoval(computeOnly=true) to compute the blast radius, closes the store, then calls RunRemove which opens the store again and calls store.Mutate with executeRemoval(computeOnly=false). This means the store is opened and closed twice, and the full validation + descendant expansion in executeRemoval runs twice on the same data. The computeOnly=true path already computes everything needed; the computeOnly=false path recomputes all of it before also performing the actual mutation.
  RECOMMENDATION: Restructure so the non-force path reuses the blast radius computation. One approach: have handleRemove pass a signal (or the pre-validated IDs and removeSet) to RunRemove so it can skip re-validation and re-expansion. Another approach: fold the mutation into handleRemove itself after confirmation succeeds, avoiding the second RunRemove call entirely and only calling RunRemove for the --force path. This eliminates the double store open and double executeRemoval.
SUMMARY: The c1 high finding (parallel computeBlastRadius vs Mutate callback) was resolved by consolidating into executeRemoval with a computeOnly flag. The remaining duplication is that the non-force path still runs the full pipeline twice: parseRemoveArgs, empty-ID validation, store open, and executeRemoval are all invoked in handleRemove and then again in RunRemove.
