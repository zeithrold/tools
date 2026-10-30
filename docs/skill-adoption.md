# Research adoption map

Track mechanisms, not the number of Skill directories. **Draft guidance** means
instructions and examples exist; it does not mean host activation, improved task
outcomes or safe unattended execution have been demonstrated. Run receipts belong
in ignored artifacts or CI. The [roadmap](roadmap.md) tracks implementation order.

The separately recorded [HERO adoption](hero-adoption.md) includes a verbatim
global prompt block and its retained MIT license. It is not one of the conceptual
synthesis rows below; their no-template and unaudited-source-terms statements do
not describe that adoption.

The source keys below identify fixed conceptual inputs. New prose is original
synthesis, not redistributed upstream templates, code or scripts. This map does
not certify upstream file licenses: copying or adapting protected expression
still requires a file/version-specific rights check and applicable notices.

For every row below, the material identity is **original conceptual synthesis**,
not a copied or translated upstream artifact. Source terms have **not been
re-audited for redistribution**; that is an explicit unknown, not licensing
clearance or a claim that copying would be allowed. Mechanisms were selected
because they change a decision or expose a concrete failure, and adapted to
explicit consumer ownership, bounded cost and no implicit publication. Stronger
authorization/isolation limits are this repository's constraints, not claims
about upstream defaults. Exact reference destinations below identify the adopted
method; final content review is separate from unestablished behavioral benefit.

Before delivery, independently compare the pinned final Skill tree with this
map, the research inputs and the retired-content inventory. Classify each
mechanism as operational guidance, partial, deferred or deliberately rejected;
separate static/distribution checks from real task outcomes. Report omissions,
contradictions, activation gaps and untested claims instead of equating a
directory or green tooling suite with adoption. Re-review affected conclusions
after content changes. Keep the run-specific judgement in ignored artifacts or
CI and summarize remaining gaps in the handoff.

## Mechanism, destination and acceptance

