# Review Tracking: Free Text Round Trip - Claims Verification

## Findings

### 1. The README's `tick show` sample is one line longer than the range naming it

**Source**: Tree measurement — `awk 'NR>=427 && NR<=452 {printf "%d: %s\n", NR, $0}' README.md`
**Category**: Source defect
**Move**: route
**Affects**: §12.1 (documentation correction table, the `tick show` full detail row)

**Problem**:
The README sample that has to be rewritten runs from line 430 to line 450 — its description block is two lines, `  Full task description here.` and `  Can be multiple lines.`. The correction table names lines 430-449, which stops one line short. Whoever replaces the range as written leaves the second line of the old indented description block stranded in the README, directly beneath the new quoted-string form that replaces it — a published sample showing both the old and new encoding at once.

**Evidence**:
Claim (§12.1 table, `tick show` full detail row):
> | `tick show` full detail | `README.md:430-449` | header, tags, refs and description block all change (§5, §6) |

Command and output:
```
$ awk 'NR>=427 && NR<=452 {printf "%d: %s\n", NR, $0}' README.md
427: ```
428:
429: ```
430: task{id,title,status,priority,type,created,updated}:
431:   tick-a1b2,Setup auth,in_progress,1,feature,"2026-01-19T10:00:00Z","2026-01-19T14:30:00Z"
432:
433: tags[2]:
434:   backend
435:   auth
436:
437: refs[1]:
438:   https://github.com/org/repo/issues/42
439:
440: blocked_by[1]{id,title,status}:
441:   tick-c3d4,Database migrations,done
442:
443: children[0]{id,title,status}:
444:
445: notes[1]{text,created}:
446:   Discussed approach with team,"2026-01-19T14:00:00Z"
447:
448: description:
449:   Full task description here.
450:   Can be multiple lines.
451: ```
452:
```

The fenced sample opens at 429 and closes at 451; its content is 430-450. The four other ranges in the same table measure correct: `README.md:307` is `summary{chains,longest,blocked}:`, `396-401` is the `tick list` fenced sample, `473-475` is the arrow transition sample, `481-486` is the JSON transition object, and `501-504` is the cascade sample carrying `(auto)` and `(unchanged)`.

Source carrying the same claim — `.workflows/free-text-round-trip/discussion/free-text-round-trip.md:786`, the "README's worked output samples" bullet:
> `tick show`'s full detail (`README.md:430-449`: the malformed header, the tags and refs lists, the indented description block)

**Resolution**: Pending
**Notes**:
