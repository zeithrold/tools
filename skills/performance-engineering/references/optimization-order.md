# Optimization order

Read this after a workload and baseline exist. The order matters: each step
removes work that later steps would otherwise preserve or cache.

```text
measure -> remove waste -> owner computes once -> pass facts -> batch -> cache
```

## 1. Remove waste

Look for work whose result is discarded, duplicated or not needed for the
requested output:

- a query whose rows nobody reads, or columns selected and then dropped;
- the same lookup repeated per item (N+1) when one query would answer all items;
- serializing and parsing between two in-process layers;
- eager loading a relation for every request when one code path uses it;
- recomputing a constant or configuration on every item;
- retrying an operation that fails deterministically (permissions, validation);
- background work on a path the workload never takes.

Check: work counters drop for the defined workload and output is byte-identical
or the intended difference is stated. Deleting work that produced a visible
behavior is a product change, not an optimization.

## 2. Owner computes once

The component that owns a fact derives it once and exposes the result. "Once"
means once per operation, not once per process forever — unless the owner also
owns the invalidation.

- Bad: three middleware layers each resolve the tenant from the request.
- Bad: a view recomputes totals already computed by the service, with slightly
  different rounding.
- Bad: a per-layer memo map keyed by request, regenerated per layer, each with
  its own lifetime.
- Good: the boundary parses and resolves identity once, and passes a resolved
  value down.

A second cache beside the owner's state is a second source of truth. When the
owner already caches, callers should use the owner's API rather than copying the
value into their own map.

Check: the derivation is called a known number of times per operation, shown by a
counter or trace, and the value is the same one all consumers use.

## 3. Pass prepared facts

Compute where the data is; pass results, not the ingredients to recompute them.

- Pass the resolved list or projection instead of a repository handle that each
  layer queries again.
- Pass a prepared statement or query fragment instead of a predicate the callee
  re-plans for every item.
- Pass the normalized or validated value instead of re-parsing the raw string in
  the renderer.
- Keep the payload close to what the consumer needs; a large object graph that
  each layer re-walks is repeated work with extra allocation.

Boundary caution: passing more context is not free. Adding a field to a widely
used struct increases memory and coupling; prefer an explicit projection for the
hot path. Do not turn this rule into passing whole services around, which hides
work behind lazy accessors that run far from the measurement.

Check: the callee performs no repeated lookup for data the caller already had,
and the hot path's statement and allocation counts drop.

## 4. Batch

Combine similar small operations into one bounded operation:

- one `IN` query with a bounded key set instead of one query per item;
- bulk insert or upsert instead of per-row round trips;
- one remote call with a list instead of a call per element;
- a single render pass over a prepared collection instead of per-item invalidation;
- grouped file writes or one archive instead of one open per record.

State the limit and what happens past it: split into chunks, or refuse with a
clear error. An unbounded batch turns a latency fix into a memory and timeout
risk.

Preserve semantics deliberately:

- ordering, duplicates and per-item errors — does the caller need to know which
  item failed, or is an all-or-nothing result correct?
- retries — a partially applied batch needs an idempotent effect or a resume point;
- fairness — one large batch can starve other requests; cap and queue.

Check: total round trips per operation drop, error handling still maps each item
to a meaningful outcome, and a single failing item does not silently pass.

## 5. Cache last

Cache only after the waste is gone. Caching a wasteful computation keeps the
waste and adds staleness, invalidation and memory cost.

Answer these before adding a cache:

1. **Key**: which inputs fully determine the value? Include tenant/user scope
   where the value varies by them; a missing key component leaks data across
   scopes.
2. **Scope and lifetime**: request, process, shared store; TTL and eviction; what
   happens on restart and in a new instance.
3. **Invalidation**: which writes change the value, and does the writer invalidate
   or version the key? "It changes rarely" is an assumption, not a mechanism.
4. **Staleness**: what does a consumer do with a stale value — show an old number,
   serve a wrong price, decide permissions? Stale authorization or money values
   are not cache candidates.
5. **Cold start and stampede**: what the first requests after deploy or expiry do;
   whether many clients refresh at once.
6. **Size bound**: maximum entries and value size, and the eviction behavior when
   the bound is reached.

Do not cache when the value is already cheap, when the hit rate is unknown, when
invalidation is not owned by anyone, or when the operation must be strongly
consistent. A cache changes failure behavior: an unavailable cache can turn a
working service into a failing one unless the miss path is the real path.

Check: measure hit rate and miss latency for the defined workload, verify
invalidation with a write-then-read, and compare against the uncached candidate.
A cache that is only measured warm is measuring the same data repeatedly.

## 6. Confirm and record

After each step, return to the workload: work counters, then the metric, with the
same environment and inputs. Keep a short log of what was tried:

```text
step                     statements/req   p95      note
baseline                 41               181 ms   measured
remove unused lookup      39               179 ms   noise; kept, simpler
owner computes totals     31               168 ms   confirmed
batch item query          4                96 ms    confirmed
warm cache                4                52 ms    added invalidation on write
```

A step that does not move the metric is not automatically wrong — less work can
still be a maintenance win — but it is reported as unmeasured, not as a speedup.

## Parallelism and concurrency

Parallelize independent work only after removal and batching, and only with a
bound on concurrency. Parallelism adds coordination, error aggregation and
resource pressure; the same total work fanned out over more workers can be slower
under contention. Prefer a queue or a batch over per-item goroutines/tasks, and
state the ordering and cancellation behavior.

## Cost of the optimization

For each accepted step, name what it adds: invalidation points, staleness
windows, duplicate sources, partial-batch recovery, memory bounds, new failure
modes. An optimization is a design change and needs the same review as any other
behavioral change; the measured win does not waive correctness or recovery
checks.
