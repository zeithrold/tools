---
name: skill-authoring
description: Use when creating or revising a reusable Skill, covering triggers and exclusions, ownership, progressive references, provenance, minimal guidance and behavioral evaluation. Not for ordinary product implementation or one-off task notes.
license: MIT
---

# Skill authoring

A Skill is reusable guidance selected for a task, not a corpus to copy or a
workflow engine. Apply the project's own instructions when they are stricter.

1. Check ownership before writing. Extend the existing Skill, reference or
   project instruction that already owns the mechanism; keep business rules,
   schemas, credentials and deployment policy with their project. Remove
   duplication instead of adding a parallel rule that says the same thing.
2. Declare triggering and exclusion in the frontmatter description, which is
   the indexed trigger text. Name observable task conditions that should
   activate the Skill and near-miss work that should not. Avoid vague
   adjectives, and do not fold unrelated workflows into one trigger.
3. Keep guidance minimal. Each rule should change a decision or prevent a known
   failure; prefer a short statement with its reason over templates, checklists
   and ceremony. Cut advice a competent model already follows by default.
4. Put background, matrices, long examples and variant details in
   `references/` and say when each file must be read. The main file must work
   without them, and loading every reference should not be routine.
5. Record provenance for every adopted mechanism, including conceptual adoption
   where no text is copied: source revision and file, license, what was taken,
   destination, changes and validation. Unknown redistribution rights block
   copying, and rewriting does not remove upstream obligations. Preserve
   required notices; never include secrets or private source. Templates are not
   forbidden wholesale, but keep operational examples and worked cases in
   `references/` and strip project-specific policy. Read
   [provenance](references/provenance.md).
6. Separate the evaluation question from the claim. A decision-only trial,
   where the agent describes actions without tools, screens policy decisions
   but cannot show tool use or task behavior. A no-Skill baseline measures
   natural behavior; descriptions-only is a trigger-text ablation, not the same
   baseline. Freeze cases, criteria, permissions, trial count and budget before
   trials, use the same cases for every condition and run, include permission
   negatives, and add held-out task families before generalizing. Read
   [evaluation design](references/evaluation-design.md).
7. Apply the hard safety gate before quality: an unauthorized action, changed
   evaluation asset, sealed-data access or budget overrun rejects the candidate
   regardless of score. Keep static checks separate from behavioral evidence:
   frontmatter, links, license and line-level clarity are static checks, and they
   do not show activation or changed behavior. Run each frozen case in a fresh
   context for every condition; a fresh context is not isolation. If an isolated,
   credential-free environment is unavailable, report execution as blocked, not
   permission to run on the host. Compare outcomes, cost, false activation and
   missed activation rather than prose. Writing evaluation instructions does not
   itself prove efficacy.
8. Budget development and acceptance separately. Set revision and trial limits
   before tuning and stop at them rather than adding cases after seeing results;
   do not keep editing after improvement stalls, and a change that only fits the
   frozen cases is overfit. Author-run trials support development; acceptance
   needs an independent judge working from the frozen criteria, and another
   prompt or more same-model reviewers is not independence. Without independent
   behavioral judgement, report acceptance as unverified and name who judged the
   development results. Label carried-over evidence as reused.
9. Distribute explicitly. Name the intended tree and the selected Skills; never
   install all Skills, overwrite unmanaged files or select Skills silently.
   Installed guidance grants no new permissions and remains advisory.
