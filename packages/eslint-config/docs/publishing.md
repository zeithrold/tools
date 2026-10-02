# Stage-only publication

`.github/workflows/publish-eslint.yml` verifies and stages `@ztd-me/eslint`. Every upload uses `pnpm stage publish`; the workflow never directly publishes or approves a stage. `NPM_TOKEN` has Read and write (stage only) rights, with Bypass 2FA disabled. The token is bound to `https://registry.npmjs.org` in the stage step only, kept in memory, and never printed. Installation/build/test/pack steps have no token.

The bootstrap trigger is a push to an explicit `release/eslint-stage-*` branch. Once the workflow exists on the default branch, `workflow_dispatch` supports either `token` or `oidc` authentication. Neither trigger merges source. A separate draft PR carries repository review.

The verification jobs use a frozen pnpm lockfile on Node 22 and 24. The Node 24 job packs the tested package. Stage jobs consume that exact artifact and save the npm stage response as a workflow artifact. The Actions run, source SHA, tarball checksum and stage ID are review evidence. Do not resubmit the same stage after an ambiguous timeout; inspect the existing stage before retrying.

## Bootstrap

The first upload uses the user-provided repository `NPM_TOKEN` secret. The package did not exist on npm at implementation start. Staging does not make it installable, and creating a repository secret does not establish an npm Trusted Publisher.

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

The `stage-oidc` job runs on GitHub-hosted `ubuntu-latest`, with job-scoped `id-token: write` and `contents: read`. It is selected only by an explicit dispatch with authentication `oidc`. No npm token is passed to that job. The token bootstrap job has only `contents: read`. The assistant does not create the trust grant or change package security settings.

References: [pnpm stage](https://pnpm.io/cli/stage), [pnpm token authentication](https://pnpm.io/blog/releases/11.10), [npm staged publishing](https://docs.npmjs.com/staged-publishing/), [npm Trusted Publishers](https://docs.npmjs.com/trusted-publishers/).

## Public-release verification

After promotion, inspect the exact public registry version, then run:

```sh
node scripts/registry-smoke.mjs 0.1.0
```

This creates a fresh consumer, installs the exact registry version using pnpm, checks default/named ESM exports and declarations, and runs JS/TS/framework smoke checks. It keeps pnpm supply-chain protection active. If the freshly published version is blocked by pnpm's minimum release age or another policy, wait for policy eligibility and retry; do not set exceptions or lower the policy. A stage response, registry metadata alone, or a local tarball install is insufficient proof of public installation.