| ID | Mechanism and source keys | Destination / disposition | Acceptance still needed beyond static checks |
| --- | --- | --- | --- |
| T1 | Independent tasks, complete handoff and one shared-interface owner — GSD, WT | [Handoff](../skills/agent-team/references/handoff.md) and [integration](../skills/agent-team/references/integration.md) | Two disjoint tasks plus a shared-file conflict; parent verifies both results without duplicate work |
| T2 | Context, filesystem and tool permissions are distinct — OC, OA-R | [Isolation](../skills/agent-team/references/isolation.md) | Show that a fresh context does not confer a worktree or write restriction; use actual host capabilities |
| T3 | Completion notification, recursion/cancellation and bounded coordination — OC, GSD | [Team protocol](../skills/agent-team/SKILL.md) and [lifecycle](../skills/agent-team/references/isolation.md) | Delayed completion, failed child and cancelled parent; no liveness polling or unbounded recursion |
| P1 | Goals, real consumers, consumes/produces and observable task checks — SP-P, DF | [Work items](../skills/task-planning/references/work-items.md) | A second executor can act without inventing interfaces or missing dependencies |
| P2 | Required behavior, compatibility, unsupported cases and alternatives — OA-P | [Scope contract](../skills/task-planning/references/scope-contract.md) | Compatibility and exclusion decisions identify an existing alternative or explicitly state there is none |
| P3 | Complexity reset and stop/reduce/revise/seek-evidence decisions — OA-P, DF | Draft: task-planning | Accumulating related patches lead to a design decision, not endless local repairs |
| P4 | Separate requirement, design, tasks and progress semantics — SK, OS | Draft: task-planning; no new artifact-graph engine | The implementation checklist cannot silently change the requirement; small work need not create four documents |
| R1 | Failure-mode specialists and risk-scaled complementary review — OA-R, CC, BM | [Review lenses](../skills/code-review/references/lenses.md) | Relevant lenses catch distinct defects without assigning every lens to every task |
| R2 | Frozen independent discovery, candidate confirmation and version-specific findings — OA-R, CC, QW | [Brief](../skills/code-review/references/review-brief.md) and [confirmation](../skills/code-review/references/disposition.md) | A real defect is confirmed, a plausible false positive rejected, and a changed head re-reviewed |
| R3 | Verification-gap questions and intent/design/patch/defer routing — BM, ADK | [Lenses](../skills/code-review/references/lenses.md) and [disposition](../skills/code-review/references/disposition.md) | Identify a real regression that leaves current tests green; route a bad requirement differently from a code defect |
| F1 | Small visual/experience decision package with existing design authority — AN, BM | Consolidated [ui-design](../skills/ui-design/SKILL.md) + [design decisions](../skills/ui-design/references/design-decisions.md) | A new flow resolves hierarchy, state and recovery before CSS; a typo needs no package |
| F2 | Appearance versus layout ownership and runtime input boundaries — MA, OC | Existing engineering + draft: [boundaries](../skills/frontend-engineering/references/boundaries.md) | Features compose layout without hiding a conflicting component appearance override |
| F3 | Restore/migrate/reset and stale-response lifecycle — OH, MA | Existing preferences + draft: [state lifecycle](../skills/frontend-engineering/references/state-lifecycle.md) | Out-of-order response, failed restore and reset cannot resurrect obsolete state |
| F4 | Useful rendered content and cache freshness are separate claims — NX | Draft: [async and cache](../skills/frontend-engineering/references/async-and-cache.md) | Built-runtime cold/warm and post-mutation checks show useful, correctly scoped data |
| B1 | Shape/range validation versus authorization/existence/conflict at the owner — MA, OC | Draft: [backend-engineering](../skills/backend-engineering/SKILL.md) | A typed input still fails a contextual permission/conflict check at execution |
| B2 | Serializable contracts, narrow core and extension-first mechanisms — OC, DF | Draft: backend-engineering | A synthetic consumer supplies policy without a project-name exception or service object in a wire schema |
| B3 | Persisted compatibility, startup DDL and distributed races — ADK | [Persistence/concurrency](../skills/backend-engineering/references/persistence-and-concurrency.md) | Old external store, concurrent startup and interrupted migration scenarios are handled explicitly |
| B4 | Durable acceptance versus execution and scoped idempotency — OC | [Persistence/concurrency](../skills/backend-engineering/references/persistence-and-concurrency.md) and [scenarios](../skills/backend-engineering/references/scenarios.md) | Lost response and repeated delivery reconcile without a false exactly-once claim |
| V1 | Discovery/readiness/execution/result distinction and native entry points — OH, QW | Existing CLI + draft: [verification](../skills/verification/SKILL.md) | Missing tools, wrong symptoms and skipped commands never become passes |
| V2 | Installed baseline versus local candidate; recipe/environment/symptom identity — OH, QW | [Reproduction](../skills/verification/references/reproduction.md) and [debugging probes](../skills/systematic-debugging/references/probes.md) | A real reproduction is distinguished from unrelated failure; candidate evidence names the actual build |
| V3 | Independently derived oracles and falsifiable tests — SP-T | [Go](../skills/verification/references/go.md) / [JS/TS](../skills/verification/references/js-ts.md) under verification | Wrong parameter, branch or missing side effect makes the regression fail without changing its expected answer |
| V4 | Independent acceptance assets are not adjusted to fit implementation — OH | [Acceptance assets](../skills/verification/references/acceptance-assets.md) | A candidate cannot repair its score by changing the protected recipe or expected result; ordinary regression authoring remains allowed |
| V5 | Targeted iteration, final gates and explicit evidence reuse/invalidation — GE, NB | [Evidence reuse](../skills/verification/references/evidence-reuse.md) | Reuse names the intervening diff and unaffected contract; changed environment/content invalidates relevant proof |
| D1 | Hypothesis, falsifying probe, root cause and reversible mitigation — QW, RX | [Probes](../skills/systematic-debugging/references/probes.md) and [scenarios](../skills/systematic-debugging/references/scenarios.md) | A misleading symptom and failed hypothesis change the investigation, not merely trigger more retries |
| E1 | Relevant-stage loading and progressive references — PI, OH | Existing routing + draft references | Relevant details are loaded; unrelated guides are not made mandatory |
| E2 | Measure, remove work, compute once, pass facts, batch, cache last — CL | [Optimization order](../skills/performance-engineering/references/optimization-order.md) | Work counts and paired interleaved timings support the chosen optimization, including correctness |
| E3 | Cold/warm separation, workload identity and bounded experiments — CL | [Measurement](../skills/performance-engineering/references/measurement.md) | Results cannot mix cold and warm runs or claim an unmeasured cost saving |
| S1 | Precise triggers/exclusions, existing ownership and provenance — SP-S, HE | [Authoring](../skills/skill-authoring/SKILL.md) and [provenance](../skills/skill-authoring/references/provenance.md) | Duplicate and near-miss tasks do not produce universal activation or a redundant Skill |
| S2 | Real tasks, no-guidance baseline, body ablation and independent acceptance — SP-S, OH | [Evaluation design](../skills/skill-authoring/references/evaluation-design.md); [decision screening](../internal/skills/testdata/workflow-evaluation.json) is limited | Real task outcomes, false/missed activation and hard safety checks; decision answers alone are insufficient |
| X1 | Facts versus procedures; correct existing rules, not incident accumulation — HE, CL-X | Draft: [evolution-proposals](../skills/evolution-proposals/SKILL.md) | An unresolved failure or private user fact is not promoted into a reliable generic procedure |
| X2 | Source lineage, trust, expiry/revocation and contamination retention — CL-X, LM | [Envelope](../skills/evolution-proposals/references/candidate-envelope.md); runtime enforcement deferred | A source change/revocation invalidates dependent proposals; summaries cannot launder trust |
| X3 | Independent permissions/data/evidence; safety before mean score — GP, OH | [Evaluation](../skills/evolution-proposals/references/evaluation.md); evaluator enforcement deferred | Candidate development cannot alter the evaluator or read sealed answers; no benefit offsets a safety violation |
| X4 | Candidate identity, reviewed activation, recoverable versions and bounded budgets — HE, CL-X | [Activation](../skills/evolution-proposals/references/activation.md); `zt evolve` provides prepare/plan, independent check and recoverable submit/status; activation deferred | Candidate identity and upload previews support PR review. Recoverable activation and a complete isolated evaluation budget/capability boundary remain unimplemented |

