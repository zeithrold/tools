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
