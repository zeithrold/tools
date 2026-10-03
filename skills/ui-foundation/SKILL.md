---
name: ui-foundation
description: Apply cross-frontend principles for information hierarchy, accessible interaction, responsive content, and visual evidence.
---

# UI foundation

Read the project's own design contract and component guidance first. Use its brand, tokens, product content, and component foundation.

For each changed flow, review the user's goal, primary and secondary actions, information that must remain visible, and the recovery path. Inspect loading, empty, populated, error, disabled, and success states. Check labels, focus order, keyboard use, accessible names, and error association. At narrow viewports, large text, long translations, and reduced motion, preserve important content and reachable actions.

Use the project's stack-specific UI skill for implementation details. Record what was observed in real runtime or device captures, what automated checks established, and what still needs human visual judgment. A design document, build, or screenshot alone does not prove the interface works.

See [review matrix](references/review-matrix.md) for a concise state and environment inventory. Keep colors, typography, spacing values, and business-specific component patterns in the project design contract.

Use [design contract extraction](references/design-contract.md) when consolidating principles from existing projects. Trace semantic token roles, component variants, local typography and preferences to source evidence; preserve project-specific values and business choices.

## Generic shared shell

`@ztd-me/frontend` provides reusable chrome, validated non-sensitive appearance/UI locale and explicit persistence mechanisms. Its defaults are Neutral + System and six palettes. Consumers supply their own brand, footer/repository/contact content, cookie name/domain/Secure choice, optional storage notification key and business slots. Tools must not select policy through project names or hostnames, enforce a repository-host allowlist, or carry legacy storage mappings. Remove rejected legacy mappings entirely rather than moving them into consumers; old business/auth storage stays untouched. Shared domain cookies are untrusted UI input. Consumers own development/preview isolation, deployment, authentication, account state and business navigation. Generic shell guidance grants no sync ownership over project components or production files. Preserve native strict gates and upstream licenses. Final integration requires a promoted exact registry version and fresh-consumer verification; staging or local packed evidence alone does not establish public installation.
