TASK: tick-core-8-2 -- Extract post-mutation output helper from create.go and update.go

ACCEPTANCE CRITERIA:
- create.go and update.go no longer contain the post-mutation output logic inline
- Both commands produce identical output to before (quiet mode prints ID, normal mode prints full task detail)
- The helper is the single source of truth for mutation command output

STATUS: Complete

SPEC CONTEXT: The specification (section "Mutation Command Output") states that `tick create` and `tick update` output full task details (same format as `tick show`), TTY-aware, with `--quiet` outputting only the task ID. Both commands share identical post-mutation output behavior, making a shared helper the natural DRY extraction.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:16-30` -- `outputMutationResult` function with exact signature from plan
  - `/Users/leeovery/Code/tick/internal/cli/create.go:173` -- `return outputMutationResult(store, createdTask.ID, fc, fmtr, stdout)`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:201` -- `return outputMutationResult(store, updatedID, fc, fmtr, stdout)`
- Notes: Implementation matches the plan precisely. The function signature is `outputMutationResult(store *storage.Store, id string, fc FormatConfig, fmtr Formatter, stdout io.Writer) error`. The logic handles quiet mode (print ID only) and normal mode (queryShowData -> showDataToTaskDetail -> FormatTaskDetail -> Fprintln). Neither `create.go` nor `update.go` contain any inline references to `queryShowData`, `showDataToTaskDetail`, `FormatTaskDetail`, or `fc.Quiet` -- confirmed via grep. The only callers of the shared query/format pipeline are `helpers.go` (via `outputMutationResult`) and `show.go` (via `RunShow`).

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/helpers_test.go:203-289` -- `TestOutputMutationResult` with 3 subtests:
    1. Quiet mode outputs only the ID (line 204)
    2. Non-quiet mode outputs full task detail with ID, title, and formatted fields (line 232)
    3. Non-existent task ID returns error (line 269)
  - `/Users/leeovery/Code/tick/internal/cli/create_test.go:465-501` -- Create command tests verify full output (line 465) and quiet mode (line 487)
  - `/Users/leeovery/Code/tick/internal/cli/create_test.go:724` -- Quiet mode with relationships
  - `/Users/leeovery/Code/tick/internal/cli/update_test.go:244-290` -- Update command tests verify full output (line 244) and quiet mode (line 274)
- Notes: The task explicitly states "Existing tests for RunCreate and RunUpdate continue to pass with no changes" and "Existing tests for quiet mode continue to pass with no changes." The helper itself has dedicated unit tests covering both code paths and the error case. The existing integration-level tests in create_test.go and update_test.go exercise the helper indirectly through end-to-end command execution. Test balance is good -- not over-tested (no redundant assertions) and not under-tested (all acceptance criteria verified).

CODE QUALITY:
- Project conventions: Followed. Helper placed in `helpers.go` alongside other shared helpers (`openStore`, `parseCommaSeparatedIDs`, `applyBlocks`). Unexported function -- correct for internal package. Doc comment provided (consistent with other helpers in the file despite being unexported). Table-driven tests with subtests per golang-pro skill.
- SOLID principles: Good. Single responsibility -- the function handles only output. Open/closed -- new output behavior can be changed in one place. DRY -- eliminates the duplicated 16-line output block from two files.
- Complexity: Low. Linear code path with one branch (quiet vs non-quiet). No loops, no nesting.
- Modern idioms: Yes. Uses `io.Writer` interface for testability. Standard Go error propagation.
- Readability: Good. Function name `outputMutationResult` clearly communicates intent. Parameters are well-typed. The code flow is immediately obvious.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. This is a clean, minimal DRY refactoring that matches the plan exactly.
