# Contributing

Keep `docs/` and package documentation focused on maintained contracts, architecture, installation,
migration and troubleshooting. Update links when moving or removing a document. Preserve unrelated
content and all required license notices.

Put disposable test and review evidence in ignored `.artifacts/`, `.zt/artifacts/` or `artifacts/`
directories. This includes screenshots, videos, traces, run-specific JSON measurements, receipts and
comparison reports. Keep reusable test fixtures in their test directories. Link CI runs and retained
artifacts in the PR description; do not copy their output into `docs/reviews/` or package documentation.
The existing browser and source-install scripts already write to artifact directories, which CI
uploads on success or failure.

`packages/ui` owns the canonical UI source and its private verification harness. Keep
`packages/frontend-checks` separate; it remains a published verification helper. Do not publish the UI
harness to npm. The existing `@ztd-me/frontend@0.2.0` artifact remains available.

When changing source registry documentation or files, regenerate the payload from the repository root:

```sh
pnpm dlx shadcn@4.21.1 build registry.json --output registry
```

Review the generated diff and run the native package checks with pnpm. A source revision requires a
fresh public installation with its full approved SHA before consumer acceptance; local or preview
receipts do not establish that gate. Keep credentials and security configuration out of evidence.

## GitHub Actions

Follow the [workflow contribution contract](skills/frontend-engineering/references/github-actions.md)
when creating or changing frontend CI/deployment workflows. It defines the common Verify → Deploy
sequence, naming, pinned toolchains/actions, pnpm installs/cache, required checks and failure artifacts.
The complete reference ships with the frontend-engineering Skill. Project-specific commands,
credentials, coverage, production builds and deployment ownership checks stay in their repositories.
Preserve established owner approval and publishing boundaries.
