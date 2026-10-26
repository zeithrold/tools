# Useful UI and fresh data are separate claims

Read when changing request dependencies, streaming, prefetching or caching.
Do not introduce a cache when no measured bottleneck or reuse requirement exists.
Use the project's framework/version documentation; a generic Skill does not
override native server/client exports or cache APIs.

## Establish the contract before optimizing

- Identify the data owner, request dependencies and the user-visible content
  needed for the task. Start independent reads together; do not parallelize reads
  whose authorization or inputs depend on an earlier result.
- State a cache key's full scope: resource, relevant parameters, tenant/account,
  locale and permission-sensitive context. Do not cache authorization outcomes
  under a key that ignores their identity.
- Define freshness, invalidation and mutation behavior. A successful write must
  not be followed by an unlabelled stale read that appears to undo the action.
- Distinguish pending UI, retained data, missing data, failures and completed
  content. A fast shell/spinner is not proof that useful content arrived.

## Verify both claims in the intended runtime

For a streaming or cache change, observe:

1. What meaningful content is present initially and when the primary action
   becomes usable, not just the first paint or a framework timing flag.
2. A cold navigation and a warm navigation, reported separately.
3. Mutation followed by read/navigation: invalidation or reconciliation must
   produce the intended state.
4. An account/locale/parameter change: no cross-scope reuse.
5. Slow, failed and out-of-order responses: the UI must retain a truthful state
   and a recovery path.

Use real component/SDK integration with controlled external responses, plus the
built-runtime checks needed to exercise framework cache behavior. A mock cache
or a dev-server screenshot does not establish production-mode invalidation.
Record correctness separately from latency; a faster but stale or empty screen
is not an improvement.
