---
name: frontend-verification
description: Use when establishing or running frontend lint, CSS, types, unit, build, browser accessibility and artifact gates with exact execution evidence and explicit blocked checks.
---

# Frontend verification

Read js-ts-testing, the design contract and native scripts. Select ui-foundation/ui-web for UI work and frontend-engineering for architecture, preferences and data boundaries. Skill guidance does not itself run checks.

1. Inspect modules/script contents using zt inspect/plan. Confirm pnpm and browser/Worker preconditions; keep discovery read-only.
2. Define an ordered explicit profile in zt.json. Follow [gate integration](references/gates.md). Required absent/failed checks must fail; do not relabel them as warnings to obtain a pass.
3. Run strict JS/TS lint, CSS validation, compiler, focused unit/integration tests and the native build. Preserve @ztd-me/eslint limits independently of existing project violations.
4. Run browser interactions/Axe against the intended built runtime. Follow [evidence](references/evidence.md). Distinguish real services, mocked APIs and native fallback coverage.
5. Retain reports, CSS findings, failure logs, traces and named captures under the artifact root. Upload on CI success/failure; unretained /tmp captures do not fulfill this contract.
6. Report revision, tool/package versions, commands, scope, results, artifact paths and blocked/manual work. Visual baseline management and performance budgets are deferred.

## Generic shared shell

`@ztd-me/frontend` provides reusable chrome, validated non-sensitive appearance/UI locale and explicit persistence mechanisms. Its defaults are Neutral + System and six palettes. Consumers supply their own brand, footer/repository/contact content, cookie name/domain/Secure choice, optional storage notification key and business slots. Tools must not select policy through project names or hostnames, enforce a repository-host allowlist, or carry legacy storage mappings. Remove rejected legacy mappings entirely rather than moving them into consumers; old business/auth storage stays untouched. Shared domain cookies are untrusted UI input. Consumers own development/preview isolation, deployment, authentication, account state and business navigation. Generic shell guidance grants no sync ownership over project components or production files. Preserve native strict gates and upstream licenses. Final integration requires a promoted exact registry version and fresh-consumer verification; staging or local packed evidence alone does not establish public installation.
