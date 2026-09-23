AGENT: standards
FINDINGS: none
COMMENT_CORRECTIONS:
- internal/storage/store.go:173 — the "full flow" line restates the body, and the change made it incomplete: the numbering of unnumbered tasks between mutate and write is missing. The numbering is the one thing a caller of the exported Mutate cannot see from the signature, so the replacement states it as the contract.
  OLD: // The full flow: lock -> read JSONL -> freshness check -> mutate -> atomic write -> update cache -> unlock.
  NEW: // Tasks fn returns without a creation sequence are assigned one before the write.
SUMMARY: Checked every decision in the specification against the implementation and found no drift. That covers:
- the seq field and how it round-trips
- new-task numbering and backfill above the file's highest sequence, in ParseJSONL and Store.Mutate
- the v3 cache: the seq column and the dependency ordinal
- both list-family ORDER BY clauses ending on seq and then id
- show's children ordered by created, seq, id, and its blockers by ordinal
- the warning-severity DuplicateSeqCheck and where it is registered
- both README sentences and their prose assertions

The §8 verification requirements are met by tests whose fixtures break the ties the spec requires them to break. Existing ordering tests are unchanged. The build, vet, golangci-lint and the full suite are green.

The one spec-versus-convention conflict is the RunDoctor doc comment. The spec said its check enumeration and count should grow with the new check; code-quality.md's cardinality and restatement rules won instead, and the enumeration was removed. That was the right call.
