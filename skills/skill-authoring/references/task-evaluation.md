# Executed Skill evaluation

Use this reference to compare real Agent behavior with tools and independently
graded outputs. It complements evaluation-design.md; it is not a runner.

## Ask one question per comparison

| Question | Conditions | Primary observation |
| --- | --- | --- |
| Does offering this Skill improve real use? | Absent / available, unchanged task prompt, no Skill name or loading hint | Task outcomes and cost across all trials |
| Can the Agent discover and use it naturally? | Available, ordinary task prompt; include positive and near-miss tasks | Discovery, selection, exact body loads and timing |
| Does using the guidance help? | Absent / available, same conditional instruction in both arms to load if available | Outcomes after observed prompted loading |
| Is the new version better? | Full old version / full candidate, same loading mode | Paired quality regressions, gains and cost |
| Did only the trigger improve? | Old / new description with body held constant | False/missed activation on fixed cases |
| Does the body help beyond its description? | Description-only / full Skill, where the host supports this distinction | Ablation result; not a no-Skill comparison |
| Does the library compose well? | Target Skill alone / selected group | Interference, redundant loading and final outcomes |

Do not label prompted loading as natural activation. Keep the natural prompt
identical across arms: naming the target or asking the Agent to choose a Skill
is a hint, even without an explicit load command. Confirm that the catalog is
actually exposed by the host. Files installed, catalog visible, body loaded,
references read and behavior changed are different observations.

Use native tool events and returned content, not the Agent's closing claim.
Shell reads can count as body access when evidenced; do not silently call them
native Skill-tool activation. Repeated/cached loads may not emit another event.
Missing visibility is unknown, not proof of non-use.

The tested system is model × Agent × Skill × task × environment × budget.
This measures system behavior, not a change to the model's weights.

## Native path before infrastructure

Read the selected framework's current quick start, task, credential, Skill,
budget and result documentation; check the installed version's help.
Run one existing task to completion using its native Agent and verifier.
A successful install or a correct answer with an execution exception is not
that milestone. If necessary, use a documented runtime option and record it.

After this wiring smoke, choose an informative task; a trivial smoke cannot
establish benefit. Use configuration to express arms, concurrency and limits.
Do not build an Agent loop, model proxy, discovery hook or grading engine just
to start a pilot. A capability mismatch calls for a scoped decision, not endless
adapter patches. Set the actual authorized isolation and credential route once;
optional hardening is not an unapproved prerequisite or permission to ignore
required controls.

Keep small pilots small: native locks, logs and one selection/condition record
usually suffice. Do not duplicate every native record in a second state machine.
Observe body loading after registration; do not reject trials using a probe
before the framework makes that state available.

## Choose a Bench for the mechanism, not its popularity

Before model outcomes, write the behavior the Skill should change and how the
task's oracle would expose its absence.

| Skill mechanism | Suitable task evidence | Important blind spot |
| --- | --- | --- |
| Debugging | Unknown-cause defect; reproduction, repair and regression checks | A disclosed one-line fix barely exercises investigation |
| Verification | Plausible incorrect solution that native checks can reject | Correct final output alone does not show honest testing |
| Backend/concurrency | Cancellation, retries, conflicts and recovery invariants | One happy-path request does not exercise recovery |
| Performance | Defined workload, equivalent outputs and measured improvement | Correctness-only grading cannot establish speedup |
| Review | Pinned defect discovery plus confirmation and false-positive cases | Repair success is only a proxy for independent review |
| Planning/teamwork | Executable handoff, dependency or integration outcomes | Individual coding reward cannot certify coordination |
| UI/frontend | Rendered states, interaction, accessibility and relevant runtime checks | Terminal tests cannot certify visual quality |
| Skill authoring/evolution | Trigger cases, controlled revision and independent acceptance | Generating convincing guidance is not demonstrated improvement |

Use only task instructions, metadata and environment requirements for selection;
do not let hidden tests or reference patches guide the participant.
Check architecture, build needs and advertised limits before spending on a batch.
Use the native verifier unchanged where it measures the claim. If it does not,
report the uncovered claim or choose a suitable task; do not quietly substitute
an easier score. Additional judgments require separately frozen criteria.

Include intended-use tasks, near misses and at least one unrelated regression
family when claiming breadth. No suitable Bench means untested, not ineffective.
Floor/ceiling results can motivate a separately declared development cohort;
retain the first cohort rather than shopping for a task that favors a version.
Do not tune a candidate on final acceptance cases.

## Freeze a cohort, not the entire project

