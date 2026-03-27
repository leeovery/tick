AGENT: standards
FINDINGS:
- FINDING: Focused no-deps edge case bypasses formatter for task info line
  SEVERITY: high
  FILES: internal/cli/dep_tree.go:61-64
  DESCRIPTION: When a focused task has no dependencies, the handler writes raw text (`fmt.Fprintf(stdout, "%s %s [%s]\n", ...)`) directly to stdout before calling `fmtr.FormatMessage()`. This violates the spec's requirement that "every command output goes through the formatter." For JSON output, this produces a raw text line followed by a JSON object, which is invalid JSON. For toon format, the task line is unstructured plain text mixed with a toon message. The spec's "Task with no dependencies (focused mode): Show the task itself with 'No dependencies.'" edge case should be handled entirely within the formatter.
  RECOMMENDATION: Route the no-deps focused case through `FormatDepTree` by having each formatter handle DepTreeResult where Target is set but BlockedBy and Blocks are empty (and Message is non-empty). The JSON formatter's early `Message != ""` return in FormatDepTree would need adjustment to check for Target presence first, producing a focused JSON object with the target task info and a "No dependencies." message field. Remove the raw fmt.Fprintf from the handler.
- FINDING: JSON FormatDepTree message check masks focused target info
  SEVERITY: medium
  FILES: internal/cli/json_formatter.go:358-360
  DESCRIPTION: `FormatDepTree` checks `result.Message != ""` first and returns a bare `{"message": "..."}` object, before checking `result.Target != nil`. For a focused task with no dependencies, the result has both a non-empty Message AND a non-nil Target. The message check wins, so the target task information (id, title, status) is lost from the JSON output. This contradicts the spec requirement to "Show the task itself" in the no-deps edge case.
  RECOMMENDATION: Restructure the priority in FormatDepTree: when Target is non-nil, always use the focused formatter (which should embed the message when both BlockedBy and Blocks are empty). The message-only path should only apply to full graph mode results (where Target is nil).
SUMMARY: The focused-mode no-deps edge case has two related issues: the handler bypasses the formatter for the task info line (producing invalid JSON output), and the JSON formatter's FormatDepTree would lose target info if called due to its message-first check ordering. Both stem from incomplete handling of this edge case within the formatter layer.
