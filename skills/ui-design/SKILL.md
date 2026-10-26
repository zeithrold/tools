---
name: ui-design
description: Use for web or Flutter UI design decisions about hierarchy, interaction states, accessibility, responsive and text-scaling behavior, typography, icon roles or visual review. Skip implementation architecture, executed gate evidence and mechanical edits with no design or experience effect.
license: MIT
---

# UI design

The principles here judge and shape an interface; they are not a component
library, a framework choice or a house style. The project's own design contract,
tokens, components, brand and copy are the authority, and every value stays
local to the product. Read that contract and the component guidance before
deciding anything. When a project has none,
[design contract extraction](references/design-contract.md) shows what to
record and from which evidence; it does not supply values on the project's
behalf.

Two Skills bound this work without being prerequisites. Implementation, state,
SSR, preferences and the shared shell belong to `frontend-engineering`;
executed checks and their evidence belong to `verification`. This Skill makes
and reviews design decisions, and runs no gate itself.

## Decide before implementing

A new flow or a meaningful visual or interaction change resolves a small
[design decision package](references/design-decisions.md) first: user outcome,
inherited system, visual concept, states and behavior, ownership, acceptance. A
copy fix, a token swap inside an existing role or a mechanical edit needs no new
document; state what changes and why in the task.

Do not replace an existing design system with a generic aesthetic, and do not
ban cards, gradients or motion categorically: judge whether a device supports
the brief, the hierarchy and a usable interaction. If a brief conflicts with an
existing pattern, name the conflict and get a decision rather than hiding an
override in one component.

## States, hierarchy and recovery

For each changed flow, name the user's goal, the primary and secondary actions,
the information that must stay visible, and the recovery path. Inspect loading,
empty, populated, validation error, service error, disabled and success states,
and preserve the user's input after a recoverable failure. Hierarchy comes from
entry point, reading order, primary action, task status and the weight given to
each region, not from enlarging everything.

[Review matrix](references/review-matrix.md) is the inventory to select from for
states, layout, interaction, semantics, recovery and appearance. Choose the
scenarios the changed flow needs; do not fabricate data to fill it.

## Accessibility and interaction

Prefer native semantic controls and explicit labels. Check keyboard operation,
visible focus, focus order, error association, announced state changes, and that
dialog, menu and sheet semantics carry an accessible name with a reachable,
localized dismissal. Give icon-only controls an accessible name and mark
decorative SVGs `aria-hidden="true"`. Keep meaningful content and reachable
actions at narrow viewports, large text, long translations, reduced motion and
real data extremes.

## Typography and icon roles

Typography uses Noto Sans or Noto Serif families with appropriate CJK variants,
plus Noto Color Emoji for all emoji, and covers at least English and
Chinese/Japanese/Korean glyphs. Prefer Noto Sans for UI and Noto Serif only where
content benefits, without loading both by default. Deliver fonts through the
Google Fonts API rather than self-hosting or vendoring binaries; document the
third-party requests, CSP and privacy implications, and load only the required
subsets and weights. Keep emoji consistently color instead of a monochrome-first
stack, verify rendered glyphs and multi-codepoint sequences with the intended
fonts, and record the choice in the local design contract.
Retain font licenses and required upstream notices with all applicable copies
and distribution paths; a family name or delivery URL does not replace them.

Use Lucide components for action, navigation and status iconography; Unicode
arrows, symbols and emoji are not substitutes for interface icons. Keep size,
stroke and alignment consistent per role, and preserve genuine prose,
mathematics, user content and intentional emoji. Audit consumer-owned icons
during source migrations without moving product navigation or content into a
shared runtime.

Role sizes, scales, spacing values and component variants live in the project
design contract; `frontend-engineering` covers how those roles reach components.
Do not shrink a whole form to supporting-text size to make a layout fit.

## Platform references

Read the one the current platform needs. The sections above apply to both, and
the stack a reference describes is the shared consumer's, not a requirement.

- Web: [web interaction and layout](references/web.md) for semantic HTML,
  responsive CSS, scrolling and overlay behavior, then
  [browser review](references/web-browser-review.md) for route and state
  coverage, keyboard and Axe evidence, built-runtime limits and retained
  captures.
- Flutter: [Flutter review](references/flutter.md) for widget semantics, text
  scaling, insets, focus and native-device evidence.

A Flutter project keeps its own widget and theme foundation and is not asked to
adopt React, Radix or web CSS; a web project likewise is not asked to imitate a
mobile design system.

## Review the rendered result

Inspect the rendered interface, not the intent in its source. A design document,
a build or a screenshot alone does not prove the interface works. Record what was
observed in a real runtime or device, what automated checks established, and what
still needs human visual judgment.

Preserve the project's required native lint, CSS, type, unit and browser gates
without weakening them, and read what each actually established. The optional
`verification` Skill covers execution methods; a design review itself is not a
test run. Captures are review evidence, not maintained baselines or design
approval, and belong in ignored artifact locations.

For requested visual refinement, named design benchmarks or award comparisons,
follow [visual design review](references/visual-design.md). Role budgets and the
visual concept remain consumer-owned.

Shared-source implementation and delivery contracts belong to the optional
`frontend-engineering` Skill or the project's own component documentation.
No sibling Skill is required for the design decisions advertised here.
