# Inherited pnpm policy

Antfu 9.5.1 enables `pnpm/yaml-enforce-settings` at error severity and requires these workspace settings:

```yaml
minimumReleaseAgeExcludePrune: true
shellEmulator: true
trustPolicy: no-downgrade
```

These inherited checks are separate from the 198-entry research catalog. They are intentional upstream defaults; this package does not silently disable them when a consumer fails lint. Changing a consumer's dependency policy requires that project's approval and native installation/build checks.

[`minimumReleaseAgeExcludePrune`](https://pnpm.io/settings/dependency-resolution#minimumreleaseageexcludeprune) was added in pnpm 11.22.0. It removes release-age exception entries that no longer match packages/versions in the lockfile. It retains an active package-wide exception, including `@ztd-me/eslint`, while that package remains installed; this does not automatically expire the exception after the age cutoff. Wildcard entries are retained. Older pnpm versions do not provide the pruning behavior merely because the YAML key is present.

[`trustPolicy: no-downgrade`](https://pnpm.io/settings/dependency-resolution#trustpolicy) was added in pnpm 10.21.0 and rejects a package release whose publishing trust drops below earlier releases. Trusted publishing ranks above provenance-only publishing, which ranks above publication without either. Release-age exceptions do not bypass this independent policy. Installation must check the actual dependency graph; switching a package from token publication to trusted publication can make a later token-only release a downgrade.

[`shellEmulator: true`](https://pnpm.io/settings/other#shellemulator) makes pnpm execute scripts with its portable JavaScript bash-like shell. This changes script semantics for portability and does not grant additional operating-system privileges. Verify scripts that depend on shell-specific behavior using the target pnpm version.

Consumers using pnpm 10 must review a pnpm upgrade before relying on the pruning requirement. A pnpm 11 migration must preserve explicitly allowed dependency builds when converting `onlyBuiltDependencies` to `allowBuilds`; it must not broaden build permissions. Do not disable release-age/trust checks or add broad trust exclusions to get an installation through. A lint pass alone does not prove that the pinned package manager implements every setting.

## Current dependency integration blocker

On 2026-10-02, enabling `no-downgrade` on the selected package graph rejected `semver@6.3.1`. Both 0.1.0 and the 0.1.1 framework patch depend on stable `eslint-plugin-react-hooks@7.1.1`, which requires `@babel/core:^7.24.4`. Resolved Babel core 7.29.7 and its compilation-target helper require `semver:^6.3.1`. Registry inspection found 27 eligible stable Babel 7 versions, all requiring that same semver range; 6.3.1 remains the latest 6.x version.

Semver 6.3.1 was published on 2023-07-10 without provenance, after provenance-bearing 7.5.4 on 2023-07-07. This meets pnpm's publication-date policy even though the major versions differ; the age exception for this ESLint package does not fix it. A policy failure is not proof of a compromised release, but it remains an installation blocker.

Babel 8 is outside the Hooks plugin's supported dependency range. Forcing Babel 8 or semver 7 through a cross-major override is not a supported package upgrade. A coherent upstream remedy is a compatible Babel 7 release using an accepted semver dependency, or stable React Hooks support for Babel 8, followed by a tested ESLint package release. No trust exclusion or policy weakening is included in this patch.
