# Verification contract

`pnpm install --frozen-lockfile` and `pnpm run check` run the native gates. Keep release-age, integrity,
no-downgrade and other existing pnpm policies. The approved @ztd-me/* exception changes only release age.
No TLS bypass, browser security exception or trust-store change is part of verification.

The source harness pins pnpm 11.22.0, React 19.3.0, TypeScript 6.0.3, ESLint 10.11.0, Vite 8.3.1,
Playwright 1.62.0 and published frontend-checks 0.1.1. CI repeats complete checks on Node 22 and 24 for
the PR's exact source SHA. The already published frontend 0.2.0 remains unchanged and available; this
source transition is not a new npm release.

## Native and consumer checks

- Strict ESLint, CSS variable validation, full TypeScript declaration checking and build.
- Eleven unit cases covering preference boundaries, generic policy, cookie/locale validation, footer
  URLs and the reproduced/patched Select declaration defect.
- A fresh consumer installs the actual packed artifact using pnpm, reinstalls with a frozen lock,
  imports all exports, validates Google Fonts CSS and six complete OFL notices plus MIT without font binaries,
  rejects invalid typed API usage, and builds real React SSR and client assets.
- A separate source consumer uses the real pinned shadcn CLI to preview/install the item, compares all
  installed file bytes, confirms no frontend runtime dependency, applies the documented declaration
  patch and runs strict lint/CSS/types, eleven units, SSR/client build and the complete browser suite.
  It preserves pnpm security settings and enables `skipLibCheck: false`.

The browser suite covers SSR/hydration, first paint without JavaScript, nonce-bearing CSP, independent
mode/palette selection, all six palettes/light/dark/system, responsive Chinese controls, keyboard/focus,
entry/exit and reduced motion, 44px touch targets, inert cleanup, fullscreen portals, storage rejection
and application slots. It preserves unrelated auth/business records and project-owned footer/policy.
The typography specimen adds actual English/CJK glyph usage, weight 600 and composed Noto emoji checks,
resource evidence and Axe. Reports and captures are evidence; they are not approved visual baselines.

Axe's CSSOM preloader copies CSS into a temporary document. The fixture adapter gives those analysis
styles the existing response nonce only while the scan runs. Its CSSOM reader also reuses the exact
Google CSS response already loaded by the real browser, avoiding an extra analysis XHR blocked by
the fixture's strict `connect-src`. Native DOM and XHR methods are restored before interaction.
No font file, initial stylesheet request or glyph check is mocked. Application injection and requests
remain checked normally; no CSP origin, rule, scan tag or browser security restriction is disabled.
Reports retain incomplete results as well as violations.

The package's selected Select 2.3.7/Popper 1.3.7 declarations have an upstream TS2320 conflict. The
registry supplies the exact declaration-only patch and consumer-owned pnpm recipe. JavaScript remains
unmodified. The old compiled API hides this declaration conflict from its packed consumer. See
[Radix compatibility](radix-compatibility.md) and the source registry installation instructions.

## Actual Google Fonts and isolated Cloud preview

The default browser gate uses the real Google Fonts API and gstatic font requests under the fixture's
CSP. There is no network stub or local-font substitution by default. The public source-install gate
rejects local preview mode. The font identities, requests and transfers must pass in a normal browser
environment before actual remote font loading is considered verified.

This Cloud executor's Chromium rejected the API certificate with `ERR_CERT_AUTHORITY_INVALID`, while
standard CLI TLS verification returned HTTP 200. The owner separately authorized a local Noto preview
in the test harness. To prepare it with the executor's normal TLS/proxy and then run preview checks:

```sh
NODE_USE_ENV_PROXY=1 node scripts/prepare-font-preview.mjs
ZTD_LOCAL_FONT_PREVIEW="$PWD/.artifacts/local-noto-preview" NODE_USE_ENV_PROXY=1 pnpm run test:pack
ZTD_LOCAL_FONT_PREVIEW="$PWD/.artifacts/local-noto-preview" NODE_USE_ENV_PROXY=1 pnpm run test:source
```

Use `NODE_USE_ENV_PROXY=1` only where an existing authorized proxy is supplied by the environment. The
preview server serves a cached Google CSS response and lazily downloads the same official gstatic font
files with normal Node TLS to ignored `.artifacts/`. Only the synthetic server replaces its CSS import
with an explicitly named local-preview route. Production source/CSS and the registry payload retain
the direct API request. Neither preview adapter, cached files nor download script is delivered in the
registry. Preview browser artifacts are kept separately in `.artifacts/browser-local-font-preview`;
font evidence sets `actualGoogleFontsBrowserLoad: false`. Captures require a local-font-preview label.

Preview success establishes layout/glyph/weight/emoji behavior with equivalent Noto files; it does not
verify actual browser Google Fonts requests, production CSP delivery, remote transfer performance or
network availability. Default CI runs use the real API. If those cannot complete, record the exact
blocker and arrange verification in an authorized normal/local browser; do not waive the remote gate.
See [typography](fonts.md) for API/privacy/CSP and font mutability.

## Public source acceptance

After the owner merges the approved revision, check out its full source SHA and run from this directory:

```sh
node scripts/source-smoke.mjs <SOURCE_SHA>
```

The script creates a fresh consumer, reads the exact public GitHub payload, compares it with the source,
checks installed file bytes and repeats native/browser gates. Only a successful receipt with
`publicInstallationVerified: true` establishes this post-merge gate. Local tarballs, loopback source
installation and preview receipts do not replace it. Consumer migration PRs must preserve local source
edits and business policy and obtain their own native/browser and owner acceptance.

Reports, screenshots, logs and receipts remain under `.artifacts/` and are retained by CI on success or
failure. Intercepted synthetic subdomains serve local business fixtures; Google Fonts is the default
suite's explicit external dependency. The fixture does not certify production CSP, every framework,
locale-sensitive caching, screen readers or every browser. Go tests/vet apply to bundled Skill changes;
no detector or planning behavior changes in this UI transition.
