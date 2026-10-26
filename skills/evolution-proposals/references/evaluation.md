# Evaluation independence

Read this when designing or judging the evaluation of an evolution candidate.
The question is not "did the score go up" but "what could explain the score
without the candidate improving behavior".

## Three independences

| Independence | What it prevents | Minimum mechanism |
| --- | --- | --- |
| Permission | candidate edits its own test, criteria, hidden data, history or cost ledger | separate write authority; the evaluator content is outside the candidate's path scope |
| Data | tuning on the cases it is scored on | development for generation, validation for selection, sealed set for the final claim |
| Evidence | self-reported success | host-collected execution records tied to candidate identity |

Permission without data independence still overfits. Data without evidence
independence still lets a model write "tests passed". All three are structural:
file permissions, dataset access and which process records the result.

## The whole loop is the learning system

Generator, reflector, archive explorer, reviewer, summarizer and any code modifier
are all influenced by what they see. If a reflector read private tests or
per-task errors, those tasks belong to development: the system has already
learned from them.

- Split by repository, task family, time period and template source, not by
  random adjacent items that share a template.
- Count accesses to the sealed set and record who opened it and why.
- Fix the evaluation plan — metrics, thresholds, comparisons, sample counts —
  before the first sealed access. Opening the set to choose the plan already
  leaks it.
- Reusing the same validation set across many candidates turns it into a training
  set; promote a fresh holdout or report the claim as development-only.

## Freeze criteria and derive expectations independently

Criteria are part of the candidate's contract and are frozen before trials. The
evaluator owns them; the candidate may not reinterpret, narrow or add criteria
after seeing a result.

Expected outcomes are derived from the requirement, not from the changed
implementation. If the test computes its expectation by running the same logic
under test, it confirms the logic agrees with itself.

## Paired execution

Run baseline and candidate in the same environment, on the same inputs, with the
same permissions, budget and model configuration. Keep the raw failure output;
an average hides which cases broke.

Report per case and overall:

- `passed`: observed execution matches the frozen criterion;
- `failed`: observed execution contradicts it;
- `blocked`: environment, dependency or permission absent;
- `not_run`: deliberately skipped, with the reason;
- `unknown`: only a report exists, or the result cannot be tied to the exact
  candidate identity.

A suite that covers part of the claim is partial. Partial is not passed. An
environment failure is blocked evidence about the environment, not a product
regression or a pass.

## Contamination and overfitting scenarios

Synthetic examples, all plausible and all failures of the loop:

- **Reflector sees the answer.** A failed-task error message contains the private
test expectation; the reflection proposes a rule that passes that test. The task
must move to development, and the rule's evidence is now unverifiable.
- **Validation reused as training.** Twenty candidates are scored on the same
  200 cases and the best is chosen. The winner's advantage includes selection
  noise; the claim is development-only until a fresh holdout agrees.
- **Per-item errors leak into generation.** The generator reads a per-task error
  list from the evaluation run. Even without private data, the candidate now fits
  the errors rather than the task family.
- **Judge with the same blind spot.** A same-model reviewer approves a candidate
  that violates a rule both model and candidate misread. Agreement is not
  independence; use deterministic checks or a differently constructed judge.
- **Cache carries a success.** A cached evaluation result is keyed only by task
  name and is reused after the candidate, model or tool version changed. The
  cached pass belongs to a different experiment.

## Hard constraints before scoring

Check these before comparing quality, cost or latency. A candidate that violates
one is rejected; high averages do not compensate.

```text
no privilege escalation        no evaluator or criteria modification
no sealed-label access         no data exfiltration or hidden logging
no native-test bypass          no budget overrun
```

Anti-hacking counters, because each of these looks like success:

| Tempting metric | Why it is not improvement |
| --- | --- |
| fewer rules or shorter text | coverage and triggers can be lost |
| fewer tool calls | the task may be incomplete |
| higher pass rate after removing tests | the gate was weakened |
| lower self-reported cost | no independent bill or token count |
| faster on one workload | may be cold-path regression or noise |
| more candidates submitted | volume is not value |

## Judges and human checkpoints

An LLM judge can triage failures, explain a diff and propose counterexamples. Use
it only with:

- a calibration set with known labels and stated agreement thresholds;
- deterministic checks for anything mechanically checkable;
- sampled human review, recorded with who reviewed and how much;
- the judge's own prompt and model version recorded, since a judge change is a
  measurement change.

Replace the judge for final acceptance with observed execution, or report the
result as judged rather than executed.

## Budget

Evaluation costs tokens, requests, wall time, concurrency and money. Decide the
budget before the loop, count what was spent, and stop at the bound. Cover
candidate generation, reviewer calls, retries, environment setup and any human
review time. A cost that was not measured is `unknown`, not zero; a reduction the
candidate reports about itself is not a measurement.

## Result record

```text
candidate     digest d, envelope e, base revision b
split         validation v-3; sealed set not opened (0 accesses)
criteria      c-1..c-5, frozen 2026-11-02
baseline      p-7, same env, same inputs
results       c-1 pass, c-2 pass, c-3 failed (case 41), c-4 blocked (no store),
              c-5 not_run
cost          measured: tokens, 3 min wall; unknown: reviewer human time
verdict       not ready: c-3 regression and c-4 unverified
```

A verdict names the missing evidence and the failed criterion, not only a score.
