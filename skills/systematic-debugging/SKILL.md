---
name: systematic-debugging
description: Use when a failure's cause is unknown, as with intermittent behavior, a regression, an environment-specific result or a fix that did not hold. Establish a reproducible baseline, test falsifiable hypotheses with minimal probes, and separate cause from mitigation. Not for a known cause with a direct fix, review requests, or choosing performance work from an existing measurement.
license: MIT
---

# Systematic debugging

A debugging pass produces a mechanism, not a patch that happens to work. Keep
the investigation separable from the fix: evidence first, then the smallest
change that removes the cause. For the status vocabulary and what a check is
worth, see `verification`; this Skill owns how to locate an unknown cause.

When the cause is already known and the fix is direct, apply it and add the
regression check. Do not run a hypothesis ritual over a typo, and do not treat
this Skill as authorization to install, reconfigure or access production.

## Establish a baseline before touching anything

Write down, in observable terms:

- the operation, the input and the expected result versus what happened;
- the exact artifact: commit, version, lockfile, image digest or built bundle;
- the environment: OS and runtime versions, configuration that differs, service
  dependencies, data class, credential scope;
- frequency and origin: always, intermittent with a rate, only after a step, only
  on one host, only since a deploy;
- whether it reproduces on the reporter's artifact or only on your local build.

Record the command that shows the failure and its exit status. Missing artifact
identity calls for clarification and authorized read-only investigation to find
it; it does not forbid investigating. Until identity is known, attribution and a
fix claim remain unverified. A result from another version cannot confirm the
reporter's symptom without establishing its applicability.

Separate environment failure from product failure. A missing service, expired
credential, unreachable network or absent fixture is `blocked`, not the defect.
Repair the environment only within your authority, and never count a repaired
environment as a repaired product.

Keep the failing output verbatim in an artifact. A paraphrase loses the line
number, error code, stack frame or request ID that a later hypothesis needs.

## State hypotheses that can be falsified

A hypothesis predicts an observation. "The retry reuses the request body stream,
so the second attempt sends an empty body" predicts a specific empty-body log on
attempt two. "Something is wrong with retries" predicts nothing and cannot be
tested.

1. Rank candidates by available evidence: what changed, what is new in the
   dependency or configuration diff, which input class fails, where the symptom
   first appears in the flow.
2. For each, write the observation that would prove it wrong, then try to produce
   that observation. A probe that can only confirm leaves you with a story.
3. Change one thing per probe. Disabling three features or trimming both the
   payload and the concurrency gives a result you cannot attribute.
4. Prefer read-only instrumentation at existing boundaries: trace or request ID
   correlation, a value logged where it enters and leaves, a query plan, an event
   count. Temporary probes must be reverted and named as temporary; a probe that
   changes product behavior has changed the experiment.
5. Shrink the reproduction before theorizing further: the smallest payload, one
   record, one flag, one concurrent request, one dependency. If it stops failing
   when shrunk, the removed element is part of the cause.

Bisect by time, input or path: history or release range for a regression; halves
of a payload for a parser; one caller for a shared utility. Bisection is useful
because it needs no theory about the mechanism, only a stable reproduction.

Synthetic shape of a working note:

```text
symptom   POST /import returns 500 for files > 2 MB; always, since v4.2.0
baseline  v4.2.0 fails, v4.1.3 passes; same fixture, same local store
H1        new streaming decoder reads the size field after the stream is consumed
          would observe: EOF at field read for the 2 MB fixture, absent for 1 MB
probe     add temporary boundary log of bytes available before decode; 1 MB ok, 2 MB 0
result    H1 supported; H2 (store limit) not exercised
```

The note records what was not exercised, so the next person does not assume it
was ruled out.

## Cause, symptom and the wrong regression

A cause is the mechanism that generates the symptom and explains why it appears
with this shape, at this time, through this path. A symptom fix — a null check, a
longer timeout, a caught exception, an added retry, a default value — leaves the
generator running and generally moves the failure.

A mitigation is acceptable when it is reversible, labeled as mitigation, and
paired with an owner and a follow-up. Say what remains unexplained and what
symptom would show the cause is still present.

Before declaring a fix, check generality: does the same class of input fail
through another entry point, a batch path, an older client or a retry? A fix that
holds for the observed sample only is a narrowed reproduction, not a cause.

A regression test must reproduce the reported symptom. If it fails for a
different reason, asserts a different message, or passes on the reporter's
artifact while the failure persists, it is not a regression test for this defect.
Fix reproduction first; a green test for the wrong failure is worse than no test,
because it closes the report.

## Urgency and reversible mitigation

Under pressure, prefer the action with a stated undo: roll back the release,
disable the feature flag, restore the previous configuration, shed the failing
workload, drain the queue. State the scope and the rollback trigger before
action.

Destructive mitigation — deleting data, forcing a migration, purging a durable
cache, rotating credentials — needs explicit authorization and a recovery path.
Urgency supplies neither.

Mitigation stops the bleeding; it does not close the investigation. Keep the
cause unknown until evidence says otherwise, and keep the mitigation visible in
the report so a later reader does not mistake a suppressed symptom for a fixed
defect.

Do not reach for green by retrying until it passes, lowering a threshold,
skipping the case, widening a tolerance or renaming a failure. A flaky test is an
unexplained defect; a timeout is missing diagnostic context. Capture the
correlated logs, resource state and timing, then explain the flake before the
suite can be called passing. `verification` owns the resulting status language.

## Stop and rethink

Each attempt should narrow the candidate set. When the attempts stop narrowing —
a patch here, a guard there, the symptom reappears a step later — the model of the
system is wrong, and more patches only add hidden state.

Stop when:

- three consecutive probes produce no new discriminating observation;
- a fix works only after several unrelated changes;
- the failing path depends on something you cannot observe (another team's
  service, unavailable logs, production-only data);
- the agreed time budget expires.

On stop, write what is established, what is ruled out and how, what remains
unknown, and the next smallest probe that would discriminate. Ask for the
missing access, evidence or decision rather than continuing to guess.

[Probes and isolation](references/probes.md) covers bisection, instrumentation
and multi-process reproduction. [Worked scenarios](references/scenarios.md)
applies the rules to synthetic intermittent, environment and regression cases.
