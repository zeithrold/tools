# State ownership, persistence and recovery

Read when adding persisted state, moving its owner, changing a stored format or
altering async recovery. A visual-only change need not create a state machine.

## Trace the owner and every transition

For each state value, identify its canonical owner, readers, write path and
lifetime: request, component, navigation, browser session, account or durable
storage. A second cache or component copy must have a stated synchronization
rule; do not silently create another writer.

| Transition | Questions and evidence |
| --- | --- |
| Initialize | Is the server default deterministic? Can the first client tree use the same facts? |
| Restore | Is stored input validated as untrusted? What happens to missing, corrupt or unsupported values? |
| Update | Who is allowed to write? Can stale responses overwrite a newer selection or another account? |
| Migrate | Which old formats remain supported? Does failure preserve recoverable data rather than partially rewriting it? |
| Delete/reset | What is intentionally removed, what remains, and can an old request or notification resurrect it? |
| Fail/retry | Are user input and intent retained? Is a repeated mutation safe or must the server reconcile it first? |

Keep business authorization and durable mutation guarantees at their backend
boundary. UI state is not an authorization fact. A local clear is not necessarily
remote deletion or revocation; label the action truthfully.

## Worked example: changing a saved filter

A server-rendered list uses a deterministic default. On the client, validate a
saved filter and apply only supported fields. When the user changes the filter:

1. Advance the request identity and preserve the new selection.
2. Cancel the obsolete request when supported, and still reject a stale response
   if it arrives after cancellation or after an account change.
3. Show data belonging to the selected filter, or explicitly label retained old
   data while refreshing; never silently present old results as new ones.
4. On failure, retain the selection and provide a retry with clear scope.
5. On reset, clear only owned keys and invalidate pending restores/writes.

Exercise delayed out-of-order responses, restore failure and reset while a
request is pending. A unit test for a normal response alone does not cover these
transitions. Use the project's existing framework/state mechanism, not a new
global store introduced just to implement this checklist.
