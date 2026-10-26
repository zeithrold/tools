# Persistence and concurrency

Read this when a change stores new data, alters a schema, runs at startup, adds a
lock or index, or promises a delivery or retry semantic. Read the project's own
migration tool, store versions and deployment order first; the mechanisms below
are requirements, not a chosen toolchain.

## Schema changes are compatibility changes

Persistence outlives the process that wrote it. Before changing a stored shape,
list every reader and writer that exists or may run concurrently: other
instances of this service, an older instance during a rolling deploy, another
service reading the same table or an exported document, a cache entry, a queued
message, a backup restored later, an external store the project does not control.

- **Optional is relative.** A field added with a default is optional to the
  writer that knows about it, and unknown to a strict decoder, a generated
  client, a validation layer or a store with a fixed column set. New optional
  fields break external backends that reject unknown keys or that must be
  provisioned before use.
- **Order writes and reads deliberately** (expand, migrate, contract):
  1. add the new representation so old and new records can both be read;
  2. write both representations (or read with fallback) while mixed versions run;
  3. backfill in a bounded, observable, restartable job;
  4. switch reads to the new representation;
  5. remove the old one only after no running version reads it.
  Collapsing this into one release assumes no concurrent old reader, which holds
  only for a single-instance deploy with a maintained rollback window.
- **Deleting or renaming is a separate later change.** A rollback that predates
  the backfill must still read the data it expects.
- **State the invariant, not only the shape.** "Every order row has a currency"
  needs a constraint or a documented repair path; a type alone does not enforce
  it, and concurrent writers can each satisfy the type while violating the
  invariant.

## Startup must not mutate data

A process start is not an authorized migration. Automated destructive DDL at
startup is a data operation with no reviewer and no pre-state, and it runs on
scaled-out instances, in fixtures and on every restart after a crash.

- Migrations are explicit artifacts: ordered, versioned, applied once, recorded
  in a ledger, and reversible or paired with a stated recovery. A boot path may
  check readiness and refuse to serve on a version mismatch; it must not invent a
  schema.
- Destructive steps (`DROP`, truncate, type narrowing, a backfill that overwrites
  user data) need an explicit owner decision, a backup or snapshot you have
  restored at least once, and a count of the affected rows before acting.
- Long backfills belong in a restartable job with progress and a bound, not in
  request or boot paths. A backfill that cannot resume redoes work and can race
  an active writer.
- Readiness must distinguish "schema is what I expect" from "migration is
  running" and "migration failed"; reporting ready in the third case serves
  traffic into a broken store.

## Locks coordinate a scope, not an intention

Ask what the lock actually excludes, in which process, and for how long.

- An in-process mutex or single-flight map excludes other goroutines, threads or
  coroutines in that process only. Once the service runs as two instances, a lock
  keyed in memory excludes nothing.
- Use the store's own primitive when the resource lives in the store: a unique
  constraint for create-once, a conditional update (`UPDATE ... WHERE version =
  ?`) for transitions, an advisory or row lock for a critical section, a ledger
  row for one-time work. Make the primitive the arbiter. A read-then-write
  outside a transaction is not made safe by the mutex around it.
- Name ownership and expiry for locks that can be stranded: who releases, what a
  crashed holder leaves, and whether an expired holder may still be executing. A
  timeout that expires a live holder produces two concurrent owners.
- Keep critical sections short enough that waiting is bounded, and state the
  queueing behavior when it is not.

## Races beyond the row

- **Concurrent index or type creation.** Two instances starting together can
  both decide an index is missing. A database-level guard (unique index with a
  fixed name, ledger row, advisory lock) is required; wrapping the create in a
  process mutex serializes one instance and hides the race.
- **Read-modify-write across requests.** Two clients reading the same value and
  writing derived results need a version check, a conditional write, or an
  aggregate the store computes.
- **Torn multi-row updates.** A balance and its ledger entry, a status and its
  timestamp: write them in one transaction or make the second derived and
  idempotent. A crash between two writes leaves an inconsistent pair.
- **Cache and store disagreement.** A cache update after the store write can be
  reordered or lost. Invalidate after commit, version the key, or document a
  bounded staleness you accept.

## Acceptance, execution and recovery

Separate these states explicitly, then show what a crash at each boundary
leaves behind:

| Boundary | Recorded | Not established |
| --- | --- | --- |
| request validated, resource created | the row exists | anything downstream happened |
| job or event written, 202 returned | accepted work | execution, retries, side effects |
| worker claims job | a lease exists | the side effect completed before the lease expired |
| side effect applied, job marked done | effect | the caller saw success |

- Prefer a mechanism that makes the last two atomic or recoverable: an outbox
  committed with the business write and drained by a worker, a unique effect key
  the downstream store enforces, a lease with a visible expiry and a handler
  that tolerates re-execution.
- A claim is not a completion. If a worker crashes after the side effect and
  before acknowledgement, the retry must be harmless; design the effect to be
  idempotent instead of assuming the crash cannot happen.
- Recovery paths need their own evidence: drain after a kill between commit and
  publish, a replayed message, an expired lease, a restored backup read by the
  current code.

## Idempotency keys

Define four things before accepting a key:

1. **Scope**: which tenant, operation, resource and time window the key covers.
   A global key space lets one tenant's key collide with another's and replay a
   foreign response.
2. **Storage and retention**: where the key and its outcome live, how long they
   are retained, and what a retry after expiry does. Expiry is a documented
   behavior, not a gap.
3. **Concurrent replay**: a second request arriving while the first is running,
   distinguished from one arriving after completion. Returning "in progress"
   with a retry hint is honest; executing twice is not.
4. **Recovery**: what a retry does when the first attempt recorded the key and
   crashed before the effect, or produced the effect and failed to record the
   response. Store the outcome with the key, or make the effect itself keyed.

Do not call the result exactly-once delivery. Name the guarantee that holds
(at-most-once for the recorded scope, at-least-once with idempotent effects,
effectively-once within the key window) and the window where it does not hold.

## Evidence for this area

Useful checks here are negative and stateful:

- an old reader or strict decoder receiving the new field;
- two processes racing the same create, index or transition;
- a kill between acceptance and execution, then a drain;
- a replayed key, a concurrent duplicate key, and a key after retention;
- a restart with migrations already applied, and a boot with them pending;
- a rollback to the previous release against data written by the new one.

Record which store version, migration state and process count the check used; a
single-process run does not exercise distributed assumptions.
