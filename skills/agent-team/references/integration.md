# Integration

Delegation ends with the parent owning the result. Parallel workers each verify a
fragment; nobody has verified the system that the fragments compose into. This
file is that step: what to collect, what to re-check, and how to integrate
without turning a set of green unit results into an unverified whole.

## 1. Collect

Collect every unit against its own brief, including failed and partial ones, and
write the claims down before judging them. A small table is enough:

| Unit | Claimed | Scope kept? | Check claimed | Status |
| --- | --- | --- | --- | --- |
| A | retry with bounded backoff, deduplicated | yes | `go test ./services/webhooks` | reported green |
| B | manual retry endpoint returns 202 | no — edited CHANGELOG.md | `go test ./services/api/...` | reported green |

Three things to inspect in each report:

- **Scope.** Paths outside the exclusive list are events even when the edit is
good. They also mean the unit's verification did not cover what it touched.
- **Interface drift.** A signature, response field or error code that is near the
  frozen one is a change, and consumers built against the frozen one are now
  wrong in a way no test will announce.
- **Evidence quality.** A check that ran proves something about that unit at that
  revision in that environment. It says nothing about the integrated revision, and
  nothing about behavior the check never exercises.

## 2. Verify

Verification is not a second reading of the reports. It is deliberately applied
to the thing you will hand over:

1. **Re-run the checks that span units** at the integrated revision. Unit suites
   rarely test the seam, which is exactly where independent work meets.
2. **Re-check each consequential claim** where its cost is proportional: the
   duplicate-prevention claim becomes a two-delivery test, the compatibility claim
   becomes a request with the old shape, the recovery claim becomes an injected
   failure.
3. **Inspect the paths nobody owned** — shared files, generated artifacts,
   configuration — because each unit was told it did not own them and so each
   believed someone else verified them.
4. **Re-run what a unit reported as blocked** if the missing piece is now
   available; otherwise keep it blocked, not passed.

What you are looking for is a claim no unit was in a position to make: the seam
behavior. Two correct units can compose into an incorrect system, and the
composition is verified only by whoever made it.

## 3. Integrate

Order by dependency, interface producers first, then consumers, then docs and
changelogs. Integrate in bounded batches, and run the cross-unit checks after each
batch rather than once at the end — a failure that appears after four batches
belongs to one of four, and the attribution work costs more than the check.

Conflict resolution is an ownership decision, not a merge heuristic:

- **Shared file, two edits:** the owner's content wins; the other unit re-requests
  the change. Do not hand-merge two authors' intentions into a third version that
  neither reviewed.
- **Interface conflict:** the interface owner decides the shape; the consumer
  adapts. If neither can, the plan's interface decision was never frozen — send it
  back to planning instead of negotiating it mid-merge.
- **Contract conflict:** a unit implemented behavior the scope contract excluded,
  or skipped behavior it required. That is a plan question, not a conflict to
  resolve by keeping the smaller diff.

Keep integration reversible. Know how to undo each batch — a revert, a rollback
step, a re-run — and know which changes are one-way doors (an applied forward-only
migration, a published artifact, a deleted resource). Place batch boundaries before
the one-way steps, not after.

## Worked example (synthetic)

Continuing the `payments` webhook work: unit A owns retry and dedupe in
`services/webhooks`, unit B adds status reads and a manual retry endpoint in
`services/api/webhooks`, and the root `CHANGELOG.md` is owned by A for the round.

B returns with: endpoint returns 202, tests green, and one line added to
`CHANGELOG.md`. A returns green with a new dedupe path keyed on event and
endpoint.

Verification at the integrated revision:

| Check | Result | Meaning |
| --- | --- | --- |
| `go test ./...` | green | Both units' own tests still pass; the seam is untested |
| New seam test: fail a delivery, call B's manual retry twice | **fails** — two delivery rows, two attempts sent | B's endpoint inserts a delivery row and calls the transport directly instead of going through `webhooks.Deliver`, so it bypasses the dedupe key entirely |
| Old-shape request against the status endpoint | green | Additive change holds |
| Files changed vs exclusive lists | B edited `CHANGELOG.md` | Ownership event, not a defect |

The failure existed before integration and no unit could see it: A's dedupe is in
A's package and B's tests assert that B's row exists — both true, jointly wrong.
The green suite is not evidence about the seam.

Disposition:

1. The seam defect goes back to B with the failing test and the specific cause —
   route the retry through the interface A owns. Not "fix integration".
2. `CHANGELOG.md`: discard B's edit, re-request the line from A. B's content is
   not lost; its authorship location was wrong.
3. A owns the interface decision; if routing B's retry requires a new entry point
   on `webhooks.Deliver`, A decides the shape and B adapts.
4. The migration is forward-only and already applied in the integration
   environment, so the schema batch boundary sits before its application; the code
   batch is revertible, the schema step is recorded as one-way.
5. Nothing is merged to a shared destination on the strength of the reports. The
   integrated revision is re-verified with the seam test plus the unit suites, and
   the result is reported that way.

Report to the requester: which units shipped at which revision, which checks ran
where (unit environment vs integrated revision), what the seam test proved, that
partner-side delivery behavior is still unverified without the sandbox, the state
of any unit that failed or stopped and where its partial files are, the budget
spent, and the follow-ups handed off (the flaky `internal/queue` test A reported,
the ownership note about `CHANGELOG.md`).

## What not to do

- Do not merge each unit as it arrives and verify at the end. That converts unit
  confidence into system risk with no boundaries to blame.
- Do not resolve a semantic conflict by picking the diff that is easier to
  revert.
- Do not let the worker that produced a unit verify its own integration, and do
  not accept "it passes locally" for a claim about the integrated revision.
- Do not report the team's work as verified because every worker said its part
  was fine. The parent's report describes what the parent observed.
- Do not hide a cancelled or failed unit inside an aggregate success count. Name
  it, name its state.
