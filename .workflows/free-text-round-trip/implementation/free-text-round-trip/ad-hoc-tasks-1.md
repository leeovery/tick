# Ad Hoc Tasks: Free Text Round Trip

## Task 1: The Terminal Stops Denying Dependencies That Exist
placement: phase 3

**Problem**: `tick dep tree` tells a person there are no dependencies in projects that have them. Seed one task blocked by an ID no task carries and the terminal prints `No dependencies found.`, while the same command in the machine format prints the edge and `chains: 1, longest: 1, blocked: 1`. A dependency cycle produces the same split. The two outputs of one command give a person and an agent opposite answers about whether work is blocked, and `tick doctor` — which has checks for both states — agrees with the agent.

The cause is the same one task 3-7 fixed for the machine formats, left standing in the terminal. `BuildFullDepTree` sets `Message` when the root set is empty (`internal/cli/dep_tree_graph.go:169-172`) without consulting `Unrooted`, and `PrettyFormatter.formatFullDepTree` returns that message on the same condition (`internal/cli/pretty_formatter.go:322-325`). Task 3-7 taught the machine formats to cover every participant but deliberately left the terminal alone, since §4.1 held its output fixed; the consequence is that the terminal now denies what its sibling formats report. §4.1 was amended this session to say that a figure the terminal reports incorrectly is corrected rather than preserved, which covers this: the sentence is not a shape being protected, it is an answer that is false.

**Solution**: Make the empty-answer condition consult every participant, not only the roots. `BuildFullDepTree` sets `Message` only when there is nothing to report at all — no roots and no unrooted trees — and the terminal draws the unrooted trees alongside the rooted ones, so a cycle or a dangling blocker renders rather than vanishing. The rendering is the existing tree drawing applied to more input; no new visual form is introduced. A project genuinely free of dependencies still prints `No dependencies found.` exactly as today.

**Outcome**: One command gives one answer. A person reading `tick dep tree` and an agent reading `tick dep tree --toon` in the same project agree about whether anything is blocked, and neither is told a project with a cycle is dependency-free.

**Do**:
- `internal/cli/dep_tree_graph.go:169-172` — set `Message` only when `len(roots) == 0 && len(unrooted) == 0`, so a graph whose participants are all blocked no longer reports itself empty.
- `internal/cli/pretty_formatter.go:322-338` — render `result.Unrooted` after `result.Roots`, through the existing tree drawing, and return `result.Message` only when both are empty. The summary line is unchanged; it already counts every participant.
- Re-pin the terminal assertions that encode today's behaviour: `internal/cli/dep_tree_test.go`'s subtest asserting the cycle fixture still prints the sentence, and any sibling asserting the same for a dangling blocker.
- Leave the machine formats, `Roots`, `Unrooted` and the counts exactly as task 3-7 left them — this task changes only which of them the terminal draws and when the sentence fires.

**Acceptance Criteria**:
- [ ] A project whose only dependency is a task blocked by an unresolvable ID renders that edge in the terminal and does not print `No dependencies found.`
- [ ] A project whose only dependencies form a cycle renders those tasks in the terminal and does not print the sentence
- [ ] A project with no dependency relationships at all still prints `No dependencies found.` and nothing else
- [ ] The terminal's summary line is unchanged on every input
- [ ] Root-reachable projects render byte-identically to before this task
- [ ] The toon and JSON dep-tree documents are unchanged on every input
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

**Tests**:
- `"it renders a dangling blocker in the terminal"` — one task blocked by an unresolvable ID; stdout carries the edge and not the sentence
- `"it renders a cycle in the terminal"` — the existing cycle fixture; stdout carries the cycle's tasks and not the sentence
- `"it still prints the sentence for a project with no dependencies"` — unchanged from today
- `"it leaves rooted terminal output unchanged"` — the A → B → C fixture and the diamond render exactly as before
- `"it leaves the machine dep-tree documents unchanged"` — the toon and JSON assertions from tasks 3-2 through 3-7 pass untouched
