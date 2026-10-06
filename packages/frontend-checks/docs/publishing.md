# Public stage-only release

`@ztd-me/frontend-checks@0.1.0` is published from source `3f9a3a7d33befc5a954ba1e86d3aa6d72e2c762f`. Its [automatic staging run](https://github.com/zeithrold/tools/actions/runs/36995178595) created stage `bc4c19ac-2c8b-4525-88f0-e0ec1e333903`; owner promotion made the exact verified tarball public. SHA-256 is `a3b7209f90746bf71c1b5b4ef69910a70ff747acc394696a778996f69efcb87c`.

`.github/workflows/publish-frontend.yml` runs automatically on relevant pushes to `main`. It uses pnpm 11.22.0, frozen installation, strict lint, declarations, CSS fixtures, Chromium tests and fresh packed-consumer checks on Node 22 and 24. The Node 24 job packs the tested bytes and records the immutable source SHA and checksums. A separate job verifies those checksums and source identity before staging that exact tarball. It never directly publishes or promotes a version. Documentation/policy changes without a version bump verify and skip the public version rather than republishing it.

The existing repository `NPM_TOKEN` secret is the authorized route. It must permit Read and write (stage only) for this package under `@ztd-me`, with Bypass 2FA disabled. Its actual scope for this new package cannot be inferred from its success publishing ESLint. If permission is missing, the package owner must supply the narrowly authorized access; the workflow fails without creating credentials or expanding rights. Organization administration rights alone do not grant package publication rights. The token is bound to the registry in memory in the stage step only; installation, checks and packing receive no token.

Before upload, the shared stage guard checks public versions and the authenticated pending-stage list. Only pnpm's structured public-metadata HTTP 404 is interpreted as a package without published versions. Missing tokens, stage-list authorization failures, network failures and malformed responses remain blockers. An existing stage is preserved and its ID recorded rather than uploading a duplicate. An ambiguous upload outcome requires owner reconciliation before retry. No manual workflow trigger, authentication selector or OIDC fallback is configured.

## Owner review and promotion

1. Review and merge the tested draft PR to `main` using the repository's normal owner-controlled review process.
2. Let the automatic workflow finish. Check the Actions source SHA, `frontend-package-<SHA>` artifact, `SHA256SUMS` and `frontend-stage-result-<SHA>` receipt. Review the pending stage's package name, version and contents against that exact artifact.
3. Promote the stage in npm's website with the owner's 2FA, or use an already-authenticated `pnpm stage approve <stage-id>` session and complete proof of presence. Do not share the credential or OTP in chat.
4. Verify the public registry bytes and fresh consumer as described below before dispatching consumer migrations.

First-package staging creates a public `0.0.0-stage` placeholder so package settings can be accessed. This placeholder does not make `0.1.0` installable. A successful stage receipt is also not a completed public release.

## Exact registry verification

From the reviewed source checkout, download the automatic release run's exact `frontend-package-<SHA>` artifact and verify `SHA256SUMS` and `source-commit.txt`. Once the owner has promoted `0.1.2`, run:

```sh
pnpm exec playwright install chromium
pnpm run test:registry 0.1.2 /path/to/verified/package.tgz
```

The script checks the exact public package/version and registry SHA-512 integrity against the reviewed tarball, then creates a fresh pnpm consumer. It installs the exact registry version, reruns frozen installation, imports both `/css` and `/playwright`, invokes `ztd-css`, checks TypeScript declarations with strict settings and executes a deliberately failing browser Axe scan that must retain the violation evidence. Browser infrastructure errors cannot satisfy that assertion.

The user authorizes `minimumReleaseAgeExclude: ['@ztd-me/*']` in tools verification and the three consumer projects. The consumer retains `minimumReleaseAge: 1440`, `minimumReleaseAgeStrict: true`, pruning and `trustPolicy: no-downgrade`. Other packages still wait 24 hours. Scope packages keep integrity/trust checks; no scope-wide trust exclusion is authorized. Preserve the existing exact `semver@6.3.1` trust exception wherever the ESLint graph needs it. Pin the verified registry version:

```sh
pnpm add -D --save-exact @ztd-me/frontend-checks@0.1.2 @playwright/test@1.62.0
pnpm install --frozen-lockfile
```

## Optional future Trusted Publisher

An npm trust grant would be package-specific: GitHub owner `zeithrold`, repository `tools`, workflow filename `publish-frontend.yml`, no environment, stage publishing only. The owner must configure it separately; this release does not create one. OIDC supports stage upload but cannot authenticate the pending-stage lookup. Switching requires a reviewed guard redesign and job-scoped `id-token: write`, rather than removing the token from the current job or adding a fallback path. Owner proof of presence still controls promotion.

References: [pnpm stage](https://pnpm.io/cli/stage), [npm staged publishing](https://docs.npmjs.com/staged-publishing/), [token permissions](https://docs.npmjs.com/creating-and-viewing-access-tokens/), [Trusted Publishers](https://docs.npmjs.com/trusted-publishers/).
