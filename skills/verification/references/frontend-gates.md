# Native gate integration

zt check runs native argv/profiles in zt.json. It does not install dependencies, provision browsers/services or deploy. inspect, plan and sync --plan remain read-only. check executes scripts and writes a unique evidence directory; review script contents first.

Order a frontend profile by dependencies: lint, css, typecheck, unit, build, e2e, a11y. Explicit commands handle project script names. A zero exit establishes that command's result, not inferred route/state coverage. Avoid recursive aggregate scripts; pnpm check can wrap zt check after native subchecks are mapped.

Profile entries default to required. Deliberate warn/off choices need project justification. The runner stops after a required failure/blocker, records later checks as not_run and preserves evidence. Missing browser/server is blocked/failed work. Missing declared artifact sources fail collection.

The helper exposes ztd-css, checkCss, assertAccessible, captureState and verificationArtifacts. Keep Playwright webServer/routes/projects local. Apply shared artifact paths while retaining native use settings. Include theme/import declaration sources and exact runtime variables in CSS configuration.

The helper's public release uses automatic pnpm staging and owner 2FA promotion.
Exact registry-version installation is a separate operator setup action after
fresh-consumer verification; select the approved version and retain its lockfile.
Optional, versioned setup history is available in the
[repository integration guide](https://github.com/zeithrold/tools/blob/b8db754ed2b7ba08bad30497840c926c8c196b8d/docs/frontend-tooling.md);
it is not a required resource or permission to install dependencies.

The existing user-authorized @ztd-me/* exception applies only to release age;
this Skill does not authorize new exceptions. Preserve other packages' age gates,
integrity checks, no-downgrade and the existing exact semver@6.3.1 trust exception.
Do not add a scope trust exclusion. CI tarballs/checksums/source SHAs support
review; the first-package stage placeholder is not the release.
@ztd-me/eslint@0.1.1 was published separately; that history does not select the
version for a consumer. Skill sync installs complete selected directories with
version/hash/local-edit protection; it does not install packages or rewrite
contracts.
