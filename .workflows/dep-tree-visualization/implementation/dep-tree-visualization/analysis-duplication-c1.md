AGENT: duplication
FINDINGS:
- FINDING: Duplicate recursive box-drawing tree renderer in PrettyFormatter
  SEVERITY: medium
  FILES: internal/cli/pretty_formatter.go:246, internal/cli/pretty_formatter.go:328
  DESCRIPTION: writeCascadeTree and writeDepTreeNodes implement the same recursive box-drawing tree pattern independently. Both iterate child nodes, compute isLast, select connector characters, write prefixed lines, compute child prefix with vertical bar or space, and recurse. The structural logic is ~18 lines each with identical control flow. They differ only in node type (cascadeNode vs DepTreeNode) and how the line content is formatted.
  RECOMMENDATION: Extract a generic writeTree helper parameterized by a function that renders a single node's text and returns its children. Both writeCascadeTree and writeDepTreeNodes would become thin wrappers passing their node-specific rendering logic. This could use a callback approach or a small interface with Text() and Children() methods.
- FINDING: DepTreeTask and RelatedTask are structurally identical types
  SEVERITY: low
  FILES: internal/cli/format.go:88, internal/cli/format.go:150
  DESCRIPTION: RelatedTask{ID, Title, Status string} and DepTreeTask{ID, Title, Status string} have identical fields and types. They were introduced by separate tasks (show command vs dep tree) for conceptually different purposes but carry the same data shape. Similarly, jsonRelatedTask (json_formatter.go:42) and jsonDepTreeTask (json_formatter.go:307) duplicate the same JSON serialization struct.
  RECOMMENDATION: Consider unifying into a single type (e.g. TaskSummary or TaskRef) used by both show and dep tree features. The JSON counterpart would likewise collapse to one struct. However, this is low-impact given only 3 fields per struct, and separate types provide clearer intent. Consolidation is optional.
SUMMARY: One medium-severity finding: the recursive box-drawing tree rendering logic in PrettyFormatter is duplicated between cascade transitions and dep tree output, with ~18 lines of identical control flow that could be extracted into a shared helper. One low-severity structural type duplication (DepTreeTask/RelatedTask) is noted but optional to address.
