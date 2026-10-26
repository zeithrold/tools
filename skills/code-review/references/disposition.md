# Finding disposition

A round produces candidates, not findings. Turning one into the other is a
separate step with a different goal: discovery is generous, confirmation is
skeptical. Skipping the second step is how a review becomes a list of plausible
worries that the author cannot act on.

## Phase 1: discovery

Each lens looks for anything that could be a defect and writes candidates
down. Generosity is correct here — the ratio of candidates to confirmed defects
is normally high, and a lens that suppresses doubts to look precise loses the
finding that mattered.

A candidate is only useful if it carries:

- the path and symbol at the pin;
- the precondition or trigger that gets execution there;
- the consequence if it happens (who is harmed, what state is wrong);
- the lens reasoning in one sentence, so confirmation can attack the reasoning.

A candidate without a trigger is not a candidate; it is a feeling about the
diff. Keep those in the note, out of the findings.

## Phase 2: confirmation

Each candidate goes to an explicit attempt to refute it on the pinned revision:

1. Read the exact path at the pin, not the diff hunk and not an earlier
   revision. Follow the call chain far enough to see whether a guard, default or
   earlier validation already prevents the failure.
2. Reproduce where the environment allows: the failing input, the boundary case,
   the second concurrent writer, the retry. Reproduction at the pin outranks any
   amount of reading.
3. Search for the specific guard named in the candidate. If the guard exists, the
   candidate is refuted and the reason is recorded — refutations are output, not
   waste, because they stop the next round from re-litigating the same question.
4. If neither reading nor reproduction settles it, the candidate stays
   unconfirmed and says what observation would settle it.

Confirmation runs in the same context or a fresh one; what it must not be is the
same reasoning with more confidence. If a second reviewer is doing the
confirmation, it receives the candidate and the pin, not the first reviewer's
conclusion.

Concurrent confirmation is safe — confirmed candidates do not change each
other's state. Rewriting the code is not: fixes wait until the round's results
are collected.

## States

| State | Means | Reporting |
| --- | --- | --- |
| Confirmed | Traced or reproduced on the pin | A finding, with the smallest fix and its evidence |
| Suspected | Plausible, not established | A risk, with the observation that would settle it and whose environment can produce it |
| Refuted | A guard or test already prevents it | One line in the round summary, so it is not raised again |
| Unknown | The needed evidence does not exist (blocked, not run, claimed only) | An evidence gap that may block acceptance on its own |

Do not compress these into a score. "Medium confidence" tells the author nothing
about what to do; "confirmed by the missing `WHERE revoked_at IS NULL` on line
88 of the delete, reproduced in the scratch schema" does.

## Categories

The category decides who acts, not how bad it is.

**intent gap** — a perfect implementation of the written requirement still would
not serve the user's need: the requirement is stale, underspecified, or the
implementation faithfully follows an assumption nobody ratified. Also covers
behavior nobody asked for, when the missing decision is scope rather than
craft. Fix: the requirement owner decides; no patch settles it.

**bad plan** — the requirement is right and the code matches the plan, but the
approach cannot satisfy the requirement. Typical shapes: retrying a
non-idempotent write, optimistic UI without a rollback path, rotation designed
as read-then-write across two requests, a per-component decision that inverts a
global invariant. Adding branches makes it worse. Fix: revise the approach and
re-plan the affected work items.

**patch** — intent and approach hold and the implementation is defective at a
specific path. Fix: the change described in the finding, then re-review the new
pin.

**defer** — real, but outside the accepted scope of this change, or in adjacent
code the change exposes. It needs a named owner, a follow-up reference and
enough reproduction detail that the owner can act without this conversation.
"Out of scope" alone is a silent drop with better manners.

Severity and category are independent axes. A deferred credential-logging
defect is severe and out of scope at once; report both facts instead of choosing.

## Synthesis

The coordinator — not a lens — combines the round:

- Merge duplicates by (path, failure), keeping the sharpest evidence and
  mentioning that two lenses reached it independently.
- Never merge findings that have different consequences. Two failure modes in
  one function are two findings.
- Resolve lens disagreement by re-checking the disputed claim, not by voting or
  meeting in the middle. If the evidence is unavailable, the finding is
  suspected and the disagreement is stated.
- Order by consequence for the user or system, not by how confident the finding
  reads.
- Name what was inspected and found clean. This tells the next round where the
  remaining risk actually sits; silence is indistinguishable from absence.
- Keep the round report short enough that every line is read. A review nobody
  finishes has zero findings in practice.

## Worked example (synthetic)

Change: token refresh moved from the web client into a shared `auth-shared`
service, with a new `refresh_tokens` table and a rotation path. Pin
`4f2c1ab`. Two lenses: security and authorization; concurrency, persistence and
recovery. Round budget: two.

Discovery and confirmation:

| Candidate | Lens | Outcome | Why |
| --- | --- | --- | --- |
| Refresh token written to the new table as plaintext | security | Confirmed · patch | `auth-shared/store.go:61` writes the value it received; the old session store hashed it, so a database read now grants sessions |
| Logout deletes only the session row | security | Confirmed · patch | Logout handler traced; `refresh_tokens` untouched, so a logged-out device refreshes forever. Existing tests assert the session row only, so all checks stay green |
| Token accepted from query string | security | Refuted | `handler.go:28` reads the header before any query fallback; the test at `handler_test.go:96` shows the query path refusing |
| Two concurrent refreshes create two live pairs | concurrency | Confirmed · bad plan | Rotation is read-then-insert with no transaction and no conditional delete on the old row. Wrapping the insert in a transaction fixes nothing: both requests read the same valid row. The approach needs compare-and-swap on the old token |
| Migration adds a not-null column without default | concurrency | Confirmed · patch | Applied once to a scratch schema with rows present; fails on non-empty tables. Needs a default plus backfill, retested with existing rows |
| Revocation and issuance may use different stores | concurrency | Suspected | Trace suggests two stores; a two-request race in a real environment would settle it. Not established, not claimed |

Classified after the round:

- **intent gap:** the plan never decided what happens to tokens already issued by
the old path. The implementation picked "invalidate", which signs mobile users
out at deploy. Nothing in the code is wrong against the assumption; the
assumption was never ratified. Route to the plan owner, not the author.
- **defer:** `admin/**` logs the token at debug level. Real and severe, outside
this pin, already known. Owner: platform security. Reproduction and the log line
number travel with the defer.

Disposition: changes required. Patches 1, 2 and 5 go to the author; the rotation
design returns to planning before more code lands; the migration needs its
backfill before round 2; the scope question goes to the plan owner and blocks
acceptance until answered; the admin logging issue leaves this review with an
owner. Round 2 re-pins the new revision and re-checks only the affected paths
plus the rotation design — not the whole diff.

What kept validation green: unit tests mock the store, so plaintext storage and
the two-writer race are invisible; only the migration failure was observable
outside a real database, and it was never run on a populated schema.

## When nothing is confirmed

Report "no confirmed findings" with the lenses run, the pin and the inspected
paths, plus whatever stayed suspected or unknown. A clean result from a real
round is a useful output. A clean result from a review that never read the
callers, never ran a check and never attempted a refutation is a claim about the
reviewer, not the code — say which one happened.
