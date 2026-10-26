---
name: task-planning
description: Use when work crosses a component, data, ownership or contract boundary, carries consequential risk, has unclear dependencies, or expands through proposed abstractions, dependencies, configuration or supporting infrastructure. Covers scope, complexity decisions, acceptance and re-planning. Skip contained low-risk edits and routine execution of a settled design unless new evidence reopens it; a plan-only request authorizes no execution.
license: MIT
---

# Task planning

A plan records the decisions an implementer must not make alone: what this change
will and will not do, which interfaces it commits to, how anyone will know it
worked, and when to stop patching and rethink. It is not a program listing, and
its length is evidence of nothing. Detail belongs where a decision is genuinely
open or a step is genuinely risky.

## When to plan, and when not to

Plan when the work crosses a component, service, data store or ownership
boundary; when it has a risky or irreversible effect; when two or more
dependencies decide the order; or when the requester's need is stated in terms
that do not yet identify behavior.

Do not plan a contained low-risk edit. The short path is: state the authorized
change, its file or symbol, and the one check that shows it worked; make the
change; run the check. A typo, a stale link, a renamed local variable and a
message-wording fix travel this path. A small diff that changes authorization,
persistence, a public interface or a release step is not small — size is not the
trigger.

If the requester asked only for a plan, planning is the deliverable. Writing the
plan file needs the same authorization as any other write; implementing,
installing dependencies, migrating data and publishing do not follow from a
planning request.

## The plan's minimum decisions

1. **Goal and non-goals.** The observable behavior that changes, who needs it,
   and what this change deliberately leaves alone. Non-goals are what keep scope
   from expanding one review round at a time.
2. **Scope contract.** Four explicit parts: what must be implemented, what must
   stay compatible, what is deliberately unsupported, and what existing
   alternative users are meant to use instead. Skipping the last two is how a
   change ends up half-supporting a case nobody wants. See
   [scope contract](references/scope-contract.md).
3. **Interfaces.** What the work consumes and produces — files, symbols, data
   shapes, events, endpoints, storage — with paths and concrete consumers.
   A consumer may already exist or be an explicitly requested, owned downstream
   work item in this plan. Hypothetical future reuse without that commitment
   belongs in non-goals, not an invented interface requirement.
4. **Acceptance.** For each work item, the done condition, the check that
   demonstrates it, and the expected observable result. "Tests pass" is not
   acceptance because it does not say which claim they support.

## Work items

Each item states what changes, what it consumes, what it produces, its
dependencies, and the check with its expected result. Order by dependency, not by
preference: an item starts when its inputs exist, not when a person is free.

Mark each item as independent or dependent:

- independent — no shared mutable state with another item, no pending interface
  decision, and a result that can be verified on its own;
- dependent — needs an interface, decision or artifact another item produces.

Only independent items may run in parallel. When several agents or owners will
execute the plan, mark exclusive files and interface ownership so the work can be
handed off without a conversation; the agent-team Skill covers that handoff when
it is available, and its absence changes nothing about the plan's contract. A
plan that requires a particular host's delegation tool is a plan that stopped
being portable.

## Decide, do not drift

Every plan of consequential scope carries open questions. Before implementation,
each is resolved into one of five outcomes — in your own words, not as required
vocabulary:

- **proceed** — the evidence supports the plan; the constraint that decided it is
  recorded, along with what would reverse it.
- **reduce** — part of the scope has no real consumer, no reachable path, or no
  operating owner, so it is cut and the cut is written down.
- **revise** — a decision or interface is wrong; change the plan before writing
  code against it.
- **stop** — the change should not be made as proposed: it duplicates an existing
  mechanism, breaks an invariant the plan cannot restore, or needs authority the
  requester has not given.
- **needs evidence** — the question cannot be answered from what is known; name
  the measurement, spike or question that would answer it and what stays blocked
  until then.

A proposal that has not been through this gate is a wish. The gate applies to new
fields, endpoints, events, modules, configuration and abstractions alike —
anything that must be owned and operated after it ships.

## Choose the smallest sufficient mechanism

Prefer satisfying the authorized scope without a new mechanism, then existing
project implementations, standard library/platform/framework capabilities,
installed dependencies, and small direct code before a new dependency or
abstraction. This is a decision preference, not a checklist to report. Do not
research alternatives or dispatch another agent for an obvious direct fix.

