# Handoff

The brief is what the worker has instead of your conversation. If it is thin, the
worker fills the gap with invention; if it is bloated, it spends its budget
re-reading instead of working. Aim for the smallest brief from which the work is
decidable.

## Fields

### goal

The observable result. One or two sentences, in the terms the requester used.

- Good: "a failed webhook delivery is retried with bounded backoff and the same
event/endpoint pair has one logical delivery record across transport attempts;
exhausted deliveries are marked terminal and visible in the status API".
- Bad: "improve webhook reliability". The worker must then guess what success
looks like, and its guess is unreviewable.

The goal is not the plan. If the approach is already decided by a plan, say that
and give the plan its place — but a goal that is really a sequence of steps hides
the constraint the worker would have noticed.

### non-goals

What this unit must not do, including things a competent worker would plausibly
think helpful: refactors of adjacent code, dependency upgrades, config renames,
new metrics, fixing the flaky test it noticed. State "leave it, report it" for
anything real it finds outside scope — otherwise the finding disappears or the
scope grows.

### exclusive files

The paths the worker may edit, and the paths it must not. Precise enough that two
workers cannot both believe they own one file.

- Give directories or path prefixes, not intentions.
- Name the shared files explicitly as owned by someone else (build manifest,
dependency file, schema, route table, changelog, lockfile), because those are the
ones that collide.
- "Read anything, write only these" is the usual shape and the safest default.

### interfaces

What the unit consumes and what it produces, with real paths and shapes.

- Consumes: the functions, types, events, endpoints, stored data or decisions it
may rely on.
- Produces: the interface others may rely on — signature, response fields, schema
columns, event names, error codes.
- The interface it may **not** change without asking, and who owns that decision.

If the produced interface is undecided, the unit is not ready for parallel work:
freeze it first, or run this unit alone.

### context

The minimum needed to decide: intent and why now, constraints that eliminate
options, decisions already made with their reasons, conventions that differ from
the obvious default, and where the relevant code lives.

What to leave out: your whole transcript, your draft attempts, and the conclusion
you want. A worker told what to conclude cannot disagree usefully, and a worker
fed your reasoning inherits your blind spots — the same reason reviewing in a
clean context is worth the cost.

What not to leave out: a constraint you discovered by accident. Unexplained
constraints are the most expensive thing to rediscover after the work is done.

### permission

- read-only or write, and where;
- commands it may run, and the ones needing approval;
- what is untouchable: credentials, secrets, remote state, deployment,
  publication, billing, other people's branches;
- the network and install policy, especially for anything that resolves
dependencies at run time.

Say which of these the host enforces and which are only asked for. A worker run
with write access under a "read-only" brief is a write-capable worker.

### test and output

The exact check and its expected result, because "verified" without a check is an
opinion. Include:

- the command or flow, at the worker's revision;
- the expected observable result, not just "passes";
- any check that must not be weakened to pass;
- the report shape: what changed (paths), what ran and observed, what did not run
  and why, open questions, and anything found outside scope.

## Worked example (synthetic)

Repository `payments` with `services/webhooks`, `services/api`, `web/`, and a
single `CHANGELOG.md` and `go.mod` at the root.

```
Goal: a webhook delivery that fails is retried with bounded exponential backoff,
with one logical delivery record per event/endpoint pair; exhausted deliveries
become terminal and readable from the API. Repeated transport attempts are
possible; receiver-side effects are not guaranteed at most once.

Non-goals: do not change delivery signing, do not touch services/api code, do not
upgrade dependencies, do not add dashboards. The flaky test in internal/queue is
known — leave it and report it.

Exclusive files (write): services/webhooks/**, and the two root files below.
Own: CHANGELOG.md and go.mod for this round — other units are told to request
changes through you.

Interfaces
  Consumes: internal/event.Event (ID, EndpointID, Payload); the existing
            internal/httpclient with its timeout setting.
  Produces: webhooks.Deliver(ctx, Event) error, plus a delivery row holding
            status (pending|delivered|terminal), attempt count and last error.
            Consumed by the api unit for status reads; do not change that shape
            without asking the platform owner.

Context: this repository previously delivered once and dropped failures; the
retry budget was accepted at 6 attempts over ~30 minutes. Scheduling dedupe must
key on event id plus endpoint id, not on attempt. An applied remote effect with
a lost response can trigger another attempt; without a receiver idempotency
contract, do not promise to prevent duplicate effects. Migration style in this
repo is forward-only SQL under services/webhooks/migrations.

Permission: write inside the listed paths; run go test and go build; no network
installs (dependencies are vendored); do not touch migrations outside the listed
directory; do not publish or tag anything.

Test and output
  Check: go test ./services/webhooks -run 'TestDeliver|TestRetry|TestDedupe'
  Expected: ok — duplicate scheduling reuses the event/endpoint delivery record;
            a lost response retries within that record and the attempt budget,
            without claiming a single remote effect; no seventh transport
            attempt starts after six failures; backoff asserted, not slept.
  Also required: go test ./services/webhooks (full package) must stay green; do
  not relax a timeout to make it pass.
  Report: changed paths; commands and observed output; what you could not run;
  any interface change you wanted but did not make.
```

That brief is decidable without a conversation, names the collision-prone files,
freezes the interface another unit consumes, and states the check plus what
passing must look like.

## Receiving a result

Accept a returned unit on its report plus your own check, not on the report's
tone:

- Does the report cover the fields you asked for? A missing "what did not run"
is not "nothing failed".
- Does the claimed interface match the frozen one? A near-match is a change.
- Did it stay inside its exclusive files? Any written path outside them is an
  ownership event, even if the edit is good.
- Does the check it ran demonstrate the goal, or only that code compiles?
- Are its open questions answerable, and do any of them invalidate the remaining
  units?

Then re-check the consequential claim yourself at the revision you intend to
integrate. That step belongs to the parent; a worker cannot verify its own
integration.
