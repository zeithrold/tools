# Frontend tooling integration

This phase adds shared guidance and native check execution to tools. Consumers retain product-specific branding, layout values, business rules, framework/runtime configuration and their existing strict ESLint profile. Visual snapshot baselines and performance budgets are deferred. Changes are reviewed through a draft PR; no consumer migration or package publication occurs in this phase.

## Distribution

- `@ztd-me/eslint@0.1.1` is already published. Its standards are unchanged.
- `zt` and complete Skill directories are shipped together. Pin an exact reviewed tools commit with `go install github.com/zeithrold/tools/cmd/zt@<SHA>` after it is reachable, or use the CI-built Linux binary from that revision. There is no official CLI release tag yet.
- `@ztd-me/frontend-checks@0.1.0` is private and un-published. CI uploads its tarball, checksum and source SHA. Download the artifact for the exact reviewed run, verify `SHA256SUMS`, retain it in a project-owned vendor location, then use `pnpm add -D ./vendor/ztd-me-frontend-checks-0.1.0.tgz`. Do not depend on an expiring Actions download URL. Alternatively checkout the exact tools revision and run `pnpm pack --pack-destination <DIR>` in packages/frontend-checks. An npm release would require separate authorization before publication; this route needs no new credentials or trust grant.

Set the consumer `packageManager` to its approved pnpm version (currently 11.22.0), retain its dependency policies and test the resulting frozen lockfile. The helper's peer is Playwright Test ^1.62.0; package validation pins 1.62.0. CI validates on Node 24. Browser binaries are installed explicitly by the consuming environment.

## Explicit project profile

```json
{
  "schemaVersion": 1,
  "modules": [{
    "id": "web", "path": ".", "stack": "js-ts",
    "profiles": { "frontend": ["lint", "css", "typecheck", "unit", "build", "e2e", "a11y"] },
    "commands": {
      "css": ["pnpm", "run", "lint:css"],
      "e2e": ["pnpm", "run", "test:browser"],
      "a11y": ["pnpm", "run", "test:a11y"]
    }
  }],
  "skills": ["js-ts-testing", "ui-foundation", "ui-web", "frontend-engineering", "frontend-verification"],
  "lint": { "js-ts": "@ztd-me/eslint" }
}
```

Commands are argv arrays, not shell strings. They run from the declared module directory. Native package scripts may themselves invoke a shell; read them before running. The `lint` map is metadata and does not install/configure a package. Modules are explicit; workspace packages are not recursively inferred. Select every required Skill; sync does not automatically install Skill dependencies.

For missing command overrides, planning recognizes lint/lint:check; lint:css/check:css; typecheck/check:types/check-types; test:unit/test-unit/test; build; test:e2e/e2e:test/e2e; and test:a11y/check:a11y. The declared packageManager takes precedence; otherwise one unambiguous supported lockfile selects pnpm/npm/yarn. Missing/ambiguous/unsupported managers produce a blocker, not an npm guess.

```sh
zt inspect --root . --json
zt plan css --module web --root . --json
zt sync --root . --plan
zt sync --root .
zt check --module web --profile frontend --root . --json
```

`inspect`, `plan`, and `sync --plan` do not execute or write. `sync` only changes selected managed Skill directories and its lock, refusing conflicting local edits. `check` validates the complete profile, creates a unique directory below `.zt/artifacts`, then executes in order. Profile entries default to required. `expect` can explicitly declare required/warn/off; it applies only to selected entries. Required failed/blocked commands stop the profile; subsequent checks are `not_run`. Warning failures continue and remain recorded. Overall failure exits 1. Zero native exit codes do not certify inferred test coverage.

`--timeout 10m` is the default per-command limit; `--artifacts DIR` changes the artifact parent. Context cancellation/timeouts kill the process group on Unix. Other platforms terminate the immediate process; native test runners must clean up child servers. The planner adds `--config.verify-deps-before-run=error` to pnpm argv and the runner sets the equivalent environment override for nested pnpm scripts, preventing pnpm 11 from auto-installing stale/missing dependencies before scripts. Install explicitly beforehand. This per-run setting does not weaken release-age/trust/build policy or alter persistent configuration. The CLI never explicitly installs or deploys; reviewed native scripts retain their own semantics.

## Native helper and retained evidence

Add `"lint:css": "ztd-css ./css-check.config.mjs"` to native scripts. Follow the [helper README](../packages/frontend-checks/README.md) for cross-file token sources and Playwright helpers. Keep `test:a11y` focused on meaningful real states, not an empty placeholder. Existing browser suites may include those checks; declare the command mapping/coverage explicitly without rerunning an identical suite merely to satisfy a name.

Under `zt check`, `ZT_ARTIFACTS_DIR` points to the unique run directory. Playwright's shared artifact settings and CSS CLI write there. For existing outputs elsewhere, declare module `artifacts` paths such as `["playwright-report", "test-results"]`; these are copied under `collected/` after execution, including failures. Missing paths, symlinks, non-regular files and oversized collections fail rather than silently dropping evidence. Keep paths within the module and keep outputs from older runs separate.

Upload the complete `.zt/artifacts` root on both success and failure:

```yaml
- run: zt check --module web --profile frontend
- uses: actions/upload-artifact@v7
  if: always()
  with:
    name: frontend-evidence
    path: .zt/artifacts/
    include-hidden-files: true
    if-no-files-found: error
```

The report records tools version, profile, native argv/directory/status/exit/duration and artifact paths. Attach the exact source revision and lockfile/tool versions in CI. CSS is a static inventory, Axe covers selected DOM states/rules, screenshots retain human-review evidence, and a framework build is separate from deployment proof. Preserve native fallback behavior, unknown-input validation, deterministic preference hydration, translations and main-only deployment verification in project-owned tests/workflows.

The existing consumer example is a target profile, not a claim that its scripts already exist. Stage 2 maps each project's native checks, resolves token/translation/coverage findings, then runs all gates. Parent confirmation of validated/distributable tools and owner review precede that rollout.
