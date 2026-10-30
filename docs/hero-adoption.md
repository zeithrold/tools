# HERO global prompt source

`prompts/AGENTS.md` is the repository-maintained source for the user's global
instructions. It contains the complete first English scope-limit block from
[HERO `RULES.md` at fae28f06b6b6f1582b1554b334dc1525b807cedc](https://github.com/wanshuiyin/HERO-Anti-OverDefense/blob/fae28f06b6b6f1582b1554b334dc1525b807cedc/RULES.md),
copied without rewriting its nine rules, six negative examples, two positive
examples or closing check questions. The user's existing Personal role routing
follows it; the full upstream MIT notice is included in the same file.

The owner selected this conservative interim policy until a stable engineering
loop supports an owner-approved high-benefit hash rule. This is not a claim that
HERO has been shown effective in this environment.

## Loading and ownership

Creating this file does **not** install or load global instructions. The user
must supply its contents, including the license, to the host's global instruction
entry point for the execution and review contexts they use. Do not assume a
repository subdirectory named `prompts` is read automatically by every agent.
No global configuration has been changed by preparing these source files.

Hosts that discover nested `AGENTS.md` files may load this file as project
instructions while working under `prompts/`. That directory-scoped loading,
including the routing and license text, is not global installation.

`zt` distributes Skills, not this prompt. This adoption adds no prompt installer,
hook, sync mechanism, fallback loader or runtime enforcement. The prompt must not
become a network resource that a Skill needs to fetch during normal work.
Task-planning and verification retain their existing task-specific guidance;
they do not duplicate or route to the global block.

The original HERO text is not rewritten to add local exceptions. Higher-priority
instructions and actual project requirements still apply at their own authority.
Required attribution and honest reporting are not silently removed to conform
to a lower-priority prompt. Existing project requirements can be retained under
HERO rule 6; that does not establish that every existing digest satisfies both
conditions of rule 2, or authorize inventing similar requirements for local work.
Runtime hashing, released `evolve` contracts and distribution protocols are not
changed by this prompt split.

## Skill-specific application and provenance

The skill-authoring evaluation references retain exact content or existing
immutable revision records instead of requiring a new hash for every evaluation
freeze. This is a conceptual application of HERO's hash and rubric limits, not
a second copy of the global policy. The independent Skill retains its own
[source note](../skills/skill-authoring/references/provenance.md#hero-evaluation-records)
and [upstream MIT grant](../skills/skill-authoring/references/HERO-LICENSE.txt).

Upstream copyright: 2026 Ruofeng Yang, MIT. No upstream runtime, installer or case
catalogue is bundled. The Personal routing text is supplied by the owner, not
attributed to HERO. No session-specific model assignment is a permanent rule.

## Evidence boundaries

Keep four claims separate: source structure, fidelity to the fixed upstream text,
actual host loading, and behavioral improvement. Native repository checks can
establish structure; source comparison and independent review assess fidelity.
Neither proves that the host loads the global prompt or that agent behavior
improves. Host loading remains uninstalled and behavioral effectiveness unverified
until those workflows are actually exercised. Use the existing native evaluation
approach if a behavioral comparison is later requested; this adoption does not
require a new evaluator, hash ledger, scoring system or approval workflow.
