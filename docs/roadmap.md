# Roadmap

This document separates what the releases provide, the drafted guidance and the
work that still needs acceptance or implementation. Current capabilities are
listed in the [README](../README.md); none of these statuses alone establish
behavioral improvement.
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
| Later | Performance engineering and evolution proposals | Guidance drafted; measured benefit is unverified and the evolution runtime's activation/rollback/revocation remain unimplemented |

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

## Evolution runtime: released slice and remaining acceptance

[v0.2.0](https://github.com/zeithrold/tools/releases/tag/v0.2.0) releases the local
lifecycle described in [the lifecycle contract](evolve.md): structured replacement
`prepare`, read-only `plan`, explicit trusted-local `check`, reviewed Draft PR
`submit` and local/remote `status`. A real-repository trial still requires
approval for each exact public payload, and a release is not evidence that
privacy, idempotency or recovery behaviour was accepted on a live submission;
acceptance of the implementation never authorizes a new candidate's publication.
Activation, rollback and revocation remain unimplemented.

The recommended MVP is a local candidate, read-only plan, explicit verification
and idempotent Draft PR followed by human review. It does not include model
training, automatic merging, deployment or package publication.

| Operation | Contract |
| --- | --- |
| `prepare` | Accept structured replacements, pinned policy/base and declared provenance/evidence; write an immutable candidate only under owned artifacts |
| `plan` | Read the candidate, trusted policy and cached baseline without network, credentials, tests, installs or writes |
| `check` | Run an operator-owned validation contract only after explicit trusted-local execution consent; bind results to the complete base/candidate tree and validation identity; do not claim an OS sandbox |
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

1. Preserve the released v1 candidate/policy identity and the offline
   `prepare`/`plan` guarantees.
2. Trusted-local execution and reviewed submission are released with
   synthetic-upstream and faithful GitHub wire-contract fixtures; keep those
   fixtures green as the surface changes.
3. Close independent execution/credential and publication/recovery findings at
   the exact integrated revision; passing fixtures alone is insufficient.
4. Validate an explicitly authorized sandbox Draft PR and its real head-specific
   checks. A created PR is not proof that CI ran. Keep candidate execution
   separate from write-token jobs; never execute an untrusted head in
   `pull_request_target`.
5. Release only after the relevant source, native-platform and distribution gates.

Submission journals target atomic visibility and recovery from process
interruption, not a guarantee against machine power loss. Unknown remote write
outcomes must be reconciled through reads; never force-push, overwrite unrelated
work or blindly duplicate commits/PRs.

If a matching PR already exists, return it without writes, even if the base has
moved. If further writes are needed against a changed base, require renewed
preparation, verification and review. Closed proposals are not reopened
automatically.

The MVP is complete only after a real authorized prepare → check → submit →
status flow demonstrates privacy, idempotency, failure recovery and CI behavior.
Unattended submission, Issue tracking integration and model-driven candidate
generation remain unimplemented. Sandbox target, execution isolation and
evaluation budget must be decided before implementation; this roadmap authorizes
none of them.

## Tracked follow-ups

`zt run` is deliberately not planned. Deferred capabilities and audit findings
are tracked below rather than treated as prerequisites for ordinary native checks.
Each issue needs its own scoped design and authorization; opening it does not
authorize publication, activation or consumer migration.

| Work | Issue |
| --- | --- |
| Scoped read-only environment readiness (`doctor`) | [#28](https://github.com/zeithrold/tools/issues/28) |
| Conservative native planning for the remaining stacks | [#29](https://github.com/zeithrold/tools/issues/29) |
| Recoverable activation, rollback and revocation | [#30](https://github.com/zeithrold/tools/issues/30) |
| Optional isolated verification for untrusted candidates | [#31](https://github.com/zeithrold/tools/issues/31) |
| Bounded evidence-based candidate generation | [#32](https://github.com/zeithrold/tools/issues/32) |
| Issue integration and controlled orchestration | [#33](https://github.com/zeithrold/tools/issues/33) |
| Stage-only npm OIDC migration and duplicate prevention | [#34](https://github.com/zeithrold/tools/issues/34) |
| CI notice layout and Go dependency-license coverage | [#35](https://github.com/zeithrold/tools/issues/35) |
| Reproduce and correct replacement-file mode discrepancy | [#36](https://github.com/zeithrold/tools/issues/36) |
| Broader natural-activation, revision and workflow/UI evaluation coverage | [#37](https://github.com/zeithrold/tools/issues/37) |
