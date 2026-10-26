# Activation and rollback

Read this before promoting a candidate, changing the active pointer, revoking a
source or deleting guidance. Activation is the step where a proposal becomes the
behavior of future tasks; it deserves the same care as a schema migration.

## Candidate and active are different objects

- A candidate is immutable: identified content plus its envelope. Editing it in
  place creates a second object with the old identity.
- Active state is a pointer to one candidate, plus the approval that authorized
  it and the evidence it relies on.
- The archive of candidates, rejected proposals and expired rules is separate
  from the active pointer, retained for audit and recovery without influencing
  runtime.

A promotion records the previous active candidate so restoration is a pointer
operation, not a reconstruction from memory.

## Approval binds to exact content

Approval names the canonical digest, base revision, target, validation contract
and evidence. It authorizes that content for that target, not a future edit and
not "self-improvement" in general.

- Any content, base or target change invalidates the approval and the results
  that depended on it; the new revision is re-reviewed.
- Verify the digest immediately before applying. Content that changed between
  review and activation is not approved content; a read-then-approve window that
  is not checked is a time-of-check/time-of-use gap.
- Keep the approval record append-only with who approved, when, what digest and
  what evidence. A mutable approval record cannot be audited.
- Do not let the candidate's process write its own approval or the active
  pointer.

## Reversible rollout

1. Verify the rollback path before activation: the previous candidate still
   exists, is intact, and can be made active without depending on the failing
   change.
2. Activate in a limited scope first when the mechanism supports it: one project,
   one task class, shadow evaluation, or a small sample. Define the signal that
   ends the trial and the one that triggers rollback before starting.
3. Observe the stated signals and keep the failure evidence; a trial that only
   records successes has not tested rollback.
4. Widen only with a new decision, not automatically because no alarm fired in a
   short window.

If the previous state cannot be restored — the old text was overwritten, the
store was mutated in place, the change was applied non-atomically — the change is
not reversible. Refuse it rather than calling rollback best effort.

## Atomic replacement and integrity

- Write the new content beside the active one, verify it, then switch the pointer
  or rename atomically. Never edit active content in place.
- Re-read what you wrote and compare its digest with the approved one; a partial
  or corrupted write must not become active.
- Guard concurrent activations with a lock or conditional write on the active
  pointer, and reject a pointer change based on a stale pre-state.
- Record the preimage (the previous content and digest) in an append-only ledger
  before switching, so recovery does not depend on a working tree that may have
  moved.

## Revocation and deletion

Revoking a source — a license problem, a leaked credential, a retracted
observation — removes everything derived from it: active guidance, archived
candidates, index entries, merged text and cached loads. A revocation that leaves
a derived copy is incomplete; the copy re-enters future retrieval.

Deleting guidance is different from deactivating it. Prefer deactivation with a
recorded reason, and delete only with an owner decision and a preserved audit
trail. Low usage, measurement gaps or an old timestamp are not deletion reasons:
counts are incomplete (shell reads, caches, uninstrumented hosts), and a rarely
used rule can be a critical safety gate. Unknown usage is reported as unknown.

Merging or shortening guidance must preserve triggers, exclusions and version
assumptions from both sides; otherwise the rewritten rule silently changes scope.

## Runtime enforcement

Enforcement is a property of the host, not of the instruction text:

| Constraint | Enforced by | Not enforced by |
| --- | --- | --- |
| allowed write paths | filesystem permissions or an owned root | "only edit these files" |
| network access | sandbox or egress policy | "do not fetch" |
| credentials | absent from the environment, scoped, short-lived | "do not print them" |
| token, request, time, concurrency, money budget | a ledger the host checks before and after | "stay within budget" |
| cancellation | process-tree termination and child tracking | "stop when cancelled" |
| evaluator integrity | separate write authority and content hashes | "do not modify tests" |

Reserve budget before a call and reconcile after; a candidate cannot raise its
own limit, and a long-lived credential in the candidate's environment is a
grant, whatever the prompt says. Permission to publish, merge or deploy stays with
the owner and is never implied by a successful evaluation.

## Failure scenarios

Synthetic examples for the activation stage:

- **Approved, then edited.** Approval was recorded for digest A; the candidate is
  regenerated with a wording change before activation. Applying without a digest
  check activates unreviewed text. The fix is a pre-apply digest comparison tied
  to the approval, and re-review on mismatch.
- **Rollback that cannot restore.** The new guidance replaced the old file in
  place, and the preimage was never saved. The failing rule cannot be reverted,
  so the team is left editing under pressure. The fix is write-aside, verified
  replace and a ledgered preimage before switching.
- **Revocation with residue.** A source is retracted and the active rule is
  removed, but an archived candidate and a cached index entry remain. Retrieval
  serves the retracted claim again. Revocation must enumerate derived artifacts
  through lineage.
- **Automatic archive of a quiet gate.** A rarely triggered safety rule is
  archived for low usage; the next incident has no gate. Low or unknown usage is
  a review signal, not a deletion policy.
- **Budget by prose.** A loop is told to spend at most N tokens. The candidate
  calls a large model repeatedly and exhausts the account. The limit existed only
  in the prompt; the host never metered it.
- **Active pointer without base check.** Two projects advance the same pointer
  from different pre-states; the second overwrites the first's approval. A
  conditional write on the pointer plus a recorded pre-state prevents it.
