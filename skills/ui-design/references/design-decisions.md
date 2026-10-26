# Make the design decision before implementing it

Read this for a new flow or a meaningful visual/interaction change, not every
copy edit. An existing project design system is the starting point, not something
to replace with a generic aesthetic. For a small change, a few sentences in the
task are enough; do not require new documents or particular filenames.

## Minimum decision package

Resolve these questions before choosing components or CSS:

| Decision | What must be explicit |
| --- | --- |
| User outcome | Primary task, supported users, information that must remain visible |
| Inheritance | Existing tokens, component variants and interaction conventions to retain |
| Visual concept | Hierarchy, layout and typography roles; the reason for a departure from existing patterns |
| State and behavior | Loading, empty, populated, disabled, success, failure and recovery; focus and navigation |
| Ownership | Design system owns appearance/variants; feature owns composition/layout; state has one writer |
| Acceptance | Real routes and states, narrow/large-text/long-label conditions, interaction checks and visual judgment |

Use named project tokens and font roles, not values borrowed from another
product. If the brief conflicts with an existing pattern, identify the conflict
and obtain a decision rather than hiding an override in a consumer `className`.
Do not ban cards, gradients or other devices categorically; judge whether they
support the brief, hierarchy and usable interaction.

Visual and experience responsibilities may live in one existing design contract
or separate documents. Preserve both responsibilities; do not force a
`DESIGN.md`/`EXPERIENCE.md` file pair merely to match a workflow.

## Worked example: an import preview

An operator needs to distinguish accepted rows from rejected rows before
confirming an import. The existing system already has table, alert and dialog
variants. Keep those variants and the existing type scale.

- The feature owns preview layout and filter placement, not new table colors.
- Make rejected-row count and the consequence of confirmation visible.
- Define parsing/loading, no rows, partial rejection, service error and
  successful confirmation separately; retain the selected file after a
  recoverable failure.
- If the preview becomes a dialog, decide initial focus, return focus,
  keyboard dismissal and how large text/narrow layouts keep confirmation
  reachable.
- Acceptance includes a real preview/confirm/retry journey and visual review
  of the meaningful states. A screenshot of a populated table alone is not
  acceptance.

This package exposes unresolved product choices early. It is not permission to
invent sample business content, change deployment policy or introduce a new
design system.
