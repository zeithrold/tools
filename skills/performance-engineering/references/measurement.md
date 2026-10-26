# Measurement

Read this before taking a baseline or reporting an improvement. The goal is a
comparison someone else can repeat, not a number that looks good.

## Workload card

Record this before the first run and keep it with the results:

```text
operation     GET /search?q=...  (one query, page 1)
inputs        10k documents, queries: 70% short, 30% two terms; 200 fixed queries
concurrency   1 client for latency; 8 clients for throughput
state         cold process, empty cache, index built
data version  fixture-2026-11-01
metric        p50 and p95 server latency (ms); statements per request
environment   runner class, 4 vCPU, 8 GB, local SSD, runtime x.y, dependency lock abc123
window        after 30 s warmup, 5 x 2 min runs, alternating with baseline
```

A workload card missing the input distribution or the metric location produces a
result that cannot be compared later. If the operation is a job, add rows,
batch size and failure handling; if it is a render, add viewport, content shape
and cold/warm state.

## What to count

Counts attribute cost and are stable across machines:

| Counter | How to observe | Why it matters |
| --- | --- | --- |
| SQL statements, rows scanned/returned | store logging, driver instrumentation, query plan | reveals N+1, over-fetch, missing predicates |
| RPCs and retries | client instrumentation, trace spans | round trips dominate remote calls |
| file reads, bytes read | process instrumentation or strace | I/O amplification, repeated parsing |
| process spawns | process accounting | startup and per-item subprocess cost |
| renders, layouts, component updates | framework profiler | work per state change |
| lock acquisitions, waits | boundary counters | contention hidden by averages |
| allocations, peak memory | runtime metrics, heap profile | GC pressure and container limits |

Report the count per operation (per request, per item, per page) and how it was
observed. A count from logs under-samples when logs are batched; say so rather
than presenting it as exact.

## Interleaved comparison

Run baseline and candidate in the same session and environment, alternating:

```text
pair   baseline p95   candidate p95   delta
1      182 ms          151 ms         -17%
2      179 ms          154 ms         -14%
3      188 ms          149 ms         -21%
4      181 ms          152 ms         -16%
median 181 ms          152 ms         -16%
```

Why alternating: machine load, thermal state, background jobs and cache contents
drift across a session. Running A once, then B once, attributes the drift to the
change. Per-pair deltas also show whether the effect is stable; a candidate that
wins in three pairs and loses in one needs the outlier explained.

Report median, spread (min–max or interquartile range) and sample count. A delta
smaller than the spread across repetitions is unmeasured; rerun with more
samples or a bigger effect before claiming it.

## Cold and warm

Measure separately and report both:

- cold: fresh process, empty caches, no connection pool, index built from empty;
  includes startup, migration checks, first query compilation, connection setup,
  first render and bundle parse;
- warm: steady state after a stated warmup window, with the stated cache state.

A warm-path optimization that adds cold work moves the cost to the first user, the
deploy or the autoscale event. Conversely, a cold regression can be invisible in
steady-state percentiles. If the operation is rare and long-lived, cold dominates;
if it runs continuously, warm does. Say which one the budget cares about.

## Noise and instrumentation

- Instrumentation changes what it measures: verbose logging, spans, profilers and
  heap sampling all add cost. Measure the metric without the extra probe, or
  report the probe's overhead.
- Fixed inputs and seeds; random data makes two runs different workloads. Where a
  distribution is unavoidable, keep the same seed and state the distribution.
- Pin the environment: CPU quota, memory limit, disk class, network shape, runtime
  and dependency versions, container vs host. A change to any of them invalidates
  the comparison.
- Percentiles need enough samples to be stable; a p99 from 20 requests is noise.
  Report the sample count and avoid percentiles the sample cannot support.

## Profiling gates

The profile is a locating tool inside an established bottleneck:

- start from the metric: which operation, which cohort, which segment of the
  timeline is slow (wait, compute, I/O, lock, render);
- profile that segment and path only; a global profile is dominated by whatever
  runs most, not by what the workload pays for;
- attribute samples to a component and a count of calls, then choose work to
  remove;
- confirm with the workload metric again. A profile that improves while the
  metric does not is a measurement of a different question.

## Budget

Set before starting:

- investigation budget: wall time, CI minutes, paid API calls, large runners;
- target: the metric and the improvement that changes the decision;
- stopping rule: stop when the target is met, the budget is spent, or the next
  experiment cannot discriminate.

Report spent budget when it matters to the decision. A 2% win that consumed days
of paid CI and adds a cache is a cost decision, not just a latency win.

## Reuse and invalidation record

When reusing earlier evidence, record the identity it was measured against and
the diff since:

```text
reused    baseline A (p95 181 ms) from run r-123; metric, inputs, env unchanged
since     commit 9f2c: renames and a log line in an unrelated module
expiry    invalid if the search query path, index fixture or runner class changes
```

Invalidate and re-measure when the hot path, data volume or distribution,
dependency or toolchain version, cache policy, concurrency, or environment class
changes. A successful benchmark identity is not transferable to a different
workload because the code "looks the same".
