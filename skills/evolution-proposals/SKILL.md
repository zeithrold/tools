---
name: evolution-proposals
description: Use when a host or agent proposes changing reusable guidance or agent behavior from experience, or when evaluating, activating or revoking such a candidate. Keeps provenance, independence, safety limits and reversible activation explicit. Not for ordinary feature work, writing a Skill from stable sources, or one-off task notes.
license: MIT
---

# Evolution proposals

An evolution proposal changes reusable guidance or agent behavior because some
past experience suggests the next task would go better. This Skill owns that
loop: what a candidate may claim, how it is evaluated, and what must be true
before it becomes active. Proposing is the easy part; a model may generate
candidates, but execution, evaluation and activation belong to a trusted host and
the owning project.

Use it when reviewing or designing such a loop, or when judging a specific
candidate. Writing a Skill from stable sources is `skill-authoring`; an
investigation of an unknown failure is `systematic-debugging`.

## Supported change objects

This Skill covers proposed changes to procedural guidance or explicit prompt
parameters. Environment/user facts remain at their owning scope. Executable
agent-program changes, automatic root-rule modification and model-weight
training are not authorized by this guidance or a guidance-digest approval.
They require separate designs for execution permissions, evaluator/data
independence, checkpoints, rollback and controlled promotion; defer them rather
than treating all agent-state changes as the same learning mechanism.

## Classify what is being claimed

| Kind | Example | Lifetime | Who may promote it |
| --- | --- | --- | --- |
| procedure | "check the migration ledger before running a local check" | until the mechanism changes | owner after evaluation |
| environment fact | "this checkout needs tool X at version Y" | until the environment changes | expires or is re-verified |
| user preference | "keep answers short in this thread" | this thread or stated scope | never globally by inference |
| one-off task note | "the fixture used ID 42" | the task | not stored as guidance |

A fact about the user or an environment is not a general method. Promote it at
the scope that owns it, with an expiry or re-verification condition; a session's
preference stored as a global rule teaches the wrong lesson to every later task.

## A failure narrative is not a method

"The task failed because I skipped X" is a hypothesis about one observation. It
is not evidence that adding X improves future tasks. Before proposing guidance:

- check the counterfactual: does the same task fail without X, and does adding X
  alone change the outcome?
- check whether another cause explains it — the environment, a missing tool, an
  ambiguous request, a model limitation, one unlucky sample;
- check whether the failure is even stable: one occurrence cannot distinguish a
  defect from noise.

Prefer correcting an existing mechanism. When an existing rule missed a trigger,
gave the wrong order or stated an exception too vaguely, fix that text. Adding a
counter-rule beside it leaves two rules and the reader decides; a synonym that
says the same thing is duplication, not redundancy. A new mechanism earns its
place when no existing owner covers the decision and the change alters what
happens on a future task.

Also ask the marginal questions: which task class does it apply to, which step of
the next execution changes, what in the evidence supports it, which tasks must
never use it, and what would make it obsolete. If the answer to the first is
"everything", it is too broad to evaluate.

## Candidate envelope

Every candidate carries its own claims, not just a patch:

- source and immutable evidence references, not a paraphrase as the sole proof;
  an authorized, scoped original transcript reference may support local review,
  but indiscriminately copied private transcripts are not a public payload;
- generator and any later modifier, with tenant or project scope;
- applicability: the task class and conditions where it should apply;
- exclusions: tasks where it must not apply;
- expiry or review date, plus the tool and dependency versions it assumes;
- lineage: which rule or candidate it modifies or supersedes;
- license and provenance for any adapted material;
- novelty: which existing rule it changes, and why the change is not already
  covered;
- trust level and derivation chain.

[Candidate envelope](references/candidate-envelope.md) expands each field with a
synthetic example and the checks that reject a candidate early. The concrete
schema belongs to the host implementation; do not freeze field names in guidance.

## Contamination survives transformation

A candidate starts untrusted, with its labels attached. Summarizing it,
embedding it, moving it between files, merging it with trusted text or citing it
from a rule does not make it trusted. Retrieval frequency is not a trust signal:
recalling a wrong rule many times makes it familiar, not correct.

- Record trust level and derivation at creation, not after the fact.
- Revoking a source revokes everything derived from it, including archived
  candidates, index entries and merged text; an archive that keeps revoked
  material is still a leak.
- Do not copy secrets, private source or personal data into candidate text; a
  content hash is not a privacy check.
- A candidate produced from a failure under an untrusted source cannot be
  promoted by majority vote among agents sharing that source.

## Independent evaluation of the whole loop

Independence is structural, not a promise:

