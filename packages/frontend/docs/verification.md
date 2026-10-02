# Verification contract

`pnpm install --frozen-lockfile` and `pnpm run check` run the native gates. Keep the release-age, integrity and no-downgrade policies; the approved @ztd-me/* exception changes only release age. The pinned Radix patch changes declarations only and is absent from the independent consumer.

The initial cloud implementation was checked on Node 24.19.0, pnpm 11.22.0, React 19.3.0, TypeScript 6.0.3, ESLint 10.11.0, Vite 8.3.1 and Playwright 1.62.0/Chromium 151. CI repeats the complete package checks on Node 22 and 24 for the PR's source commit.

- Strict ESLint, CSS variable validation, full TypeScript declaration checking and package build pass.
- Eight unit cases cover normalization, enum defaults, non-sensitive schema boundaries, trusted deployment policy, duplicate/malformed/oversized/future cookies, weighted locale negotiation, limited legacy extraction and the reproduced/patched Radix declaration conflict.
- An independent consumer installs the actual packed artifact with pnpm, installs again with a frozen lockfile, imports every export, verifies both WOFF2 files and included licenses, rejects invalid typed API usage, checks declarations with skipLibCheck=false/exactOptionalPropertyTypes=true, and builds actual React SSR and client assets. It uses unpatched registry Radix dependencies.
- Fourteen Chromium scenarios verify SSR/hydration and loaded Inter, first paint with JavaScript blocked and CSP, system/explicit mode independence, keyboard skip/menu/Select interactions, ref/focus/inert restoration, all six palettes in both modes, Chinese controls at 320/390/768/1024/1440 widths, rejected storage, legacy/business storage separation, synthetic subdomain sharing/preview isolation, same-origin notifications, fullscreen portals and application slots. Denied cookie getters/setters are exercised on desktop/mobile, during Memory legacy extraction, and through refresh/recovery while preserving the SSR snapshot and business drafts.
- The sampled page and both popup states produce 44 complete Axe WCAG A/AA scans with zero violations. Fifteen named captures support review; they are evidence, not approved visual baselines.
- Repository Go tests and vet pass. Read-only zt inspection discovers the new frontend module without running checks or changing project files.

Reports, screenshots, logs and the packed-consumer path/checksum receipt stay under `.artifacts/` and are retained by CI on success or failure. Browser subdomains are intercepted and served locally; no production or authenticated service is contacted. The fixture permits Radix style attributes and per-response nonce-bearing style elements while disallowing inline scripts. Consumers remain responsible for their actual CSP, framework/RSC integration, locale-sensitive caching and business content. This fixture does not certify those integrations, screen readers or every browser.

## After owner promotion

Staging does not make the package installable. Once the owner has approved the staged version with npm 2FA, run from this source revision:

```sh
node scripts/registry-smoke.mjs 0.1.0
```

The script checks the exact public registry version/integrity, creates a new pnpm consumer using that exact version rather than a local file, and repeats import/types/build/browser verification. Only a successful `.artifacts/registry-0.1.0.json` receipt establishes the public installation gate. Do not substitute a registry placeholder, workflow staging receipt or local tarball for this result.

## Companion CSS helper patch

The original frontend-checks 0.1.0 rejects valid Tailwind block custom variants because Stylelint sees `&` without a native scoping root. The companion 0.1.1 candidate adds only `ignoreAtRules: ['custom-variant']` to the still-enabled nesting rule. Its regressions check the exact system-dark source, native Tailwind 4.3.3 compilation, unscoped root/media selectors outside the directive, and retained property/token/color failures. Source and packed CLI checks pass. Consumer revalidation must use this fixed source/helper candidate; final dependency locks must wait for owner promotion and fresh registry verification of 0.1.1.
