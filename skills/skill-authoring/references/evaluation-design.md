# Evaluation design

This is a screening design, not a quality certificate. Writing it, or writing
instructions that tell an agent to evaluate, is not evidence that any Skill
changed behavior. Evidence exists only where trials ran in identified
conditions, and only to the extent those conditions were independent and
frozen. No model runner, installer, browser or external service is invoked by
this file; a team that has one uses its own tooling and records the version.

## Decide what question the trial answers

| Design | What the agent does | What it can show | What it cannot |
| --- | --- | --- | --- |
| Decision-only | Reads a scenario, states intended actions; no edits or commands | Policy-shaped decisions: permissions, refusal, ordering, reporting | Tool use, discovery, real execution, recovery, artifact quality |
| Task / tool-use | Works in a real workspace with tools and produces changes or artifacts | Whether the guidance changes executed behavior and outcomes | Causality beyond the sampled tasks, models and environment |

A decision-only screen that avoids all unsafe proposals is useful and remains a
screen. Do not describe it as task performance or pass rates.

For real tool-use trials, [task evaluation](task-evaluation.md) owns the
operational design: native execution first, separate activation and efficacy
questions, benchmark fit, version comparisons and scheduled paired cohorts.
Use a decision screen when it answers the question, not as a mandatory extra
stage before every authorized task trial.

## Choose conditions that isolate the variable

| Condition | Receives | Answers |
| --- | --- | --- |
| No Skill | Base system/project prompt only | Natural behavior and the failure baseline |
| Descriptions only | The catalog descriptions, no bodies | Whether trigger text alone changes selection (an ablation) |
| Previous version | The Skill as it exists before the revision | Whether this revision changed anything |
| Candidate | Description plus revised body or references | Combined effect of the proposed guidance |

Descriptions-only is not the no-Skill baseline, and a body-only revision is not
shown by descriptions-only. For a body change, compare previous full version with
candidate full version; use no-Skill and descriptions-only to interpret baseline
and triggering separately. Compare the same case across conditions with the same
permissions and budget.

## Freeze before the first trial

Record before running, retaining the exact content or an existing immutable
revision that contains the evaluated content so that later changes are visible.
A baseline or HEAD that omits uncommitted candidate changes is not that revision;
retain the exact evaluated content instead.

- Cases, prompts and acceptance criteria, given separately to the trial agent.
- Permissions, allowed tools, workspace contents and isolation.
- Model, configuration, context policy and number of trials per condition.
- Response or action budget, time budget and cost budget.
- Judging process, judge identity and whether the judge sees criteria.
- Stopping rule and maximum revisions.

Changing any of these after seeing outcomes turns the next run into development
work, not a comparison. Add cases only as clearly labeled development and leave
them out of the acceptance claim.

## Case families and activation

Cover, across distinct task families rather than one repeated template:

- Intended activation: the task the Skill exists for.
- Near-miss non-activation: adjacent work the Skill must not take over.
- False activation: work outside its ownership where it still triggers.
- Missed activation: intended work where the description fails to select it.
- Permission negatives: requests that try to gain network, write, deployment,
  credential or publication authority the trial does not grant.
- Held-out family: at least one family not consulted while drafting, when
  claiming generalization. Cases from one repository, template or time window
  are correlated; group by source, not by adjacent sample.

## Hard gate first

Evaluate safety before quality. Any of the following rejects the candidate for
that trial regardless of task scores, cost or elegance:

- acting outside granted permissions, or reading credentials it was not given;
- editing the evaluation, its criteria, expected answers or sealed data;
- installing software, contacting production or merging/publishing without
  authorization;
- exceeding the declared budget after it was frozen.

Do not average a violation away with a higher pass rate. Record it as a failed
trial with the exact action.

## Measures and budgets

For each case and condition record: criterion met (yes/no, per criterion), any
unsafe action, whether and how the Skill was selected, cost (tokens, time,
steps), and whether the result needed retries. Report denominators, not a single
composite score that hides a safety failure behind quality gains.

- False and missed activation are measured against the frozen cases; they are
  not production rates.
- Cost includes model calls, environment setup, reviewer time and retries;
  unrecorded cost is `unknown`, not zero.
- Overfit budget: a maximum number of revisions and trials, and a limit on
  held-out accesses. Stop at the cap. Do not expand it because two conditions
  nearly tied.

## Independence

The author may design and run development trials, find failures and revise.
Acceptance needs a judge who did not write the candidate, working from the frozen
criteria, ideally with execution evidence collected by the host rather than by
the candidate. A different prompt over the same model, or several same-model
reviewers, is not permission, data or evidence independence. Record who ran each
condition, who judged it, and whether conditions were blinded.

## Worked design (synthetic, not executed)

A hypothetical library has four workflow Skills. The question is whether adding
operational references changes behavior without creating unsafe proposals.

```text
cases: 12 fixed scenarios in three families
  A. evidence and completion claims (5)
  B. permission refusal and scope (4)
  C. near-miss/low-risk edits that must not use a workflow (3)
conditions: no Skill, descriptions-only, previous version, candidate
contexts: one fresh context per case and condition
budget: 96 decision-only runs (12 cases x 4 conditions x 2 trials), 5 revisions maximum
freeze: case text, criteria, permissions, model/config, exact content retained
safety gate: unauthorized action or changed criteria rejects the candidate
measure: criteria met, unsafe action, selection observed, tokens/time
phase two: six task cases in a scratch repository for tool use, only after
  the decision screen passes; no dependency installs in that workspace
```

Decision rule for the synthetic plan: any unsafe action rejects the candidate;
otherwise it is better only if it meets criteria the previous version missed,
without a new missed activation and within the same budget. Ties mean no claim.

Limits of this design, stated up front: one model and configuration, small and
authored case set, no statistical inference, author-side construction and
judging unless a separate judge is named, no external runner or automatic
installation, and decision-only results cannot support tool-use claims. The
task phase is a plan here; it is evidence only once it actually runs in an
identified environment. If any of these limits cannot be met, report the
behavioral claim as unverified rather than estimating an effect.

## Report template

```text
claim: what changed in guidance, one sentence
conditions and identities: model/config, environment, exact Skill revision/content
cases: count per family, held-out designation, exact criteria revision/content
trials: planned vs run per condition, reuses and their labels
results: per-case outcomes, unsafe actions, activation observations, cost
judgement: who judged, blind or not, with which criteria
limits: unrun phases, unavailable isolation, single-trial cells, overfit risk
conclusion: supported / not supported / unverified for the stated scope
```
