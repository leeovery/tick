---
topic: blocked-ancestor-ready
cycle: 1
total_findings: 3
deduplicated_findings: 2
proposed_tasks: 1
---
# Analysis Report: blocked-ancestor-ready (Cycle 1)

## Summary
Three findings across two agents (duplication and architecture). The standards agent found no issues. The primary finding -- identified independently by both the duplication and architecture agents -- is that `BlockedConditions()` contains hand-written SQL that duplicates the bodies of `ReadyNoUnclosedBlockers()`, `ReadyNoOpenChildren()`, and `ReadyNoBlockedAncestor()` instead of composing from them. One low-severity test helper duplication was discarded as it does not cluster into a pattern.

## Discarded Findings
- runReady/runBlocked test helper duplication (duplication agent, low) -- isolated low-severity finding in test code only; 13 lines with a one-word difference; does not cluster with other findings and does not warrant a standalone task
