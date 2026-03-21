# Assignee Field

Add an assignee field to tasks. Not unreasonable on the surface, but Tick is single-user by design. The JSONL storage model creates conflicts on concurrent multi-user edits — especially updates to existing lines — which makes this problematic without rethinking the storage layer.

The main value of assignees only materialises in a multi-user context. For a solo developer, knowing who owns a task is redundant. If multi-user ever becomes a goal, this would need to land alongside a storage model that handles concurrent writes safely, likely the event-sourced approach rather than the current line-update JSONL format.

Could potentially have lightweight value even in single-user mode for delegation tracking — noting that a task is "waiting on someone else" — but tags or notes already cover that use case without a dedicated field.
