# Verification contract

`pnpm install --frozen-lockfile` and `pnpm run check` run the native gates. Keep the release-age, integrity and no-downgrade policies; the approved @ztd-me/* exception changes only release age. The pinned Radix patch changes declarations only and is absent from the independent consumer.

The 0.2 cloud implementation was checked on Node 24.19.0, pnpm 11.22.0, React 19.3.0, TypeScript 6.0.3, ESLint 10.11.0, Vite 8.3.1 and Playwright 1.62.0/Chromium 151, using the published frontend-checks 0.1.1. CI repeats the complete package checks on Node 22 and 24 for the PR's source commit.

- Strict ESLint, CSS variable validation, full TypeScript declaration checking and package build pass.
- Eleven unit cases cover normalization, enum defaults, non-sensitive schema boundaries, generic policy defaults/validation, existing explicitly configured version-1 cookies, duplicate/malformed/oversized/future cookies, weighted locale negotiation, consumer-owned footer links/URL validation and the reproduced/patched Radix declaration conflict.
- An independent consumer installs the actual packed artifact with pnpm, installs again with a frozen lockfile, imports every export, verifies both WOFF2 files and included licenses, rejects invalid typed API usage, checks declarations with skipLibCheck=false/exactOptionalPropertyTypes=true, and builds actual React SSR and client assets. It uses unpatched registry Radix dependencies.
- Sixteen Chromium scenarios verify SSR/hydration and loaded Inter, first paint with JavaScript blocked and CSP, system/explicit mode independence, keyboard skip/menu/Select interactions, ref/focus/inert restoration, all six palettes in both modes, Chinese controls at 320/390/768/1024/1440 widths, rejected storage, synthetic consumer domain sharing/host-only isolation, optional same-origin notifications, fullscreen portals and application slots. They also prove old keys/unrelated cookies/auth/business records remain untouched, no mirror means zero localStorage access, and a synthetic consumer supplies Codeberg/contact/copyright links. Denied cookie getters/setters are exercised on desktop/mobile and through refresh/recovery while preserving the SSR snapshot and business drafts.
- The sampled page and both popup states produce 46 complete Axe WCAG A/AA scans with zero violations. Fifteen named captures support review; they are evidence, not approved visual baselines.
- Repository Go tests and vet pass after the reusable Skill guidance change. No detectors or planning behavior changed.

Reports, screenshots, logs and the packed-consumer path/checksum receipt stay under `.artifacts/` and are retained by CI on success or failure. Browser subdomains are intercepted and served locally; no production or authenticated service is contacted. The fixture permits Radix style attributes and per-response nonce-bearing style elements while disallowing inline scripts. Consumers remain responsible for their actual CSP, framework/RSC integration, locale-sensitive caching and business content. This fixture does not certify those integrations, screen readers or every browser.

## After owner promotion

Staging does not make the package installable. Once the owner has approved the staged version with npm 2FA, run from this source revision:

```sh
node scripts/registry-smoke.mjs 0.2.0
```

The script checks the exact public registry version/integrity, creates a new pnpm consumer using that exact version rather than a local file, and repeats import/types/build/browser verification. Only a successful `.artifacts/registry-0.2.0.json` receipt establishes the public installation gate. Do not substitute a registry placeholder, workflow staging receipt or local tarball for this result. This source candidate has passed local packed-consumer verification; public 0.2.0 installation remains pending owner merge, automatic staging and promotion.

## Published CSS helper

Development verification now pins published frontend-checks 0.1.1. Its `ignoreAtRules: ['custom-variant']` exception is limited to Tailwind's native scoping directive; nesting, property, token and color rules remain active. No package-manager security settings or CSS standards are relaxed in this upgrade.