Before starting, record task IDs/revisions, arm definitions, Skill content,
shared prompt, model/Agent/runtime configuration, all effective stopping limits,
attempt count, schedule, oracle and how interruptions will be treated.
Include inherited message, tool, token and time caps, not just explicit CLI flags.
Use existing native revision records only when they identify the actual evaluated
inputs, including candidate changes; otherwise retain the exact inputs. A baseline
or HEAD that omits uncommitted changes is insufficient. Keep case labels outside
participant inputs. Do not put answers, previous results or other arms' artifacts
in its tree.

Verify resolved configuration semantically: serializers may omit default fields.
An absent optional empty list is not necessarily a broken configuration.
If the chosen preparation check fails, execution must not fall through to launch.
Bind preparation and launch with ordinary shell error handling, not a new runner.
Never reconstruct a missing pre-run freeze after observing the outcome.

## Schedule by decision

| Stage | Bounded example | Decision supported |
| --- | --- | --- |
| Wiring | One native task, then loading smoke if needed | Execution and evidence collection work |
| Development screen | One matched task × two conditions × one attempt | A descriptive signal worth investigating |
| Replication | Predeclare, for example, three paired repetitions on several task families | Consistency, disagreements and likely regressions; not automatic statistical significance |
| Revision comparison | Old/new on affected and unrelated regression families | Whether the revision's gains justify its costs and regressions |
| Acceptance | Fixed cases not used to author/select the candidate, independent judgment | A scoped adoption decision |
| Maintenance | Rerun affected cases after relevant changes, periodically refresh the suite | Detect drift without pretending old evidence is fresh |

These are example sizes, not a universal test quota. Choose repetitions, budget
and stopping rule before seeing scores; do not keep running until a win appears.
For more than a pilot, choose the sample size and uncertainty analysis for the
decision rather than attaching significance to a handful of correlated cases.

Pair by task and repetition. Counterbalance order (old/new then new/old), or
randomize it with a recorded schedule. Different contexts and fresh workspaces
are required; an identical seed, where supported, does not ensure identical
hosted model behavior. Record unsupported seed/temperature controls as such.

Parallel quality trials are appropriate with independent workspaces and enough
resources. Use native concurrency limits. For time-sensitive graders or precise
speed measurements, run serially with matched resource/cache conditions.
Do not pool serial and parallel durations as a speedup estimate. API throttling
and setup contention are environment observations, not automatic Skill defects.

Keep task, runtime and Skill identities together. Compare old and new versions
contemporaneously after model, Agent, dependency or environment changes; never
credit the candidate with a gain caused by a new runtime. If trigger and body
both changed, use old/new natural and prompted comparisons when attribution
matters; do not require every ablation for a small body-only edit.

## Read outcomes without hiding failures

Report every planned cell: native reward and subtests, errors/limits,
availability/body-load evidence, execution/setup durations, token/cache fields,
retries and estimated cost versus actual billing. Successful process completion,
correct task output and complete evaluation coverage are separate facts.

Keep the all-trial denominator for natural availability effects. A table limited
to successfully loaded trials is diagnostic and selected after treatment; it is
not an unbiased benefit estimate. Report non-loading and infrastructure failures
separately, with counts; neither silently drop them nor relabel all as task zeros.
Legitimate task failures and budget exhaustion remain visible outcomes under
the frozen limits.

For paired binary outcomes show both-pass, both-fail, old-only and new-only counts;
show per-task results rather than one pooled average hiding regressions.
For continuous scores retain paired differences. Describe uncertainty appropriate
to the sample, task clustering and stochastic model. One positive pair is a signal,
not a general success-rate claim. Cheaper failure is not efficiency; compare cost
alongside quality and distinguish setup, Agent time and grader time.

On interruption, preserve native logs, stop only owned work, reconcile resources
and mark incomplete cells. A stale "running" record is not live progress.
Any replacement run is a new identified cohort with a reason; no silent retries.

## Provenance and scope

Original procedural synthesis extending evaluation-design.md at repository base
553629217641a719ee0ce06f6afae756c216ffdb, informed by owner-directed local evaluation
and the distinction between activation, use and independent outcome evidence.
No upstream templates, benchmark tests or private transcripts are reproduced.
First-party text follows the repository MIT license.

Native-operation concepts were cross-checked against Harbor's public
[job configuration](https://docs.harborframework.com/jobs/configs.md) and
[Skill staging](https://docs.harborframework.com/jobs/skills.md) documentation.
Those pages are mutable; verify the installed framework rather than treating
them as a pinned runtime contract. Worked run receipts and incident histories
belong in ignored artifacts, not this reusable guide. These rules are a candidate
method, not evidence of improved Skill behavior; validate changed decisions on
new tasks before making that claim.
