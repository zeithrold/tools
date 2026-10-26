# Worked scenarios

Synthetic cases that show the rules in use. They are exercises, not reports
about a real system; the specific tools are interchangeable.

## 1. Intermittent failure under load

Symptom: about one upload in forty returns 500. Locally, with one client, it
never fails.

- First move: capture the failing request's correlation ID, the stack frame and
  the concurrency at failure time. Do not restart the service first; the restart
  destroys the state that makes it fail.
- Hypothesis A: the payload is too large. Probe: same payload with one client, no
  failure. Narrowing removes size as sufficient, though it may still be necessary.
- Hypothesis B: a connection is held across a retry. Probe: two concurrent
  uploads with a temporary delay before the retry; the failure rate rises and the
  pool wait time at the boundary spikes. That predicts the observation; the
  competing "store is slow" hypothesis predicts store latency, which is flat.
- Cause: the retry re-reads the body while still holding a pool connection, so a
  slow client starves the pool. Symptom fixes (larger pool, longer timeout) move
  the threshold without removing the holder.
- Regression: N concurrent uploads through a slow body reader must fail before the
  change and pass after, with the pool acquisition count recorded. A single-client
  test cannot regress this.

## 2. Works locally, fails in CI

Symptom: an integration test fails in CI and passes on the author's machine.

- Baseline first: the artifact is the same commit; the environment differs. Name
  the differences instead of re-running until green in CI.
- Probe: run the same build with the CI locale and timezone in the local
  container. If it fails, the defect is environment-dependent product behavior
  (a date boundary, decimal formatting, case-sensitive path) rather than a broken
  CI runner.
- If it still passes, the difference is elsewhere: dependency cache, data fixture,
  clock, concurrency or network policy. Capture the CI environment, not just the
  error text.
- An environment failure is `blocked`, not a product failure. Fixing the runner so
  the test passes does not establish the product is correct under the other
  environment, and it hides the portability bug.

## 3. Reported regression that does not reproduce locally

A user reports that v5.0.0 breaks export for a certain account. HEAD passes the
same test.

- Wrong move: add a regression test at HEAD that passes, then close the report.
- Correct baseline: reproduce on v5.0.0 with the reported artifact. If the
  symptom is not reproduced there either, the input or the environment is still
  unknown; ask for the request ID, data class or a capture.
- Confirm the regression pair on the same input: old version passes, reported
  version fails. If the old version also fails, this is not a regression and the
  history is a red herring.
- If it reproduces only on the reporter's data, the cause is data-dependent. Shrink
  the data set before reading code; a guard added at HEAD for a different failure
  is not a fix.

## 4. The accumulating-patch loop

Symptom: a background job intermittently processes a record twice. Attempts:

1. add a dedupe check before the write — duplicates drop but occasionally a
   record is skipped;
2. add a retry when the write is skipped — duplicates return;
3. add an in-memory set of processed IDs — duplicates drop in one process, return
   with two workers;
4. add a longer sleep before the retry — observations change but nothing is
   explained.

Each patch narrows a symptom and adds state. Three unrelated changes later, the
model is wrong: nobody can say what guarantees a record is processed once.

- Stop. Return to the boundary: where is the record claimed, where is the effect,
  where is completion recorded, and what does a crash between them leave?
- The likely mechanism is a claim without atomic completion, so a retry after a
  timeout re-executes the effect. A stable invariant — an effect key enforced by
  the store — replaces the guards.
- Report the ruled-out attempts with their observations; the next person should
  not repeat them, and the dead ends are legitimate evidence.

## 5. Pressure to retry until green

A required test fails once in about two hundred runs. The release is late and the
request is to rerun until green and report the suite as passed.

- The failure is an unexplained defect. Capture the seed, the order, the
  environment and the correlated logs from the failing run; a single rerun that
  passes adds no information.
- Distinguish environment flakiness (port collision, shared fixture directory,
  wall-clock boundary) from product nondeterminism (unordered iteration,
  unsynchronized shared state). The first is a test-infrastructure defect that
  still needs a fix; the second is a product defect hidden by luck.
- A bounded rerun is legitimate only to test a specified hypothesis, for example
  "the failure needs two tests in this order". Repeating until green selects the
  passing outcome and discards the signal.
- Do not lower the threshold, increase the timeout without capturing why, skip
  the case or mark it as known. Report the suite as failing or as having an
  unexplained intermittent failure; the decision to ship belongs to the owner.

## 6. A deploy that only correlates

Symptom: error rate rose "after the deploy". The deploy also changed traffic
routing and the cache size.

- Correlation with time is the weakest evidence. Compare: was the error present
  before the deploy at lower volume, does it follow the new routing, does it
  appear with the old release under new traffic?
- A rollback that restores the rate is evidence the change is involved; it does
  not show which part. State the residual uncertainty.
- If a second change was bundled in the same release, separate them before
  claiming the cause; the deploy boundary is not a single variable.
