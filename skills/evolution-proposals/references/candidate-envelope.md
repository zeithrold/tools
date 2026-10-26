# Candidate envelope

Read this when creating or reviewing an evolution candidate. The envelope is the
candidate's claim about itself: where it came from, where it applies, what it
assumes and when it stops being valid. Field names below describe required
content; the host implementation owns the concrete schema.

## Source and evidence

Reference the immutable original, not a summary of it: evidence identifiers,
commit hashes, artifact paths, transcript or issue references within their
access scope. A summary loses the conditions that made the experience true and
is not re-checkable.

- Separate observed facts ("the check exited 1 at revision R") from the
  proposer's interpretation ("the order was wrong").
- Never include secrets, credentials, private source or personal data; a hash is
  not a privacy check.
- Keep the evidence readable by whoever reviews the candidate; a reference to a
  lost artifact is not evidence.

## Generator, scope and trust

Record who produced the candidate, which project or tenant it observed, and its
trust level with the full derivation chain. A candidate derived from an
untrusted source stays untrusted after summarization, embedding, merging or
moving. Derivation includes machine edits and paraphrases, not only explicit
"derived from" links.

Scope is what the candidate may affect: one project, one task family, one Skill.
A candidate observed under one project's conventions is not automatically
general; widening scope is a new decision, not a formatting choice.

## Applicability and exclusions

Applicability names observable conditions: the task class, the tools present, the
failure shape, the environment. Exclusions name where it must not apply —
production credentials, untrusted input, a project whose contract differs, a task
with a hard deadline.

A candidate with no exclusions and no bounded applicability cannot be evaluated:
any result can be explained afterwards. Write the exclusion before seeing the
result, and keep it through revisions.

## Tool versions and environment assumptions

The mechanism may depend on a command flag, an API version, a language feature or
a framework behavior. Record the versions and the environment class, and state
what happens when they drift. An environment fact ("this checkout needs X") is
not procedural guidance; it belongs to the environment with an expiry.

## Lineage and supersession

Record the parent candidate, the rule it modifies and any candidate it replaces.
Lineage supports three things: revoking derived work, avoiding two active rules
that contradict, and reconstructing why the active text looks the way it does.

When a new candidate supersedes an existing rule, deactivate the old one in the
same change; leaving both active produces the ambiguity the proposal was meant to
resolve.

## License and provenance

Adapted material needs a known license and the source revision and file. Unknown
redistribution rights block copying; rewriting an upstream text does not remove
its obligations. Preserve required notices, and record the destination and the
changes made.

## Novelty and the alternative of correction

Before proposing an addition, search existing guidance for the mechanism:

1. Is there already a rule that covers this decision? If yes and it is correct,
   the candidate is duplicate guidance.
2. Is there a rule that gave the wrong order, missed a trigger or stated an
   exception vaguely? Prefer editing that rule; a counter-rule leaves both.
3. Is the gap in the mechanism or in the evidence discipline? A missing check is
   often a verification problem, not a new behavioral rule.
4. Does the candidate change a decision on a future task, or only phrasing? If
   only phrasing, it is not an evolution candidate.

Record which rules were searched and how the candidate differs. "New" is a claim
that must survive the search.

## Illustrative content

The block shows the content a review needs. It is an example, not a schema, and
not a record from any real system.

```text
claim         read the migration ledger before changing a stored schema
source        review note r-77; failure log at revision 3f2a; rule DDL-4 modified
generator     assistant, project P, untrusted session
derivation    summary of r-77 -> candidate (summary does not raise trust)
applicability production schema change touching a shared table
exclusions    synthetic fixtures; local-only stores; emergency rollback
assumptions   migration tool v2 ledger; single writer per migration
expiry        re-verify when the migration tool major version changes
license       original text, no upstream material
novelty       DDL-4 says "run migrations" but not when; this adds the ordering
lineage       modifies DDL-4; no other candidate supersedes it
```

## Rejection reasons

A candidate is rejected or returned, not activated, when: evidence is missing or
unreadable; trust is laundered through a summary; scope is unbounded; no exclusion
or expiry exists; an existing rule already covers it; the license is unknown; it
encodes a user preference as a global rule; it requires evaluator or budget
changes; or its stated benefit cannot be tested. Report which one applies, so the
proposer can fix the envelope rather than resubmit the same text.