## Deliberate exclusions and deferrals

- Do not impose every expert or a multi-Agent team on every task. Independent
  work and failure modes decide staffing, not role count.
- Do not copy production domains, credentials, business schemas, deployment policy
  or dangerous sandbox-bypass defaults. Consumers own those decisions.
- Do not treat synthetic workers, static document validators, successful scan
  completion or model confidence as task success.
- Do not implement automatic Skill deletion from incomplete usage statistics,
  background extraction from arbitrary sessions, self-modifying root rules or
  automatic merge/deployment.
- Host prompt-cache internals, file permissions and budget enforcement need host
  interfaces, not universal instructions pretending to enforce them.
- Full independent candidate evaluation, sandboxed execution,
  activation/rollback/revocation and offline search remain separate runtime
  stages. The released `zt evolve` lifecycle covers local verification and Draft
  PR submission; a working auth adapter, safe Skill sync or a merged submission
  does not complete the remaining stages.

## Fixed source anchors

Sources were recorded in the earlier read-only research. These anchors identify
the material behind each mechanism; they are not claims that upstream tests,
permissions, defaults or effectiveness were reproduced.

| Key | Fixed source and relevant material |
| --- | --- |
| OA-R | [OpenAI Agents Python review](https://github.com/openai/openai-agents-python/blob/0dc4cf152bff1f50bdc0e41f28808563406545b2/.agents/skills/implementation-final-review/SKILL.md), including its high-risk review reference |
| OA-P | [OpenAI implementation strategy](https://github.com/openai/openai-agents-python/blob/0dc4cf152bff1f50bdc0e41f28808563406545b2/.agents/skills/implementation-strategy/SKILL.md) |
| GSD | [GSD execution](https://github.com/open-gsd/gsd-core/blob/7129a135c7271ccea59df7561df83ac11cbc94bd/skills/gsd-execute-phase/SKILL.md) |
| WT | [Task handoff](https://github.com/wshobson/agents/blob/0874a27b17a22e2b7c2e2bcd725768e0803cad06/plugins/agent-teams/skills/task-coordination-strategies/SKILL.md) |
| OC | [OpenCode boundaries](https://github.com/anomalyco/opencode/blob/5be4b1642bf8d84f69b3c53e24147b85637bba1c/AGENTS.md), [schema ownership](https://github.com/anomalyco/opencode/blob/5be4b1642bf8d84f69b3c53e24147b85637bba1c/packages/schema/AGENTS.md) and [subagent lifecycle](https://github.com/anomalyco/opencode/blob/5be4b1642bf8d84f69b3c53e24147b85637bba1c/packages/core/src/tool/plugin/subagent.ts) |
| SP-P | [Superpowers planning](https://github.com/obra/superpowers/blob/2ce8aa15d05bd57213e6f746f953aafb7ded7a8b/skills/writing-plans/SKILL.md) |
| SP-T | [Superpowers test quality](https://github.com/obra/superpowers/blob/2ce8aa15d05bd57213e6f746f953aafb7ded7a8b/skills/test-driven-development/writing-good-tests.md) |
| SP-S | [Superpowers Skill evaluation](https://github.com/obra/superpowers/blob/2ce8aa15d05bd57213e6f746f953aafb7ded7a8b/skills/writing-skills/SKILL.md) |
| DF | [DeerFlow change decisions](https://github.com/bytedance/deer-flow/blob/5b7daf0e1f563a42817e85d0c6e602cadbdfd678/.agent/skills/engineer-system-change/SKILL.md) and [extension-first rules](https://github.com/bytedance/deer-flow/blob/5b7daf0e1f563a42817e85d0c6e602cadbdfd678/AGENTS.md) |
| CC | [Claude review lenses](https://github.com/anthropics/claude-code/blob/318b7e6ac4e726e88a4ad326214866a93c373b1b/plugins/pr-review-toolkit/commands/review-pr.md) and [finding confirmation](https://github.com/anthropics/claude-code/blob/318b7e6ac4e726e88a4ad326214866a93c373b1b/plugins/code-review/commands/code-review.md) |
| BM | [BMAD verification gap](https://github.com/bmad-code-org/BMAD-METHOD/blob/61e01f82644c4bb533750cfa0d6d61ab61d2cc9b/skills/bmad-build-auto/review-prompts/verification-gap.md), [finding routing](https://github.com/bmad-code-org/BMAD-METHOD/blob/61e01f82644c4bb533750cfa0d6d61ab61d2cc9b/skills/bmad-build-auto/step-04-review.md) and [UX contracts](https://github.com/bmad-code-org/BMAD-METHOD/blob/61e01f82644c4bb533750cfa0d6d61ab61d2cc9b/skills/bmad-ux/SKILL.md) |
| QW | [Qwen review rules](https://github.com/QwenLM/qwen-code/blob/07ae9419e544f3016a8daa2c559a3a9c7a2e5443/AGENTS.md) and [test engineer](https://github.com/QwenLM/qwen-code/blob/07ae9419e544f3016a8daa2c559a3a9c7a2e5443/.qwen/agents/test-engineer.md) |
| ADK | [Google ADK review](https://github.com/google/adk-python/blob/da35ca77d71f7d74c8b7d7550f6e1b591a6690e0/.agents/skills/adk-review/SKILL.md) |
| AN | [Anthropic design decisions](https://github.com/anthropics/skills/blob/dbd4588f9e1033efb41dad4bef2f7947c8993d44/skills/frontend-design/SKILL.md) |
| MA | [Mastra ownership](https://github.com/mastra-ai/mastra/blob/d49143a299757859bdbca93aa4b688c1383ace1e/.claude/skills/mastra-frontend/SKILL.md) and [validation constraints](https://github.com/mastra-ai/mastra/blob/d49143a299757859bdbca93aa4b688c1383ace1e/AGENTS.md) |
| OH | [OpenHands acceptance boundaries](https://github.com/OpenHands/OpenHands/blob/b7fdc6c312039a0b840fdcd34a5c23ca7e65c218/AGENTS.md) and [reproduction identity](https://github.com/OpenHands/OpenHands/blob/b7fdc6c312039a0b840fdcd34a5c23ca7e65c218/.agents/skills/verify-openhands/scripts/lib/known-bug.mjs) |
| NX | [Next.js useful UI check](https://github.com/vercel/next.js/blob/cbf49db30a655ac17c92c2951885c4256db14b9e/skills/next-cache-components-optimizer/SKILL.md) |
| PI | [Pi progressive Skill loader](https://github.com/badlogic/pi-mono/blob/d5b3de22d0e24a3e9c38a9410d3329b099717283/packages/coding-agent/src/core/skills.ts) |
| GE | [Gemini check budgeting](https://github.com/google-gemini/gemini-cli/blob/cd1b45c2db0a5ea7b7fc3b67c1ba5879601d6005/GEMINI.md) |
| NB | [Nanobot evidence reuse](https://github.com/HKUDS/nanobot/blob/d14c54c31cf549cfa2cb5438aaff3d16d51feb6e/.agent/workflow.md) |
| CL | [OpenClaw performance](https://github.com/openclaw/openclaw/blob/db38cc1066957e278d529d531160bb2773163682/.agents/skills/openclaw-performance/SKILL.md) |
| CL-X | [OpenClaw source validity](https://github.com/openclaw/openclaw/blob/db38cc1066957e278d529d531160bb2773163682/src/skills/workshop/experience-review.ts) and [promotion checks](https://github.com/openclaw/openclaw/blob/db38cc1066957e278d529d531160bb2773163682/extensions/memory-core/src/short-term-promotion-apply.ts) |
| HE | [Hermes experience separation](https://github.com/NousResearch/hermes-agent/blob/ad249d9cac01d57523d2a9832810de88f5cf35ee/agent/background_review.py) and [ownership guards](https://github.com/NousResearch/hermes-agent/blob/ad249d9cac01d57523d2a9832810de88f5cf35ee/tools/skill_manager_guards.py) |
| SK | [Spec Kit planning outputs](https://github.com/github/spec-kit/blob/67e0f7294e6e728ab60bf7b4590deb470448bc2d/templates/commands/plan.md) |
| OS | [OpenSpec artifact dependencies](https://github.com/Fission-AI/OpenSpec/blob/6d04a1b58dedc95729d34e4f223d9d483aae210a/src/core/artifact-graph/graph.ts) — dependency semantics, not an adopted graph implementation |
| LM | [LangMem extraction](https://github.com/langchain-ai/langmem/blob/48e3c11f5bb527282c7d5339c6a87a0b35abccfc/src/langmem/knowledge/extraction.py) — conceptual lineage only |
| GP | [GEPA candidate selection](https://github.com/gepa-ai/gepa/blob/7af5f0c5c79fcc16b2d5248b187d0864683edd4d/src/gepa/core/engine.py) — conceptual evaluator/search input; no optimizer code adopted |
| RX | [Reflexion failure feedback](https://github.com/noahshinn/reflexion/blob/218cf0ef1df84b05ce379dd4a8e47f17766733a0/hotpotqa_runs/agents.py) — conceptual falsifiable retry input; no implementation adopted |
