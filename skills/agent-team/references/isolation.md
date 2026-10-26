# Isolation, ownership and lifecycle

Delegation goes wrong in a small number of predictable places. This file covers
those: what isolation actually buys, how shared files fail, and what a parent owes
a worker that stops early.

## Context isolation

A fresh context receives the brief and nothing else. It buys:

- **independent judgement** — the worker does not inherit the author's framing,
  so its disagreement is informative rather than a restatement;
- **focus** — no accumulated transcript competing for attention;
- **a second chance at a wrong premise** — a new reader can notice the plan is
  wrong, which the author's context has already accepted.

It buys nothing about files. Two contexts, one working tree, the same concurrent
edit problem as two people sharing a directory.

## Workspace isolation

Physical separation of the files: a separate worktree, a separate checkout, a
copy, or an environment specific to that unit.

- Separate workspace + separate context: the clean case. Work is attributable,
  collisions are impossible until integration, and integration is where conflict
  is handled once instead of continuously.
- Shared workspace + separate context: the dangerous case. Both workers write the
  same filesystem. One sees the other's half-finished edits, a test run reflects
  someone else's change, and a "passing" result may describe code the worker did
  not write. Exclusive-file declarations are the only coordination left, and they
  are honor-based.
- Shared workspace, shared context: not delegation. It is one worker with more
  turns, and it should be called that.

If the workspace is shared, prefer serializing any unit whose paths overlap, or
reduce to a single owner. Do not rely on the declared file boundary for
correctness; verify the paths at integration.

## Tool permission isolation

What the unit can do, as enforced by the host: read, write, run commands, reach
the network, use credentials, push, publish, deploy, spend.

- Declared permission and enforced permission are different facts. A prompt that
  says "you are read-only" does not remove write tools; a reviewer persona
  described as read-only frequently ships with an editor. Check what the worker
  actually has, not what the description says.
- When the host cannot enforce a limit, the honest options are: serialize the
  work, reduce its authority by not giving it the relevant input (a token, a
  remote, a production config), or run it and report the boundary as unenforced.
- Permission is also about inputs. A worker told not to publish gains nothing from
  holding a publication credential; a worker told not to touch production gains
  nothing from a production connection string in its environment.
- A worker's report of what it did is a claim. If a boundary is load-bearing for
  safety or correctness, look for the artifact: the diff, the remote state, the
  log. Cheap checks (did any file outside the list change; is the branch pushed)
  are cheap because they are usually enough.

## Shared files and interfaces

One owner per shared file for the duration of the round. The protocol:

1. The parent assigns ownership when dispatching, not when the collision appears.
2. A non-owner that needs a change there sends the request with the reason and the
   intended edit; it does not edit and does not wait silently.
3. The owner applies it, or answers with the reason it will not, and the round
   continues. A serialized two-minute change is cheaper than a conflict found at
   integration.
4. The parent records shared-file changes as separate items when collecting, so
   they are reviewed once by their owner and not argued per worker.

Typical shared files: build and dependency manifests, lockfiles, schema and
migration directories, route and registration tables, shared constants and
helpers, changelogs, generated indexes, CI configuration.

Interfaces follow the same rule with higher stakes: only the interface owner
changes a published shape. A consumer with a different need asks; shipping a
parallel variant is how two contracts end up live at once.

## Lifecycle

A unit moves through: dispatched, running, completed, failed, cancelled. The
parent's obligation differs by state, and none of them is "assume the tree is
as I left it".

- **completed** — collect, verify, integrate. Even a clean completion may have
  touched files outside its scope.
- **failed** — read the failure before redispatching. A retry with the same brief
  and the same missing environment fails again; a redispatch into the same worktree
  may land on the failed attempt's partial writes.
- **cancelled or stopped** — the work stopped, the writes did not. Reconcile the
  partial state yourself: report where it is, remove or keep it deliberately, and
  tell the next owner. Cancelling a unit that shares a workspace does not
  un-write its files.
- **abandoned** — no result and no signal is the worst case. Treat it as
  unknown state: inspect the workspace before assuming either outcome.

Parent cancellation must propagate to descendants, subprocesses and background
calls. Confirm the host actually supports and performed that propagation; a
cancel request or prompt instruction alone is not proof that execution stopped.
If work can outlive its parent, name that limitation and reconcile it before a
replacement owner writes into the same state.

Re-dispatch decisions: reuse a worktree only if you inspected it; a fresh one is
safer and loses nothing but setup time. Do not cancel and immediately respawn into
the same files without reconciling, which produces two authors for one edit.

## Recursion, fan-out and budget

- Depth: a worker does not delegate unless the brief grants it, and the grant
  states a depth. A default of zero is easier to defend than a depth nobody
  remembers.
- Fan-out: cap the number of workers created per parent and per round. Unbounded
  fan-out makes cost, ownership and review status unknowable at exactly the moment
  the work is largest.
- Budget: units, rounds, retries, cost or duration are decided before dispatch.
  When a budget is exhausted, the correct output is a partial report plus what
  remains — not an unbudgeted continuation.
- Loop guard: if two rounds have not improved a result, stop and report. A third
  identical attempt is a cost with a known outcome.

## Enforced versus asked

| Boundary | Enforced example | Merely asked |
| --- | --- | --- |
| Workspace | separate worktree or checkout | "don't edit my files" |
| Role | a permission profile without write access | "you are a reviewer" |
| Scope | dispatch with a subset of the repository readable | "stay in services/webhooks" |
| Recursion | a runtime that refuses to spawn at depth | "don't delegate further" |
| Budget | a host cap on units or runtime | a sentence in the brief |
| Publication | no push target, no credential | "do not merge" |

Read the column that actually applies and say so in the report. A boundary
reported as enforced when it was requested is a false assurance, and false
assurance is worse than a stated gap because nobody re-checks it.