For new abstractions, dependencies, configuration, stored state or custom
protocols, name the current problem and why the direct option is insufficient.
Hypothetical reuse and unrequested flexibility are not requirements. Shared code
should represent a rule that its consumers must change together, not merely code
that looks alike. A single consumer can still justify a correctness boundary;
neither call-site counts nor line counts decide abstraction.

Apply the same burden to supporting work: test harnesses, reporters, migration
maps, proof gates and agent coordination. They serve delivery, not a separate
completeness goal. Prefer native tool results; propose a custom proof system
separately rather than making it an implicit refactor prerequisite. Compare the
places a normal change must update and the layers a failure must traverse.
Moving complexity into mocks, metadata or configuration is not simplification.
Do not add a scoring system, mandatory receipt or extra approval workflow for
this comparison; explain a real tradeoff briefly where it arises.

Existing tests, configurations and compatibility branches are evidence of past
decisions, not automatically permanent requirements. If carrying all of them
forward defeats the requested simplification, propose the narrower scope and its
concrete risks for owner approval. Do not silently remove required behavior or
coverage. Preserve security, data integrity, accessibility and explicit
compatibility guarantees; after an approved reduction, do not rebuild the
discarded obligations through a replacement framework.

For source attribution when maintaining these rules, see
[complexity sources](references/complexity-sources.md); it is not a task checklist.

## Complexity reset

When adopting an existing framework, read its current task, configuration and
lifecycle documentation and run the smallest authorized native workflow before
building adapters. Separate required constraints from optional hardening; do not
silently turn a stronger threat model into a prerequisite for a bounded pilot.
Prefer documented configuration for a demonstrated gap. If integration starts
owning transport, scheduling or cleanup that the framework already owns, reopen
the approach instead of adding another control layer. This does not authorize
unsafe execution or waive an explicit constraint.

The plan stops being the right artifact when the change stops having a shape.
Signals, in order of how early they appear:

- related review findings keep arriving on the same mechanism, and each fix adds
  a branch, condition or special case rather than removing one;
- patch branches accumulate for one requirement, and each new one needs the
  previous ones to be intelligible;
- a single behavior is now decided in several places, or the call path gains
  protocol hops, retries or fallbacks to reach the same outcome;
- the acceptance check needs to be weakened, narrowed or split to keep passing.

When these accumulate, stop patching and return to the scope contract and the
interface decision. The trigger is accumulation across related findings, not a
count of patches, rounds or lines: two findings that each add a special case to
the same decision point are already a signal, and a dozen independent fixes in
different modules are not. Record the reset, what triggered it, and which
decisions are reopened — silently rewriting the plan mid-execution defeats the
point of having one.

When a self-built supporting mechanism repeatedly becomes the repair target,
compare repairing it with reducing it or returning to native capabilities.
State the strongest current reason to retain it, then choose against the
authorized outcome and total maintenance burden, not sunk effort. Fixing the
mechanism correctly does not by itself establish that it remains necessary.

## Execution

When implementation is authorized, the plan is a commitment to be revised by
evidence, not defended:

- follow dependency order and start items only when their inputs exist;
- revise when a check or a reading contradicts an assumption, and say what
  changed and why;
- report deviations from the plan, not just completions;
- report each check with what it actually ran and observed, keeping blocked,
  not-run and unknown distinct from passed;
- a change existing is not acceptance evidence; the named check observing the
  expected result is.

Stop expanding when the agreed acceptance conditions are met; unrelated
improvements are follow-ups, not new prerequisites. If local milestones keep
completing without the remaining delivery work converging, reopen scope or
approach instead of reporting only that work is unblocked. A newly discovered
correctness or safety issue must still be surfaced, not hidden to declare done.

Escalate rather than decide alone when the open question is authorization,
credential handling, data loss, an interface owned elsewhere, or a choice that
changes what users can do. Unresolved questions stay named as unresolved; a plan
that quietly picks an answer is the plan a reviewer has to reject later.

## Keeping the plan usable

- Write it for the implementer who was not in the conversation, not for the
  archive. A file, a change description or the request thread can all carry it;
  the medium is not the contract.
- Prefer one line naming a constraint over a table of options when only one
  option meets the constraint. Record the deciding constraint, not a menu.
- Keep requirements, design, task list and progress as separate statements where
  they differ. One TODO list cannot explain why, define the contract and track
  status without doing all three badly.
- Delete superseded plan text. A plan with three historical versions of the same
decision teaches the reader to distrust all of it.
