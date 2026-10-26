# Failure scenarios

Synthetic cases, not reports of any real system. Each names the tempting local
fix, why it leaves the failure in place, and the check that owns the guarantee.
Use them to derive your own negative tests; do not copy the scenarios as project
requirements.

## 1. Optional field breaks a strict reader

A service adds `discount_code` to a stored order document and starts writing it.
A second service decodes the same document with a strict schema and now fails on
orders it used to read.

- Tempting fix: relax the reader's unknown-field handling in the same release.
- Why it fails: the release order is still unproven. A rolling deploy runs the
  new writer against the old reader; a failed half-deploy leaves mixed versions
  and no way to tell which orders are affected.
- Owning check: run the previous reader version against a document written by
  the candidate. Then verify the declared deploy order (reader tolerant first,
  writer second) and that rollback still reads new records.

## 2. Startup DDL on every instance

A service creates missing columns and drops "unused" ones at boot so operators do
not have to run migrations.

- Tempting fix: wrap the boot DDL in `IF NOT EXISTS` and a try/except.
- Why it fails: the destructive step still runs on a restart, against whatever
  schema the store has by then, without a reviewer, backup or row count. A
  swallowed error hides a partially applied schema.
- Owning check: start the new version twice against a copy of an old database and
  assert no schema mutation happens outside the migration step; then run the real
  migration with its ledger entry and confirm readiness reports pending, running
  and failed states distinctly.

## 3. Two instances create the same index

Every instance checks for an index and creates it when absent, guarded by an
in-process mutex.

- Tempting fix: add a retry around the duplicate-index error.
- Why it fails: the mutex is per process, so both instances still act; the retry
  hides which instance won and whether a partially built index exists. On a
  non-transactional index build the failure can leave an invalid artifact.
- Owning check: start two processes with distinct IDs against one store and
  assert exactly one create is attempted, or that a fixed-name unique guard makes
  the second a no-op. Repeat with a killed holder to see the stranded state.

## 4. Accepted work that never runs

The handler inserts a job row, commits, publishes to the queue, and returns 202.
The publish fails after the commit.

- Tempting fix: publish before the commit, or retry the publish in the handler.
- Why it fails: publishing first can announce a job the transaction later rolls
  back; retrying in the request path cannot cover a process death between commit
  and publish.
- Owning check: kill the process between commit and publish, restart, and drain.
  A durable linked record (outbox, or a scanned claim state) must eventually
  execute the job once per accepted row, with a visible record of the retries.

## 5. Global idempotency keys

Keys are stored in one table keyed by the client-supplied string, without tenant
or operation scope.

- Tempting fix: add a unique constraint on the key alone.
- Why it fails: the constraint now rejects a legitimate key from another tenant,
  and a replay can return a response body belonging to someone else. The
  uniqueness is enforced; the scope is wrong.
- Owning check: two tenants submit the same key for the same operation. Each must
  execute once, and neither may receive the other's stored response.

## 6. Lost response, duplicated effect

The key is written only after the operation succeeds. The client times out after
success and retries.

- Tempting fix: shorten the timeout, or log the retry.
- Why it fails: the first attempt completed, so the second executes the effect
  again; the timeout was never the defect. A log proves the retry happened, not
  that duplication stopped.
- Owning check: perform the operation, drop the response, replay the same key.
  The second call must return the recorded outcome without a second effect, and a
  crash between effect and key write must leave a state the recovery path can
  resolve.

## 7. Check-then-insert without a constraint

The handler reads for an existing account, finds none, inserts, and relies on the
read to prevent duplicates.

- Tempting fix: retry when the insert is slow, or shorten the read window.
- Why it fails: two requests can both read nothing before either insert.
- Owning check: two concurrent creates for the same natural key, repeated enough
  to expose the interleaving. The store constraint must be the arbiter, and the
  handler must translate the violation into the documented conflict outcome.

## 8. Authorization at the edge only

The public handler checks the caller's permission and forwards to an internal
endpoint that performs the write.

- Tempting fix: add the same check in the client that calls the internal
  endpoint.
- Why it fails: the internal endpoint remains callable from a job, a console, a
  retry worker or a misconfigured network. A client check is a hint, not a
  boundary.
- Owning check: call the internal endpoint directly as a caller without
  permission and assert refusal at that resource path. Repeat for the job path
  that writes the same resource.

## 9. Permission decided inside the schema

A validator loads the resource to reject requests when the caller does not own
it, so every request path "gets" the check for free.

- Tempting fix: keep the schema check and add an allowlist of trusted callers.
- Why it fails: shape validation now needs a live store and runtime identity, so
  it cannot run offline, on a replay or in another language; and a caller passing
  a pre-parsed object bypasses it entirely. One rule has two implementations.
- Owning check: validate a document with no store available, and separately call
  a path that writes the resource without ownership. The first must succeed on
  shape alone; the second must be refused by the execution boundary.

## Using these cases

Each scenario has a before and after shape: the check must fail on the unfixed
candidate and pass after the change, against a stated store, migration state and
process count. A check that would pass in both states tests something else.
Record the exact command and environment in the artifact, and keep the failure
output rather than only the final green run.
