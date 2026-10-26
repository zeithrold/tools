---
name: agent-team
description: Use when coordinating multiple delegated units or owners, assigning shared-file ownership, or integrating their results. Covers handoff contracts, isolation levels, completion events and integration. Skip single-owner sequential work and a standalone independent review, which belongs to code-review; size alone does not justify a team.
license: MIT
---

# Agent team

A team costs coordination and buys parallelism or independence. Both are real:
two owners can work at once on work that does not interact, and a context that
never saw the author's reasoning can disagree with it. Everything else about
delegation is overhead.

So: delegate when the work is genuinely independent, or when the point is a
judgement uncontaminated by the person who did the work. Do not split a task
because it is large, because a team is available, or because parallelism looks
thorough. One owner with a clear plan is usually faster than three owners with
the same plan.

This Skill describes a contract, not a mechanism. Whatever the host provides —
subagents, separate sessions, worktrees, hand-run copies — must carry a brief in
and a result out. Where a host provides nothing, the contract still applies to a
brief handed to another person.

## Independent or dependent

- **Independent:** touches disjoint files, depends on no undecided interface,
  and produces a result verifiable on its own. Safe to run in parallel.
- **Dependent:** needs an interface, decision or artifact another unit produces.
  Keep it with the owner of that decision, or sequence it after the producer
  finishes.

Dependence through an interface is the one people miss: two workers editing
different files can still be dependent, because both consume a decision nobody
has made yet. Freeze interfaces before parallel work starts, and re-freeze them
by agreement — never by one worker shipping a change and letting the others find
it.

## Three isolations, and they are independent

1. **Context isolation** — a fresh conversation that does not inherit the
   parent's reasoning. It removes bias and reminder; it protects nothing else.
2. **Workspace isolation** — a separate checkout or worktree. This is what keeps
   concurrent edits from colliding. A fresh context is not a fresh worktree, and
   the most common delegation failure is a worker with a new context editing the
   same files as its parent.
3. **Tool permission isolation** — what the worker can actually do: read, write,
   run, publish, spend. A brief that says "read-only" does not remove write
   access, and saying so in prose is not enforcement.

Assume none of the three implies another, and state which ones actually hold.
[Isolation and ownership](references/isolation.md) covers the mechanics, the
shared-file problem and what happens when the host cannot enforce a boundary.

## The handoff contract

Every delegated unit states:

- **goal** — the observable result, not the activity;
- **non-goals** — what it must not do, including adjacent work it might notice;
- **exclusive files** — the paths the worker may edit, and the paths it must not;
- **interfaces** — what it consumes and produces, with paths, including the
  interface it may not change without asking;
- **context** — the minimum it needs to decide: intent, constraints, prior
  decisions and their reasons, relevant conventions. Not the parent's whole
  history, and not the conclusion the parent wants;
- **permission** — read-only or write, which commands may run, what needs
  approval, what stays untouchable (credentials, remote state, publication);
- **test and output** — the exact check with its expected result, plus the report
  shape: what changed, what ran, what it observed, what stayed unverified.

A brief missing any of these produces work that looks finished and cannot be
trusted. [Handoff](references/handoff.md) has each field in detail plus a full
worked example.

## Shared files and interfaces get one owner

Parallel work fails predictably at the shared file. The rule is one owner, not
mutual respect:

- A shared file — build manifest, dependency file, route table, schema, shared
  helper, lockfile, changelog — has exactly one owner during the round. Others
  request the change, describe why, and wait; or the parent applies it after the
  round.
- A shared interface has one owner, and only that owner changes it. A consumer
  that needs a different shape asks rather than shipping a parallel variant.
- An exclusive-files declaration coordinates people; it does not lock the
  filesystem. Two isolated worktrees can both edit the same path, and integration
  is where that becomes visible.

## Completion is an event, not a poll

Workers signal completion — result, changed paths, checks run, open questions —
when they finish. The coordinator waits for that signal and does other work in
the meantime; repeatedly checking whether a worker is done burns the budget the
team was supposed to buy and makes ordering decisions from partial state.

If the host offers no completion signal, the fallback is an explicit checkpoint
with a stated inspection time, not a loop. And a worker that cannot finish says
so with its partial state and its blocker, instead of going quiet.

## The parent collects, verifies, integrates

When results arrive, the parent owns three steps and can skip none:

1. **Collect** — every unit's report, against the brief it was given, including
   the ones that failed or came back partial.
2. **Verify** — re-run or inspect the claim rather than trusting the summary. A
   worker's "tests pass" is a claim about its environment and its revision. Where
   the claim is consequential, check it at the integrated revision.
3. **Integrate** — merge the verified work deliberately, order it by dependency,
   resolve conflicts against the interface owner's decision, and re-run the
   checks that span units.

Blind merging is the failure this section exists to prevent: a set of green unit
results can integrate into a red system. [Integration](references/integration.md)
covers ordering, conflicts, rollback and a worked example.

## Lifecycle, recursion, budget

- **Stop conditions** are declared with the handoff: what makes the worker stop
  and report instead of improvising (missing environment, a needed interface
  change, a scope question, repeated failure).
- **Cancellation** stops the work, not the consequences: whatever the worker
  already wrote stays until someone removes it. The parent reconciles partial
  writes, reports where they are, and does not assume a cancelled unit left the
  tree unchanged.
- **Recursion** stops at the brief: a worker does not delegate further unless the
  brief grants it, and even then with a stated depth. Unbounded sub-delegation
  makes cost, ownership and review status unknowable.
- **Budgets** — number of units, rounds, retries and wall-clock or token cost —
  are set before dispatch, because a budget agreed after the failure is a
  negotiation, not a limit.

Most of this is prose in a brief, and prose is not enforcement. Separate what the
host enforces (a worktree really is separate; a permission profile really is
read-only) from what only the prompts ask for (not editing a shared file, not
recursing, staying in scope). For anything load-bearing, prefer the enforced
mechanism, verify the artifact rather than the promise, and record which
boundaries were assumed rather than guaranteed. A boundary reported as enforced
when it was only requested is a false assurance to whoever reads the report.

## Team size

Start with one worker per independent unit and stop there. A second reader or
worker is justified by a second failure mode or a second independent unit — not
by importance, and not by wanting the result faster. When review is the reason
for the second agent, the code-review Skill defines the round; this Skill only
defines how the brief and the result travel.

When a task needs no second owner, say so and use no team. Planning and review
both work standalone; neither requires a multi-agent setup.
