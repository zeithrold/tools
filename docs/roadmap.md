# Roadmap

This document separates drafted guidance and the offline candidate slice from
proposed next stages. Current capabilities are listed in the [README](../README.md);
none of these statuses alone establish behavioral improvement.
Installation and authentication contracts live in [distribution](distribution.md),
[global Skills](global-skills.md) and [GitHub App setup](github-app-setup.md).

## Skill improvements

Extend existing Skills before adding overlapping guidance. Generic mechanisms
belong here; business rules, deployment policy and project knowledge remain with
consumers. Distribute only explicitly selected Skills to the user's global tree.

| Priority | Work | Source status and scope |
| --- | --- | --- |
| First | Planning, verification, code review and Skill authoring | Drafted with conditional methods, examples and independent-acceptance boundaries |
| First | Native Go/JS/frontend testing | Absorbed into verification references; no parallel testing Skill IDs |
| Next | Team coordination, debugging and backend engineering | Drafted: handoffs/isolation, falsifying probes, boundary/persistence/recovery methods |
| Next | UI/frontend decisions | UI foundation/Web/Flutter consolidated into principle-led ui-design; engineering keeps architecture/state |
| Later | Performance engineering and evolution proposals | Guidance drafted; measured benefit and the complete evolution runtime remain unverified/unimplemented |

These are capability areas, not commitments to separate directories. Merge
overlapping responsibilities rather than reproducing every upstream workflow.

Use the [adoption map](skill-adoption.md) to track each researched mechanism's
destination and remaining acceptance work. New prose is original synthesis, not
copied upstream templates. Structural and synchronization checks do not establish
behavioral improvement. [Decision screening](../internal/skills/testdata/workflow-evaluation.json)
and the [synthetic source-review fixture](../internal/skills/testdata/review-fixture/README.md)
cover only limited questions; retain run results separately and independently
compare the pinned final Skill group against research before delivery.

### Selection and acceptance

- Questions and mechanical documentation fixes need focused context and content
  checks, not mandatory planning, delegation or full test suites.
- Local behavior fixes need domain guidance and a targeted regression. Add
  debugging when the cause is unknown.
- Cross-module changes need explicit interfaces and acceptance criteria; delegate
  only genuinely independent work.
- Authorization, persistence, migration and concurrency changes need recovery
  planning, complementary independent review and relevant integration negatives.
  Urgency does not waive these gates.
- Performance work starts with a workload and baseline, not speculative caching.
- Skill changes need trigger/non-trigger cases, an unchanged-guidance baseline
  and independent behavioral evaluation. Static Markdown checks are insufficient.

For each adopted mechanism, record the source revision and file, license,
destination, adaptation rationale and validation method. Unknown redistribution
rights block copying; rewriting does not erase upstream obligations. Preserve
licenses and references through embedding and synchronization.

Evaluate incorrect activation, missed activation, permission violations and
regressions under time pressure or missing tools. Fix the evaluation cases before
running candidates, keep reviewers independent and report unmeasured cost as
unknown. Store run evidence in ignored artifacts or CI, not this roadmap.

Useful source anchors, not pre-approved material for copying:

- [Risk-based review](https://github.com/openai/openai-agents-python/blob/0dc4cf152bff1f50bdc0e41f28808563406545b2/.agents/skills/implementation-final-review/SKILL.md)
- [Task planning](https://github.com/obra/superpowers/blob/2ce8aa15d05bd57213e6f746f953aafb7ded7a8b/skills/writing-plans/SKILL.md)
- [Independent acceptance](https://github.com/OpenHands/OpenHands/blob/b7fdc6c312039a0b840fdcd34a5c23ca7e65c218/AGENTS.md)
- [Frontend ownership](https://github.com/mastra-ai/mastra/blob/d49143a299757859bdbca93aa4b688c1383ace1e/.claude/skills/mastra-frontend/SKILL.md)
- [Performance workflow](https://github.com/openclaw/openclaw/blob/db38cc1066957e278d529d531160bb2773163682/.agents/skills/openclaw-performance/SKILL.md)

## Evolution runtime: offline stage and remaining work

The [offline candidate stage](evolve.md) provides structured replacement
`prepare` and read-only `plan`. It does not execute verification or implement
remote submission/status/activation. The table below describes the full target
workflow, not additional shipped commands.

The recommended MVP is a local candidate, read-only plan, explicit verification
and idempotent Draft PR followed by human review. It does not include model
training, automatic merging, deployment or package publication.

| Proposed operation | Contract |
| --- | --- |
| `prepare` | Accept an explicit upstream patch and reviewed evidence; write an immutable candidate only under owned artifacts |
| `plan` | Read the candidate, trusted policy and cached baseline without network, credentials, tests, installs or writes |
| `check` | Run an explicit native profile in a reviewed trusted or isolated environment and bind results to candidate identity |
| `submit` | Require authorization for the reviewed digest, target, base and public payload; create a dedicated branch and Draft PR |
| `status` | Report local observations; use explicit remote reads to reconcile the PR, exact head and checks |

### Required boundaries

- Start with changes under explicitly allowed upstream `skills/<id>/` paths.
  Root instructions, runtime code, workflows, credentials and package manifests
  are outside the initial automatic write scope.
- Keep consumer module configuration, trusted operator policy and OS credentials
  separate. A candidate cannot expand its paths, budget, evaluator or authority.
- Use the existing GitHub App Device Flow and native credential adapters. App
  installation and authentication do not authorize publication. Do not pass
  tokens to candidate processes or fall back to PATs or plaintext files.
- Bind approval to the canonical manifest, patch, base tree, target, validation
  contract and public evidence. Changes invalidate approval and relevant results.
  Keep mutable submission journals outside immutable candidate identity.
- Require known provenance and compatible licensing. Publish only reviewed,
  bounded payloads, not transcripts, reasoning, secrets, private source, absolute
  paths or raw logs. Hashes are not privacy checks.
- Validate real filesystem paths and content identity; reject traversal,
  symlink escapes, unsupported file types and content changes during use.
- Run untrusted candidates without credentials in an appropriate sandbox.
  Existing argv handling and process timeouts are not an OS security boundary.
- Missing environments or checks remain blocked/not-run. Agent-reported results
  are not independently observed execution or platform-issued proof.

### Delivery sequence

1. Freeze candidate schema, trusted policy storage, allowed paths, authorization,
   validation contract and publication budget.
2. Implement offline prepare/plan with synthetic upstreams and negative tests.
3. Bind native verification and independent review to exact candidate identity.
4. Implement Git data/PR API submission with deterministic branches and a durable
   journal. Reconcile unknown write outcomes through reads; never force-push,
   overwrite unrelated work or blindly duplicate commits/PRs.
5. Validate an explicitly authorized sandbox PR and its real head-specific checks.
   A created PR is not proof that CI ran. Keep candidate execution separate from
   write-token jobs; never execute an untrusted head in `pull_request_target`.

If a matching PR already exists, return it without writes, even if the base has
moved. If further writes are needed against a changed base, require renewed
preparation, verification and review. Closed proposals are not reopened
automatically.

The MVP is complete only after a real authorized prepare → check → submit →
status flow demonstrates privacy, idempotency, failure recovery and CI behavior.
Unattended submission, Issue integration and model-driven candidate generation
remain later work. Sandbox target, execution isolation and evaluation budget
must be decided before implementation; this roadmap authorizes none of them.
