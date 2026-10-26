# Evidence reuse and invalidation

Reusing a sound result saves work. Presenting an old result as a new run is
dishonest evidence. Reuse is valid only inside the scope the evidence actually
covered. Examples are synthetic.

## Record the scope

For every retained result, name:

| Field | Content |
| --- | --- |
| Claim | The one sentence the result supports |
| Evidence | Artifact, report or run identifier |
| Revision | Build identity of the code and of dependencies |
| Recipe | Command, arguments and order |
| Environment | Runtime, OS, services, credentials, data class |
| Result | Status, output path and observation |
| Reuse | Who relied on it, when, and what was rechecked |

## Conditions that invalidate it

- A diff touching the exercised path, its callers or the contract it proves.
- A dependency, lockfile or toolchain change.
- A runtime, platform, service or configuration change.
- A recipe, fixture or data-version change.
- An expired result, or a source whose state can no longer be established.

A change outside those conditions may leave the evidence valid, but the report
has to say why the intermediate diff cannot affect the proven contract. "It
still works" is not that statement. When scope is uncertain, rerun the narrow
check or report the evidence as `unknown`.

## Reused is not new

Mark each retained result as reused, with its original revision and date, and
list what was newly checked. A report that lists a previous pass among today's
checks without that label misstates what ran.

### Synthetic ledger

| Claim | Evidence | Scope | Change since | Status |
| --- | --- | --- | --- | --- |
| Unit suite green | run at `abc123`, 2025-03-04 | package `quote`, toolchain from `go.mod` | docs-only edit | reused; docs are not part of the proven contract |
| Browser checkout works | trace at `abc123` | chromium 121, built app | lockfile changed | invalid; browser suite not rerun |
| Migration preserves rows | report at `abc123` | fixture v3 | fixture v4 | invalid; new run required |

Do not reuse a pass across a changed test command, a changed dependency, a
different platform where the behavior is platform-specific, or a different
environment whose configuration is part of the claim. Partial reuse is partial:
name which sub-claims are new and which are carried over.
