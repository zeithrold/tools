# Scope contract

The scope contract answers "what is this change responsible for" before any file
is touched. It has four parts, and each part is a commitment that later work is
measured against. A contract missing one of them is incomplete, and the missing
part is exactly where scope drift enters.

## 1. Must implement

The behavior this change is accountable for, as observable outcomes rather than
internal activity. Each entry should be checkable after the change ships without
reading the code: a response, a stored state, an error a caller can distinguish,
a migration outcome.

Write down the requirement's source — the request, issue, contract or observed
defect. A requirement with no source is either derived from something unstated or
invented; both are worth catching now.

Bad: "refactor auth into a shared service".
Usable: "web and mobile refresh through one service; a refresh writes a new
access/refresh pair atomically; logout revokes the pair for every client".

## 2. Must stay compatible

What existing callers, stored data, deployed clients, operators and external
contracts already depend on — named specifically, with the versioning rule that
applies (additive only, versioned endpoint, staged rollout, a deprecation window).

This part is where "it's just internal" claims fail. Enumerate consumers from the
code and from deployment reality: a consumer compiled from the same source tree
but shipped separately is an external consumer. State what happens to data and
clients that exist at deploy time, not only to the next fresh install.

The compatibility promise needs an owner and an end condition: until when, and
what removes it.

## 3. Deliberately unsupported

What this change will not handle, and why that is acceptable. This part turns a
limitation into a decision; left unwritten, the same limitation becomes a bug
report later.

Each entry needs three things or it is not a decision:

- the case (a shape of input, client, environment, scale or migration state);
- the reason it is out of scope (no real user, disproportionate cost, forbidden
  by a constraint, replaced by something else);
- what a user in that case sees — an explicit rejection, a documented
  limitation, or a failure they can diagnose. A confusing error is not a scope
  decision.

## 4. Existing alternative

Investigate and record the supported alternative, if one exists: another command,
endpoint, flag, documented workflow or neighbouring system. An intentional
security, compatibility, cost or product exclusion may have no replacement.
State "no supported alternative", its reason and user-visible behavior, and
obtain the owner's decision when consequential. Do not manufacture an alternative
or silently turn an unintended gap into an approved exclusion.

This part also covers duplication: if an equivalent mechanism already exists, the
contract should say why this change does not extend it. "New module" is an
answer to "where", not to "why not the existing one".

## Putting it together (synthetic)

```
Must implement
  - refresh issues a new access/refresh pair through one service for web and
    mobile
  - the pair is written as a unit; local write failure preserves the old record,
    but remote rotation may require reauthentication rather than token rollback
  - logout revokes every refresh path for that session

Must stay compatible
  - old /session/refresh keeps working for one release; mobile client ships
    separately and cannot be updated in the same window
  - tokens issued before deploy stay valid until their natural expiry
  - admin tooling reads the session table; its schema is unchanged

Deliberately unsupported
  - third-party OAuth refresh: no consumer, and the platform owner rejected the
    scope for this quarter
  - >1 refresh per second per session: rejected with a 429 and a retry hint so
    the client can back off, not silently dropped

Existing alternative
  - third-party integrators continue to use the long-lived API key flow,
    documented in docs/api-keys.md; it is not deprecated by this change
```

That contract is short, and each line can be tested later: it names the
compatibility window, the data that must survive deploy, the rejection behavior
and the replacement path. Nothing in it is a task, an estimate or a design, which
is why it survives contact with implementation.

## How the contract drives the rest of the plan

- Missing part 2 turns into a surprise at deploy; missing part 3 turns into an
  argument in review; missing part 4 turns into "where do I go now" support load.
- Acceptance items trace back to part 1 and part 2. A test that supports neither
  is a check, not acceptance.
- Work items are bounded by the contract: a task that serves none of the four
  parts is scope creep with a schedule of its own.
- The contract is the first thing to reopen on a complexity reset. Accumulating
  findings usually mean one of the four parts was never decided, and the code is
  absorbing that gap as special cases.
