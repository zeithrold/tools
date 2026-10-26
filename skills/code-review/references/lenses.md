# Review lenses

A lens is a failure-mode question plus the evidence that answers it. If you
cannot state both, you have a role title, not a lens. Lenses are chosen per
change, and a lens that finds nothing on a revision is a normal outcome — not a
reason to rerun it under a different name.

## Selecting lenses

1. List what can actually break in this revision: which boundary, which stored
   state, which consumer, which credential, which release action, which user
   visible outcome.
2. For each real risk, pick the lens whose question can find that failure, not
   the lens whose label resembles the change's subject. A migration is not
   automatically a concurrency problem; a rename in an auth module is not
   automatically a security problem.
3. One lens can cover an ordinary change. High-risk changes get two lenses whose
   questions do not overlap — the second exists because the first would not ask
   its question.
4. Add a third only when a named risk has no lens asking about it.

Two lenses asking comparable questions are one lens with extra latency, and they
make the round more likely to converge on the same blind spot.

| Change | Lens A | Lens B | Why the second is not redundant |
| --- | --- | --- | --- |
| Session refresh moved into a shared service | security and credential lifetime | concurrency, persistence and recovery | An auth trace passes while two refreshes still write one token pair |
| Schema addition with a migration | persistence, migration and recovery | contract and consumer compatibility | Recoverability and downstream compatibility fail independently |
| New background retry with backoff | concurrency and idempotency | silent failure and error propagation | Duplicated side effects and swallowed terminal errors are distinct defects |
| Manual review of a documentation policy change | intent and authority | test and verification gap | Policy text can be wrong while the checksum check is green |

For risks these lenses do not name — destructive data operations, release
automation, cost or performance claims — derive the lens before starting:

```
Failure question: what would have to be true for this change to hurt someone?
Evidence:        the observation on the pinned revision that would show it
Not covered:     what this lens cannot establish, so nobody reads silence as a pass
```

## security and authorization

Ask: which principal reaches this path, and what actually proves the principal
rather than asserting it? Can a caller who is legitimate but less privileged
obtain the same effect — another tenant, another user, another role? Does the new
code trust a value the client controls (identifier, tenant, role, redirect,
filename, header, deserialized body)? Where do credentials and tokens live, for
how long, what invalidates them, and does the new path create a second copy that
survives revocation? On failure does the path fail open or closed, and does it
log the denial without logging the secret?

Evidence: the entry point and its guard chain at the pin; the check for absent,
expired and revoked credentials, and for a different tenant; the error-path
response and its log content; where the secret is stored and how it is rotated.

Weak substitutes: a middleware import that is registered but not in this route's
chain; a unit test that constructs the handler without its guards; a green
permission test that only asserts the 403 branch of an already-authorized user.

## concurrency, persistence and recovery

Ask: what holds when two writers interleave, a request is retried, a process
dies between two writes, or a migration runs twice or half-way? Which write is
idempotent and which is compensating? Is the lock or transaction scope the same
as the scope of the invariant it protects — and does that scope survive the
service boundary? What does an operator do after a partial failure, and does the
code say what state it left behind? Is the recovery path itself tested, or only
the happy path around it?

Evidence: the write order and transaction or lock boundary; the retry key that
makes a retry safe; an injected-failure run or a trace of the unprotected window;
the recovery and rollback path with its preconditions; whether the migration is
reversible or explicitly a one-way door.

Weak substitutes: "uses a transaction" without showing the read that decides the
write is inside it; a lock taken per request when the invariant spans requests;
a rollback described in prose that no environment can execute.

## contract and compatibility

Ask: who consumes this, and what did the previous version promise them? Is the
type, nullability or optionality change expressed on every consumer path,
including serialization, dynamic keys and casts where the type is erased? Does
an error become an exception, a status, a null or a silent no-op at this
boundary? Are the compiler's view and the wire or database view the same? What
happens to unknown or legacy values from an older producer, and to a consumer
that has not been redeployed yet?

Evidence: actual consumers enumerated from the pinned revision — real references,
not imagined future reuse; a boundary or round-trip test; the compatibility
decision (breaking, additive, versioned, staged) and how consumers learn about
it; the places where the type guarantee stops.

Weak substitutes: interface definitions with no reference search behind them; a
passing build in a monorepo whose consumers compile against the local source but
ship separately.

## test and verification gap

Ask: **what real regression could occur while every current check stays green?**
Where would the failure surface — a boundary value, a permission, a race, a
recovery step, a generated artifact, a second caller — that no assertion touches?
Does a green test name actually exercise the behavior or only its construction?
Would the new test fail on the previous revision?

Evidence: the named check run on both revisions where feasible (old fails, new
passes), the negative or boundary assertion the claimed risk needs, and the gap
between the checks that exist and the evidence the claim requires. Report the
gap as an untested claim an owner can close, not as a coverage percentage.

Weak substitutes: coverage numbers, test counts, a suite that passes without the
changed path being reached, a ticket saying tests were added.

## silent failure

Ask: where can this code fail without anyone noticing — swallowed exception,
default-on-error, empty result indistinguishable from no result, retry that hides
a permanent failure, log line missing the identifier needed to act, error mapped
to a success status, background task with no owner? Which failure here is worse
than a crash, and does anything detect it?

Evidence: each error path traced to its terminal handler (log, metric, alert,
return value, exit status); a demonstration that a failing dependency surfaces;
the distinction between "no data" and "failed to fetch" in output and stored
state; what an operator would see at 3am.

Weak substitutes: a catch block whose comment says it is intentional; a metric
that exists but is never alerted on; structured logs without the correlation
identifier.

## Keeping a lens honest

- A lens reports what it established and what it could not. Reading source never
  becomes a runtime claim, and a green badge is not execution of this revision.
- A lens does not edit the code it reviews; that is what preserves its
  independence.
- A lens does not open its own reviewers. Parallelism belongs to the round, and
  the coordinator owns the budget.
- When a lens needs a permission, credential or environment it does not have,
  the finding is blocked or unverified — not negative.
