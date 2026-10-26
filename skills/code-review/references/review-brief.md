# Review brief

The brief is the reviewer's whole world. Everything the review can establish is
bounded by what the brief supplies, so a thin brief produces a thin review and no
amount of reviewer skill compensates.

## What the brief must contain

1. **Decision.** What the result is used for: accept a change, choose between
   proposals, confirm a risk is handled, give a human owner what they need to
   decide. A brief without a decision invites an encyclopedia.
2. **Pin.** Commit, range or artifact identifier, plus the paths in scope and the
   paths deliberately out of scope. If the intended review is "the diff only",
   say so and name what a diff-only review cannot establish.
3. **Intent.** The observable behavior the change should produce and who needs
   it, with its source (request, issue, plan, contract, previous behavior).
   Quote the requirement rather than paraphrasing it into your own confidence.
4. **Applicable instructions.** Project rules, design or interface contracts,
   versioning policy, review conventions that constrain acceptance. Name the ones
   that bind this change; do not hand over the whole rule set.
5. **Ownership.** Who owns the touched paths, and who may change a shared
   interface. Findings about code outside the owned paths route differently.
6. **Execution evidence.** Each command or flow that ran, at which pin and in
   which environment, what it observed, and what it did not touch. Include failing
   and absent checks; a brief that hides them manufactures a false clean review.
7. **Known gaps and unknowns.** Untested environments, unavailable services,
   unverified claims, decisions still open. These are inputs, not conclusions.
8. **Limits.** Read-only permission, round budget, deadline, and who decides the
   disposition.

## What the brief must not contain

- Your findings, your preferred verdict, or a request to confirm a decision.
  Independence is lost the moment the reviewer knows the answer you want.
- Your reasoning chain. A reviewer that inherits your framing inherits your blind
  spots; give the requirement and the evidence, not the path that produced them.
- A diff summary standing in for the code. Reviewers read the pinned source and
  its callers; summaries conceal exactly the context that makes a finding
  correct or wrong.
- Claims as facts. "Tests pass", "backwards compatible" and "just a refactor" are
  hypotheses until they come with the observation behind them.

## Example brief (synthetic)

```
Decision: accept or reject merging the token-refresh move into auth-shared
before the mobile client update ships next week.

Pin: branch auth/shared-refresh at 4f2c1ab, paths auth-shared/**, web/session/**,
and migrations/0017_refresh_tokens.sql. Out of scope: admin/** (unchanged, known
legacy token logging).

Intent: web and mobile clients refresh through one service; a refresh writes a
new access/refresh pair atomically; logout revokes the pair everywhere. Source:
plan-2026-04/refresh-service.md, plus the mobile client owner's requirement that
the old endpoint keep working for one release.

Instructions: docs/api-compat.md (additive changes only), AGENTS.md rule that
credential handling needs an independent high-risk review.

Ownership: auth-shared owns the service and migration; web/session owns its
caller. Interface changes need the platform owner.

Evidence: auth-shared unit tests green at 4f2c1ab (14 tests, they mock the store);
web integration suite not run (needs the staging credential service, unavailable);
no test runs against a real database; migration applied to a scratch schema once,
not re-run.

Gaps: mobile client still on the old endpoint by design for one release.
Concurrency of two simultaneous refreshes: unverified.

Limits: read-only, two rounds, decision belongs to the platform owner.
```

Notice what the brief does not say: it never asserts the change is safe, never
asks for a verdict anyone can guess, and states the failing or missing evidence
plainly enough that a reviewer can start from it.

## Briefs for in-context review

When the same agent reviews its own adjacent work, or a reviewer has no separate
context available, write the brief anyway. The exercise exposes intent gaps and
unverified claims before the review starts, and the resulting review must still
be reported as self-review rather than independent assessment.
