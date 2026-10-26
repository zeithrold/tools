---
name: verification
description: Use when selecting behavioral evidence, choosing native Go, JS/TS or frontend checks and gate profiles, reproducing a regression or reporting completion for acceptance. Skip conversational answers and wording-only edits with no behavioral or policy effect.
license: MIT
---

# Verification

This Skill owns the evidence contract and native test strategy: what a result
is worth, which observation supports a claim, how stack checks are selected and
what their limits are, and how to report it. Read the project's declared checks
and entry points first: discovery is not execution, so a configured script or a
command name is a candidate check, not evidence that anything ran.

When this Skill is not needed: a question, a wording-only documentation edit, or
a change with no observable behavior. Read what changed, confirm the content is
intact, and stop instead of imposing a suite, plan or delegation.

State the claim first, in one sentence a reader could disagree with, and derive
checks from it rather than from what is easy to run. Trace the entry point the
claim travels through, the boundary it crosses and the failure it could cause.
Operational procedures and synthetic examples are in `references/`; read the
one the current claim needs, not all of them.

## Select real checks

- Prefer the project's own entry point (task runner, package script, task file,
  CI equivalent) over a hand-assembled command; it owns the setup, isolation and
  environment a raw invocation may skip.
- Inspect unfamiliar or changed entry points for side effects before running
  them: network calls, file writes, spawned processes and credential access.
  Native scripts do not authorize installs, production access or credential
  use. Keep execution within the task's permissions; untrusted candidates need
  an isolated, credential-free environment with only the access the task grants,
  not merely a fresh model context.
- Run the narrowest check that exercises the changed path, then the project's
  declared suite when the claim spans modules or shared behavior.
- Inspect a named entry point when its coverage is in doubt. A `test` script
  that only lints, or a target that exits before running tests, supports a
  different claim than its name suggests.
- Exercise what actually changed: error handling for error claims, the published
  artifact for packaging claims, the real browser or device where a simulator
  cannot establish the behavior.
- Record the exact command, the environment and what the check did not touch.

## Native stack strategy

Native toolchains execute the checks; this Skill selects the claim, the stack
strategy and the evidence limit. Use the project's declared scripts for setup
and isolation. This is the only testing and evidence Skill: stack commands and
gate profiles live under `references/`, and you read the reference for the stack
in use instead of all of them. Statuses, oracles, reproduction, acceptance
assets and evidence reuse below apply to every stack; these strategy rules hold
regardless of stack:

- Keep the code under test and its real dependencies on the path. Replace only
  an external boundary needed for determinism, and assert the interaction and
  outcome rather than the presence of a double.
- Bound fuzz search explicitly. Seed-corpus execution is different evidence, and
  a reproducing input is retained as a reviewed regression.
- Mutation testing has no universal runner. Discover the project-selected tool
  and scope, and report absent tooling separately from test failures and
  surviving mutants.
- Select race, platform and native integration checks when the claim depends on
  them. A cross-build, a DOM emulation and a widget-under-test harness are not
  native runtime evidence. A platform simulator or emulator of the target OS
  does run that native runtime, so it can establish layout, input, focus,
  semantics and lifecycle behavior; it cannot establish hardware-only facts such
  as real device performance, cameras, sensors or an OS credential store. Name
  which of the two the claim needs, and report a missing native environment as
  blocked rather than substituting a weaker one.
- Keep generated corpora, reports and captures in approved locations. Clean up
  processes, sessions and temporary credentials, and inspect the artifact that
  was produced rather than a summary of it.

Read the one reference for the stack in use: [Go](references/go.md) with
[Go checks that can fail](references/go-checks.md), [JS/TS](references/js-ts.md)
with [JS/TS checks that can fail](references/js-ts-checks.md), or
[frontend verification](references/frontend.md) for the ordered lint, CSS, types,
unit, build, browser accessibility and artifact gates, with
[gate integration](references/frontend-gates.md) and
[gate evidence](references/frontend-evidence.md).

## Independent expectations

An expectation computed by the code under test, or regenerated from its current
output, moves with the defect and cannot expose it. Derive expected values from
the accepted contract, a hand calculation or a separate rule, and record which.
See [independent oracles](references/oracles.md).

## Show the check can fail

For a new or materially changed check, ask which plausible wrong behavior would
make it fail. Select applicable, risk-relevant perturbations in a disposable copy:
wrong arguments, a bypassed precondition, a missing side effect or stale data.
These are alternatives, not four mandatory mutations for every behavior.
If the selected fault stays green, strengthen the assertion or narrow the claim.
Do not re-mutate established unchanged gates on every use; say when
discriminating power was not exercised. Grep results, mock definitions and stored
snapshots are static artifacts, not executed behavior.

For a defect, the decisive perturbation is the old revision: the new check fails
there and passes on the changed revision. Feasible means no production data,
paid service or nondeterminism is required; otherwise record the nearest stable
equivalent and say why the cycle was not observed, never that it was.

## Reproduce the reported problem

Establish the symptom on the build the reporter uses, then verify the change on
the exact candidate build; do not substitute one for the other, and do not edit
source while establishing the baseline. Match and record the build, recipe,
environment, symptom and repetition count. A different symptom, or a failure
caused by a missing service or permission, is not a reproduction. See
[reproduction](references/reproduction.md) for the identity record and verdicts.

## Keep acceptance assets separate

Protected acceptance recipes, drivers, fixtures and expected results belong to
the acceptance process, not to the implementation being accepted. Ordinary
regression tests beside the code remain the implementer's work. Editing a
protected expectation, fixture row or skip condition so the candidate passes is
adaptation, not a fix; an incorrect acceptance asset changes separately under
independent review. See [acceptance assets](references/acceptance-assets.md)
before touching one.

## Reuse evidence within its scope

Earlier evidence remains usable only while the revision, environment,
dependencies, recipe and proven contract are unchanged. State that it was
reused, from which run, and what was rechecked; never present it as a new run.
See [evidence reuse](references/evidence-reuse.md) for the record and the
conditions that invalidate it.

## Report status honestly

- `passed`: it ran at a real entry point and the observation matches the claim.
- `failed`: it ran and contradicted the claim.
- `blocked`: the environment, dependency or permission to run is absent.
- `not_run`: you chose not to run it; name the reason.
- `unknown`: it exists only as a report, or cannot be tied to the exact content.

A status describes this run. The observation is an exit status, session output
or generated artifact tied to the run, not a final message, summary table or
reused report. Where the claim requires a process or session, confirm it started
and inspect what it left behind; an assistant's account of it is not evidence.

Never turn `failed`, `blocked`, `not_run` or `unknown` into a pass by relabeling
it a warning, retrying without a cause, relaxing a threshold, deleting or
skipping a check, or narrowing a check after seeing it. A suite covering part of
the claim is partial, and partial is not passed.

## Independence and cost

Acceptance of consequential work (authorization, persistence, migration,
concurrency, release) belongs to someone other than the author and rests on
observed execution, not a summary or an edited acceptance asset. Keep runs
bounded: target the claim, stop when evidence suffices for the decision, install
nothing unexpected, and leave project state as you found it, stating what ran,
what did not and what remains assumed.
