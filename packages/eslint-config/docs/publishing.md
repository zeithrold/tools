# Stage-only publication

`.github/workflows/publish-eslint.yml` verifies and stages `@ztd-me/eslint`. Every upload uses `pnpm stage publish`; the workflow never directly publishes or approves a stage. `NPM_TOKEN` has Read and write (stage only) rights, with Bypass 2FA disabled. The token is bound to `https://registry.npmjs.org` in the stage step only, kept in memory, and never printed. Installation/build/test/pack steps have no token.

The only trigger is a push to `main` affecting `packages/eslint-config/**` or `.github/workflows/publish-eslint.yml`. There is no manual trigger, temporary release branch, or authentication selector. Package versions are independent of the Go CLI's `v*` tags: review and merge an ESLint package version bump to request a release. Source changes without a version bump still run verification, but the existing version is skipped.

The verification jobs use a frozen pnpm lockfile on Node 22 and 24. The Node 24 job packs the tested package. A single staging job consumes that exact artifact and saves the npm stage response as a workflow artifact. Runs share one concurrency group without cancellation. Before upload, the script checks public registry metadata and the authenticated pending-stage list. Published or already-staged versions produce a skip receipt, preserving an existing stage ID. Registry/authentication errors and malformed responses fail before upload. The Actions run, source SHA, tarball checksum and stage ID are review evidence. Do not resubmit after an ambiguous timeout; inspect the existing stage before retrying.

## Maintainer approval

The user-provided stage-only repository `NPM_TOKEN` is the currently verified authentication route. Staging does not make a version installable, and creating a repository secret does not establish an npm Trusted Publisher. The first release, `0.1.0`, was staged and owner-approved separately; normal workflow runs must skip it rather than recreate its stage.

After the stage succeeds, the package owner must review and promote the stage using npm's website and their 2FA, or run an authenticated `pnpm stage approve <stage-id>` and complete the required proof of presence. Keep the OTP and credential out of chat. This user-controlled promotion is required before stage 1 can be called complete.

## Trusted Publisher settings

The package owner configures the trust grant in npm's package settings after npm permits access to those settings. Exact GitHub values:

| Setting | Value |
| --- | --- |
| Organization or user | `zeithrold` |
| Repository | `tools` |
| Workflow filename | `publish-eslint.yml` |
| Environment name | Leave empty; this workflow uses no GitHub environment |
| Allowed actions | Stage publishing only; do not enable direct publish or dist-tag mutation for this workflow |

These are the exact settings for a future OIDC migration, not evidence that the grant exists. After the owner confirms a stage-only grant, update the existing staging job to job-scoped `id-token: write`, remove `NPM_TOKEN` and its in-memory token binding, and use pnpm's native OIDC authentication on the same GitHub-hosted runner. Keep the single automatic trigger and the workflow filename. Do not add a parallel authentication job or a manual selector. The assistant does not create the trust grant or change package security settings.

References: [pnpm stage](https://pnpm.io/cli/stage), [pnpm token authentication](https://pnpm.io/blog/releases/11.10), [npm staged publishing](https://docs.npmjs.com/staged-publishing/), [npm Trusted Publishers](https://docs.npmjs.com/trusted-publishers/).

## Public-release verification

After promotion, inspect the exact public registry version, then run:

```sh
node scripts/registry-smoke.mjs 0.1.0
```

This creates a fresh consumer, installs the exact registry version using pnpm, checks default/named ESM exports and declarations, and runs JS/TS/framework smoke checks. The temporary store/cache settings are inherited by pnpm's pre-run install checks, including in cloud sandboxes without a writable home directory. Supply-chain settings are left unchanged. pnpm 11's default non-strict release-age policy can automatically record a version-specific exception for a fresh explicit dependency; projects with strict release-age policies must wait until their cutoff. If an active policy rejects installation, wait for eligibility and retry; do not manually set exceptions or lower the policy. A stage response, registry metadata alone, or a local tarball install is insufficient proof of public installation.
