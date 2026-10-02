---
name: frontend-engineering
description: Use when implementing or reviewing frontend architecture, component variants, data boundaries, localization, preferences or SSR hydration while preserving native framework and product contracts.
---

# Frontend engineering

Read AGENTS.md, the project design contract, scripts, compiler config and framework entry points. Use ui-foundation/ui-web for design and frontend-verification for execution evidence. Select Skills explicitly in zt.json; sync does not resolve their dependencies.

1. Map the feature from route/server boundary through validation, state owner and components. Follow [boundaries](references/boundaries.md); keep business/deployment assumptions local.
2. Reuse semantic tokens, local fonts and component variants. Define missing semantic roles, including destructive actions, in the project contract without copying another product's values.
3. Validate unknown runtime input using the existing schema mechanism. Preserve retry and input; do not expand a boundary improvement into an unrelated client/server rewrite.
4. Normalize supported preferences and render deterministic SSR defaults. Apply [preferences and localization](references/preferences-i18n.md) to hydration, persistence, errors and dialogs.
5. Preserve framework exports and native browser fallback. Run the compiler/build alongside strict ESLint; lint alone cannot validate routing or Worker deployment.
6. Report changed decisions, affected states, exact checks and uncertainty. Visual baselines and performance budgets are deferred.

## Approved shared shell

For ztd.me shell consumers, use `@ztd-me/frontend` for common chrome and validated non-sensitive appearance/UI locale. Default Neutral + System, six palettes, and the fixed © Zeithrold/project GitHub/hello@ztd.me footer are approved. Public-site appbars do not require navigation; Memory owns business navigation. Keep business content, auth, account and clock/workspace state local. Shared domain cookies are untrusted UI input; preview/local policies stay isolated. This shell contract supersedes project-local chrome defaults without granting sync ownership over components or production files. Preserve native strict gates and upstream license obligations. Install a promoted exact registry version only after fresh-consumer verification; a stage placeholder or local copy is not final integration.