1. **Permission**: the generator cannot edit the evaluator, criteria, sealed data,
   historical scores or the cost ledger.
2. **Data**: development data for generating candidates, validation for choosing
   among them, a sealed set for the final generalization claim. Split by
   repository, task family, time and template source rather than adjacent random
   samples, count accesses, and fix the evaluation plan before opening the sealed
   set.
3. **Evidence**: success comes from host-observed execution or a trusted
   environment, never from the candidate's own report that its tests passed.

Every participant that learns from outcomes — reflector, reviewer, archive
explorer, code modifier — belongs to the learning system and must respect the
same split. Criteria and acceptance are frozen before trials; a candidate may not
adjust them, and a re-run after seeing the score is a new experiment.

Compare baseline and candidate under the same environment, inputs and budget, and
keep failure output rather than averages alone. Report `passed`, `failed`,
`blocked`, `not_run` or `unknown` for what actually happened; a missing
environment is blocked, and agent-authored summaries are not execution evidence.

An LLM judge can explain failures and propose counterexamples. It is usable only
when calibrated against known cases, sampled by humans, and paired with
deterministic checks. Another reviewer sharing the model, prompt or summary is
not an independent judge. `skill-authoring` and `verification` own authoring and
evidence discipline; this Skill owns the loop around them.

## Hard constraints before any score

Check limits first, then quality:

- no privilege escalation, no evaluator or criteria change, no sealed-label
  access, no data exfiltration, no bypassing native tests, no budget overrun.

A violation rejects the candidate; a high average score cannot compensate for
one. Keep the counters honest: fewer rules is not a better library, fewer tool
calls is not efficiency, a pass rate that rose after tests were removed is not an
improvement, and self-reported cost is not a bill.

## Candidate, active state and approval

A candidate is immutable identified content plus its envelope. Active is a
pointer to a specific candidate with a recorded approval. Approval binds to the
canonical digest, base revision, target, validation contract and evidence; if the
content or base changes, the approval and the results that depended on it are
invalid, and the diff is re-reviewed.

- Require explicit human approval of the exact digest. "You may improve yourself"
  is not approval for a specific change.
- Roll out reversibly: shadow or limited scope first, monitor the stated signals,
  and verify the rollback path before activation. If the previous state cannot be
  restored, refuse the destructive change instead of downgrading rollback to
  best effort.
- Keep the candidate archive and the active pointer separate; promotion is not
  deletion of alternatives.

[Activation and rollback](references/activation.md) covers digests, staged
rollout, revocation and the failure cases from the earlier loop stages.

## Non-use is not usefulness

Usage counters are incomplete: shell reads, cached loads, unlogged executions and
sessions outside the instrumented host all undercount. Low or unknown usage is a
reason to review or quarantine, never to delete a low-frequency but critical
capability automatically. Expiry should ask for re-verification, not silently
remove guidance. Revocation and archiving are audited, reversible operations with
an owner.

## Enforcement belongs to the runtime

File ownership, network access, credential use, subprocess scope and budgets
(token, requests, wall time, concurrency, money) are enforced by the host
environment. Prose instructions are not a boundary: an agent with terminal access
and a sentence saying "only edit Skills" still has terminal access. Reserve budget
before use and reconcile after; cancellation propagates to children and
background work; a candidate cannot raise its own budget; no long-lived
credentials are readable from a candidate's execution environment.

## Aligning with the CLI

The released `v0.2.0` provides `zt evolve prepare`, `plan`, `check`, `submit` and
`status`. `prepare` stores structured complete file replacements as one immutable
candidate under the workspace's `.zt/artifacts/evolve/`; `plan` revalidates that
candidate without a write, credential, network request or execution. `check` runs
the operator's own validation contract against a materialised complete pinned
base, and only with explicit trusted-local consent: it is neither an OS sandbox
nor credential isolation, and its result is operator-local rather than an
independent attestation. `submit` requires that local record plus human approval
bound to the exact public payload, reconciles by reading before it writes, and
creates at most one draft pull request; it never merges, force-updates a ref or
reopens a closed proposal. `status` reads local state and contacts GitHub only
with `--remote`. Consult the installed CLI's help rather than assuming a source
checkout matches an installed release.

Activation, rollback, revocation, host-enforced cost/capability budgets,
model-driven candidate generation and Issue-tracked follow-up remain
unimplemented. A roadmap stage, a released command or a draft pull request is not
evidence that a gate exists: a draft is not a merge, a publication or an
activation, and platform check observations are not independent proof of improved
guidance. Report absent stages as unavailable or blocked for the requested task;
do not simulate them with prose or invent output schemas.
