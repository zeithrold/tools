---
name: performance-engineering
description: Use when a defined workload must get faster, cheaper or smaller and a baseline can be measured. Count work first, remove waste before caching, and compare baseline and candidate under the same conditions. Not for unmeasured slowdown suspicions, correctness or dependency questions with no workload, or optimizing code that no measurement implicates.
license: MIT
---

# Performance engineering

Performance work starts from a workload and a measurement, not from an
observation that some code looks wasteful. A change is faster only relative to a
named metric measured under stated conditions; every other claim is a
hypothesis about work, which is still useful, but must be stated as one.

Use this Skill when a workload must get faster, cheaper or smaller and a
baseline can be taken. Do not use it to justify speculative caching, to answer a
correctness or dependency question, or to optimize a function no measurement
implicates. For a failure whose cause is unknown, use `systematic-debugging`
first.

## Define the workload and metric

A reportable performance claim needs all of:

- the operation: page load, one API call, a job over N rows, a build, a query;
- the input: size and distribution, data volume, concurrency, cold or warm state;
- the metric and its unit: p50/p95 wall time, throughput, CPU time, peak memory,
  bytes transferred, statements executed, cost;
- where it is measured: client, edge, server, process, store; and what is
  excluded;
- the environment and versions: hardware or runner class, runtime, dependency
  lock, dataset version, seed, config.

"Faster" without these is not checkable, and a comparison across two of them is a
comparison of two experiments. [Measurement](references/measurement.md) has a
workload card, the interleaving protocol and the budget rules.

## Count work before timing it

Instrument useful work per operation first: SQL statements and rows, RPCs, file
reads, process spawns, lock acquisitions, renders, allocations, serialized bytes.
Counters help attribute cost; record their variability too. A timing difference
of 5% on a shared CI runner may be noise, not a threshold for accepting a claim.

Then time the metric users or the budget care about. Report both counts and
observed effects. Unchanged counts leave the cause unattributed; they do not
disprove an optimization. The same number of queries or renders may cost less
through a better plan, less contention or cheaper work per operation. Investigate
that mechanism with controlled paired measurements before attributing the gain.

## Remove work, in this order

1. **Measure a baseline** for the defined workload. Keep the raw output.
2. **Delete unnecessary work**: queries nobody reads, fields fetched and
discarded, duplicated fetches, N+1 patterns, work done per item that could be
done per batch, serialization round trips, retries that always fail, eager
prefetch for unused paths.
3. **Have the owner compute once**: the component that owns the fact derives it
   once and exposes the result. Do not let each caller recompute it, and do not
   add a second cache beside the owner's own state.
4. **Pass prepared facts**: compute where the data is and pass the result. Passing
   a repository, filter or connection so each layer re-derives the same value
   moves work instead of removing it. Prefer a resolved value or a projection
   with exactly the needed columns.
5. **Batch** similar small operations: one query with a bounded key set, bulk
   insert, one round trip, batched renders. State the limit and the per-item
   failure granularity; a batch that fails as a unit can be slower to recover.
6. **Cache last**, and only with a known key, scope, invalidation source and
   acceptable staleness. Caching before removing waste keeps the waste, adds a
   consistency problem and can hide the regression that made the cache look good.

[Optimization order](references/optimization-order.md) expands each step with
when it applies, what it costs and the check that shows it worked.

Caching and parallelism are the expensive tools. Prefer removing work to
parallelizing it; parallelism adds coordination, can increase total work and
makes failures harder to attribute. Every optimization adds states — staleness,
invalidations, partial batches, duplicate sources — so price the maintenance cost
and say who owns it.

## Compare baseline and candidate honestly

- Interleave runs: baseline, candidate, baseline, candidate, in one session and
  one environment. Report per-pair deltas and their spread, not one run of each in
  sequence; machine load, thermal state and cache warmth drift between them.
- Separate cold and warm measurements. A warm-path win can hide a cold-start
  regression; first-request latency, migration, index build and connection
  establishment belong in the cold number.
- Use fixed inputs, seeds and dataset versions, the same concurrency, and the same
  power or runner class. Record them.
- State the sample size, the median and the spread. A shift inside the noise band
  is unmeasured, not an improvement, and a single lucky run is not evidence.
- Do not turn one measured factor into a universal multiplier. "Caching made this
  10x faster" holds for this workload, these inputs and this cache state; report
  the factor with its conditions instead of applying it to another path.

## Profile to locate, not to justify

Profile (CPU, allocation, query plan, trace) after the workload measurement shows
a bottleneck the change targets, and profile the path that measurement implicates.
A profile of an unrelated path shows where time goes in that path, not permission
to optimize it; a micro-optimization there cannot appear in the metric.

Use the profile to choose which work to remove, then return to the workload
measurement to confirm the effect. Never replace the workload metric with the
profile: an operation count improving while p95 stays flat is an unproven user
result.

## Scope, reuse and invalidation

Reuse an earlier measurement only while its identity still holds: same workload
and inputs, same code path, same dataset version, same environment, same config.
A later diff may still leave that identity intact, but say which diff and why it
cannot touch the measured path; otherwise re-measure.

Re-measure when any of these change: the hot path, data volume or distribution,
dependency or toolchain version, cache policy or TTL, concurrency, or the
environment class. A development-container benchmark is not a production claim;
when extrapolating, name the workload differences.

## Budget and report

Agree a time and cost budget for the investigation and the target improvement
before starting, and stop when either is reached. Include the cost of the
measurement itself: a benchmark that consumes paid API calls, large runners or
hours of CI is part of the decision.

Report: workload, metric and unit, baseline and candidate numbers with spread,
environment and versions, which work was removed, what regressed, the maintenance
and operational cost, and what remains unmeasured. If the improvement required a
behavioral tradeoff — more memory, staler data, weaker failure isolation — say so
and get the owner's decision; a performance win does not authorize a product
change.
