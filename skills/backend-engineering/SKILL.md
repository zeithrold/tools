---
name: backend-engineering
description: Use when designing or reviewing a server boundary — where validation belongs, what a shared contract may carry, persistence or migration change, concurrency, idempotent acceptance and recovery. Not for client-only UI, a contained refactor with no runtime boundary, or a data model whose semantics the project has already fixed.
license: MIT
---

# Backend engineering

Ownership, trust boundary, lifecycle, authorization, persistence and recovery
outlive layering choices: the number of tiers, the database, the queue and the
cache policy follow the consumer's real failure modes. Read the project's own
instructions, schema and entry points before applying anything here.

This Skill owns the backend domain decisions only. Work order and scope stay with
`task-planning`, independent assessment with `code-review`, status vocabulary
with `verification`, unknown-cause investigation with `systematic-debugging`,
measured optimization with `performance-engineering`, and framework or language
checks with the relevant native references in `verification`. These are optional
ownership routes, not required sibling installations or an instruction to load
every workflow for a backend task.

## Put each guarantee where it is owned

Parse untrusted input where it crosses into trusted code, then pass values whose
type already carries the guarantee. A second check deeper in duplicates the rule
and drifts from it, leaving two truths about the same request.

Split deterministic constraints from state-dependent ones:

| Constraint | Owner | Example |
| --- | --- | --- |
| shape, type, range, format, required fields, declared enum values, size limits | schema or deserializer at the boundary | `quantity` is an integer in 1..99; `status` is one of the declared values |
| authorization, existence, conflict, quota, legal transition, validity of an external response | execution layer that holds the current state | caller owns the order; the order is pending; stored version matches the request |

A schema cannot decide permission without querying the store, and a handler that
re-derives field types duplicates the boundary. Authorization belongs to the
resource path itself, not to its usual caller: HTTP handlers, workers, jobs and
internal endpoints each own their own checks and each needs a negative test that
the check exists.

Synthetic shape, not a project policy:

```text
PATCH /orders/{id}   body {"quantity": 3, "version": 7}
schema    quantity int 1..99; version int > 0; unknown fields rejected
handler    caller may modify this order (authorization)
           order exists and is "pending" (existence and transition)
           stored version equals 7 (conflict)
store      unique (tenant, external_id); quantity update is atomic
```

One owner per line. Dropping the conflict check because the schema accepted the
body is the defect class this split prevents. Keep diagnostic causes
distinguishable in appropriately protected logs without recording secrets.
Caller-visible distinctions follow the project's security and API contract:
unauthorized and absent resources may intentionally share one response to prevent
account, tenant or object enumeration. Do not impose universal HTTP mappings.
Test that a caller cannot infer protected existence while an authorized operator
can diagnose the failure.

## Contracts carry data, not instances

A shared schema or model holds serializable facts: names, types, optionality,
enumerated values, version, error codes. It must not hold clients, connection
pools, loggers, request contexts, lazy singletons or behavior that needs a live
process. Importing a service instance makes every consumer's tests, wire format
and deployment depend on that service's lifecycle, and the contract can no
longer be persisted or sent.

- One source of truth per contract; derive or generate other representations.
- State the version consumers may rely on and what a change to it means.
- Treat an added field as a compatibility change with an ordering, not as a free
  addition; see persistence below.

## Extension-first, narrow core

When a consumer needs behavior the core does not express, first use an existing
extension point. If none fits, add a generic hook, strategy or interface with
consumer-supplied configuration instead of branching on a project name, tenant,
route or environment. A named exception inside a generic core becomes a rule
every consumer carries and no consumer owns.

- A new core capability needs a reason that holds for every consumer. When only
  one caller needs it, the caller owns it.
- Prefer one narrow mechanism to several special cases; every option multiplies
  the states later readers must reason about.
- Keep business rules, schemas, credentials and deployment policy with the
  project that owns them, and pass them in explicitly.

## Persistence, migration and concurrency

Read [persistence and concurrency](references/persistence-and-concurrency.md)
before changing stored schema, startup behavior, locks, indexes or delivery
semantics. The failures that recur most:

- An added optional field can still break an external store, a cached payload,
  an older reader or a rolling deploy. Optional for this code is not optional for
  every consumer. Choose an order: write both, read new, migrate, then retire.
- Startup must not run destructive DDL, hidden migrations or silent backfills.
  Migrations are explicit, reviewed, ordered and observable, with a known
  pre-state and a recovery path. A service that alters data on boot turns every
  restart into a data operation.
- A process-local lock does not coordinate processes. Concurrent instances
  creating the same index, row or enum need a database-level guarantee: unique
  constraint, advisory lock or migration ledger. In-memory mutexes narrow the
  window and hide the race.
- Durable acceptance is not execution. An inserted job row, an accepted event or
  a returned 202 records an intention; it does not establish that the work ran.
  Name the boundary, what a crash between record and execution leaves behind, and
  how that state is drained.
- An idempotency key has a scope, a replay rule and a recovery rule: which
  operation and tenant it covers, how long it is retained, what response replays,
  and what happens when a retry arrives while the first attempt is still
  running. Claim at-most-once only within the scope actually enforced. One
  process remembering a key does not make retries across processes exactly-once.

## Evidence

Backend claims are established at the resource path, against a real store or a
faithful double, not by reading the handler. For each touched boundary name the
check that would fail if the guarantee were removed: a wrong-owner request, a
stale version, a replayed key, a crash after acceptance, a second instance
creating the index, an old reader receiving the new field.
[Failure scenarios](references/scenarios.md) works through synthetic cases with
the tempting fix, why it is wrong, and the owning check.

Report what ran, against which store and migration state, and what remains
unverified. A passing schema test is not evidence about authorization, conflict
handling or recovery. `verification` owns what a result is worth and the status
vocabulary; the requirement here is that the check exercises the owning resource
path, not the handler's source.
