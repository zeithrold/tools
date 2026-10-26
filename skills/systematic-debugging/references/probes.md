# Probes and isolation

Read this when a reproduction is unstable, spans processes, or the obvious probe
would change several variables at once.

## Choose the discriminating probe

A probe is worth building when two competing hypotheses predict different
results. Name them first; otherwise the probe measures something and you decide
afterwards what it meant.

| Question | Probe that answers it | What it does not show |
| --- | --- | --- |
| Is the new code involved? | disable or revert that change, same input | whether another path has the same defect |
| Is the data or the code at fault? | same build with a minimal known-good input | which field matters |
| Does it depend on concurrency? | run serial, then N concurrent, same inputs | which interleaving is required |
| Is it process or environment? | same artifact in a clean environment | behavior in the reporter's data set |
| Is the store the bottleneck? | query count and plan at the boundary | user-visible latency |

Order probes from read-only and cheap to invasive. A trace, existing log or
query plan costs nothing to add; a code change in the middle of the path can
create the defect you are looking for.

## Instrumentation discipline

- Log at boundaries where a value enters and leaves, with the identifier that
  lets you join records across processes. Logging a variable in the middle often
  reports a transformed value you cannot map back.
- Include the discriminating fields: counts, lengths, status codes, versions,
  durations, correlation IDs. A message "failed" without the error and input is
  not evidence.
- Mark temporary instrumentation with a recognizable prefix so it cannot ship
  unnoticed, and remove it before claiming the fix; a probe left in production
  changes what future debugging observes.
- Sampling changes what you see. Prefer full capture for a bounded window over a
  1% sample when the failure is rare, and say which window you captured.

## Shrinking a reproduction

Remove, one dimension at a time: payload size, optional fields, concurrency,
number of records, enabled features, cache state, middleware. Keep the failing
outcome after each removal. The last removal that preserved the failure points at
the cause; the first that fixed it is a component of the trigger, not necessarily
the cause.

When the failure disappears under shrinking, try to reintroduce only the removed
element. A failure that needs both halves is usually a seam: serialization,
timeout, lock scope, size limit or boundary conversion.

## Bisection

Bisection needs a stable reproduction, not a theory.

- History bisection: identify the first release or commit where the symptom
  appears, then confirm the boundary pair (last good, first bad) on the same
  input. A bisect across unrelated refactors produces a plausible but wrong
  boundary; check the commits actually exercised.
- Input bisection: halve the payload or record set. Useful for parsers,
  encodings, pagination and limits.
- Path bisection: call each layer directly, bypassing the ones above it, to find
  the layer where the wrong value first appears. Bypassing a layer removes its
  guards too; note that in the result.

## Concurrency and timing

A race that appears once in fifty runs cannot be studied by rerunning it fifty
times at full speed.

- Widen the window deliberately: add a temporary delay or barrier at the
  candidate boundary, so interleavings become frequent. The delay proves the
  window exists; it does not prove the production trigger.
- Reduce to two actors with controlled order: one writer and one reader, one
  replica leader change, one expiring lease. Raise the count only after the pair
  is understood.
- Assert the invariant between steps (version, row count, lease owner) rather
  than only the final output, because a race often converges to a plausible final
  state.
- For distributed symptoms, correlate by request or trace ID across services and
  record clock skew; a timeout upstream and a slow query downstream may be the
  same event, or two different ones.

## Environment differences

The reporter's artifact and your build are different experiments. A useful
comparison names what differs: dependency lock, locale, timezone, filesystem case
sensitivity, CPU count, container limits, feature flags, data volume, clock,
network policy, credential and permission scope.

- Reproduce on the reporter's artifact when the claim is about that artifact.
  Verify a candidate fix with a controlled build, and say which is which.
- Do not adjust production configuration, credentials or data to make a local
  reproduction succeed.
- If the environment cannot be recreated, say the result is unverified rather
  than generalizing from a nearby case.

## Reproduction matrix

Keep a small matrix of what was actually run, so gaps are visible:

```text
variant                 result   artifact/env
reporter artifact       fail     v4.2.0, their config, 2 MB fixture
local build             fail     same commit, clean store
old release             pass     v4.1.3
two processes           fail     2 instances, shared store
single process          pass     1 instance
```

The matrix is evidence of scope, not a proof of cause. A row that contradicts the
hypothesis is more informative than one that confirms it.
