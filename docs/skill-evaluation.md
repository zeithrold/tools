# Native Skill evaluation baseline

Use an existing benchmark, stock Harbor Agent integration, Docker and the
task's original verifier. The generic methodology is owned by
[skill-authoring's task evaluation reference](../skills/skill-authoring/references/task-evaluation.md).
Read it for natural activation, version comparisons, benchmark fit and scheduling.
This page owns the repository's concrete execution recipe, not a new evaluator.

## Start with the native path

Read Harbor's [quick start](https://docs.harborframework.com/getting-started/quick-start.md),
[job configuration](https://docs.harborframework.com/jobs/configs.md),
[Skills](https://docs.harborframework.com/jobs/skills.md), and
[authentication](https://docs.harborframework.com/agents/agent-auth-resolution.md).
Check the installed CLI help. The local baseline used Harbor 0.25.0 and
OpenCode 1.19.4; these are compatibility references, not permanent latest versions.
Runtime changes start a new comparison cohort.

With Harbor on PATH and the explicitly approved inference credential exported
as `DEEPSEEK_API_KEY`, the initial wiring command is:

```sh
harbor run -t hello-world/hello-world \
  -a opencode -m deepseek/deepseek-flash --ak version=1.19.4 \
  --ak 'opencode_config={"snapshot":false}' \
  -n 1 -k 1 -r 0 \
  -o .artifacts/skill-evaluation/jobs
```

The OpenCode snapshot setting is a native workaround for observed EPIPE
behavior, not a proven root-cause fix or a universal requirement.
Do not add lifecycle hooks or modify framework sources to hide an exception.
The hello-world task is only wiring evidence.

## One informative paired experiment

For example, `cancel-async-tasks` in Terminal-Bench 2.0 exercises concurrency
and cancellation recovery. Select the task before seeing this cohort's outcomes.
Read its declared limits: its 900-second Agent limit becomes approximately
300 seconds with the multiplier below.

For prompted-use evaluation, run this command once without the optional Skill
argument and once with `--skill ./skills/backend-engineering`:

```sh
harbor run -d terminal-bench@2.0 -i cancel-async-tasks \
  -a opencode -m deepseek/deepseek-flash --ak version=1.19.4 \
  --ak 'opencode_config={"snapshot":false}' \
  --extra-instruction 'If backend-engineering is available, load it with the native skill tool before implementing. Otherwise proceed normally. Solve the original task.' \
  --agent-timeout-multiplier 0.3333333333333333 \
  -n 1 -k 1 -r 0 \
  -o .artifacts/skill-evaluation/jobs
```

On an arm64 host, when the selected task image requires amd64, use Harbor's native
`--extra-docker-compose` option with this operator-owned overlay:

```yaml
services:
  main:
    platform: linux/amd64
```

Give runs distinct job names or use Harbor's generated names; never overwrite
historical results. Record the resolved task commit from native locks. For exact
task pinning, use native `tasks[].git_url`, `git_commit_id` and `path` fields.
Keep credentials out of arguments, committed configs and reports. Native credential
routing is allowed within explicit authorization; it does not grant access to
unrelated host credentials or production.

For **natural activation**, remove the extra loading instruction from both arms
and do not name or hint at a Skill in the task prompt.
For **version regression**, supply old and candidate full Skill directories,
keep the chosen loading mode unchanged, and use immutable copies without
mounting both versions in one participant environment.

## Parallel execution and evidence

Prefer one native config with two `agents` entries and
`n_concurrent_trials: 2`, rather than a custom process scheduler.
Each entry keeps the same Agent/runtime settings and differs only in Skill input.
Concurrency does not create repetitions: set `n_attempts` separately.
Run precise performance comparisons and timing-sensitive graders serially.

Inspect native trial `result.json`, `lock.json`, `agent/opencode.txt` or
`trajectory.json`, and verifier output. Confirm the actual body load and read
exceptions/limits even when reward is positive or the CLI exits zero.
Use native records rather than inventing a second success state.
Native aggregate metrics may group different Skill configurations under one
Agent/model label. Separate arms using trial locks; an aggregate `Pass@2` across
two different conditions is not two repeated attempts of either condition.

## Maintain a small evaluation schedule

1. Wiring smoke only when the relevant execution path changes.
2. A declared development screen for a new Skill or mechanism.
3. Old/new paired repetitions on affected and unrelated regression cases.
4. Separate held-out acceptance before claiming broad benefit.
5. Contemporaneous re-baselining after model, Agent or environment drift.

Keep experiment manifests, scores, incident lessons and interrupted-run receipts
under ignored `.artifacts/` or CI artifacts. Record what did not run. Do not turn
these receipts into source documentation or retroactively describe them as frozen.
This guide changes evaluation procedure; it does not certify any Skill's quality
or authorize global Skill installation, publication or an unbounded experiment.
