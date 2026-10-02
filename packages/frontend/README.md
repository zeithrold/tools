# @ztd-me/frontend

Shared shadcn new-york/Radix chrome for React 19 projects. The default is **Neutral + System**; mode and palette are independent. All six palettes support light and dark. Business content, navigation, identity and application state stay with the consumer.

```sh
pnpm add @ztd-me/frontend@0.1.0
```

Install from npm only after the owner promotes the staged version. Requires Node >=22.14 and React/react-dom >=19.2 <20. The verified package uses React 19.3, TypeScript 6.0.3 and pnpm 11.22.0.

## Exports

| Entry | Contract |
| --- | --- |
| `@ztd-me/frontend` | Server-safe types, defaults, normalization, cookies, locale negotiation and legacy extraction; no DOM or React runtime import |
| `@ztd-me/frontend/client` | `FrontendProvider`, `useFrontendPreferences`, `PublicShell`, `ApplicationShell`, `Appbar`, `SiteFooter`, `AppearanceMenu`, `LocaleSelect`; preserved `use client` boundary |
| `@ztd-me/frontend/styles.css` | Compiled namespaced CSS and bundled Inter Latin/Latin Extended variable fonts; no Tailwind scan or node_modules `@source` requirement |

ESM only. Client callbacks, custom links, identity and portal containers belong in a client adapter, not a serialized server payload. Use one provider per document; its initial preferences and policy identify that mounted document.

```tsx
// Server: use a trusted deployment configuration, not an arbitrary forwarded Host.
import {
  createPreferencePolicy,
  frontendRootAttributes,
  resolveInitialPreferences,
} from '@ztd-me/frontend'

const policy = createPreferencePolicy({
  environment: 'production',
  namespace: 'website',
  hostname: 'ztd.me',
  protocol: 'https:',
})
const initialPreferences = resolveInitialPreferences({
  policy,
  cookieHeader,
  acceptLanguage,
})
// Render <html {...frontendRootAttributes(initialPreferences)}> on the server.
// Pass exactly this snapshot and policy to the client adapter.
```

```tsx
'use client'

import { FrontendProvider, PublicShell } from '@ztd-me/frontend/client'
import '@ztd-me/frontend/styles.css'

<FrontendProvider initialPreferences={initialPreferences} policy={policy}>
  <PublicShell
    brand={{ label: 'zeithrold.', homeHref: '/' }}
    repositoryUrl="https://github.com/zeithrold/website"
  >
    {existingContent}
  </PublicShell>
</FrontendProvider>
```

`brand` accepts `label`, a local `homeHref` and optional `mark`. Optional shell slots are `projectActions` and `identity`; `mainId` defaults to `ztd-main`, with a skip link and focusable main. `linkComponent` provides the consumer's routing link. Public shells have no navigation slot and add no cross-site menu. `ApplicationShell` adds `businessNavigation`, `contextSidebar` and `serviceNotice`; Memory determines their contents, destinations and responsive behavior. Slots must not create a second main landmark.

Footer content is fixed: **© Zeithrold**, the current project's GitHub repository, **hello@ztd.me**. Pass the actual project repository, not the owner profile or tools repository. No year, home/blog/showcase menu or invented service links are added.

`useFrontendPreferences()` returns `preferences`, `resolvedMode`, `persistence` and setters `setMode`, `setPalette`, `setLocale`. The provider accepts `onPreferencesChange` for an existing project translation/theme bridge, `onPersistenceError` for meaningful recovery feedback, and `portalContainer` for fullscreen overlays. Optional `styleNonce` passes the current document's style nonce to Radix's upstream scroll-lock injector and Select viewport styles. Hooks require the same provider instance. Authentication remains an opaque project-owned UI slot, never a preference field.

## Appearance and CSS

Modes: `system`, `light`, `dark`. Palettes: `neutral`, `terracotta`, `moss`, `ocean`, `plum`, `graphite`. Locales: `en`, `zh-CN`, with complete shared control labels and native language names.

CSS uses root attributes `data-frontend-mode` and `data-frontend-palette`, with `lang`. System colors resolve through CSS media queries before hydration, without an inline prepaint script. Radix uses inline style attributes for positioning/hidden controls and injects scroll-lock style elements. A consumer CSP must account for these: the packed fixture keeps `script-src 'self'`, permits style attributes, and allows style elements only from self or a per-response nonce supplied as `styleNonce`. This does not promise compatibility with `style-src-attr 'none'`; the package does not change any consumer CSP. Client media changes update `resolvedMode`; an explicit mode stays fixed. The package does not add `.dark` or the old `data-palette` attribute. Consumers bridge their own business tokens or legacy attributes through the hook; shared root tokens are `--ztd-background`, `--ztd-foreground`, `--ztd-surface`, `--ztd-muted`, `--ztd-muted-foreground`, `--ztd-border`, `--ztd-accent`, `--ztd-focus`, `--ztd-action`, `--ztd-action-foreground`, and `--ztd-color-scheme`.

Chrome is at most 1280px wide, 72px high on desktop and 64px below 1024px, with 16/24/32px gutters and 44px control targets. Body width, content, typography and layout remain project-owned. Import CSS once. Include its actual declaration file as a `tokenFiles` source in the existing CSS checker when business CSS references its tokens; do not exempt arbitrary unknown variables. Font licenses ship with the assets; CJK uses system fallbacks.

## Preferences and boundaries

Version 1 contains exactly `{ version: 1, mode, palette, locale }`. No auth, account, business data, workspace IDs, drafts or clock settings enter shared persistence. Unknown keys are discarded. Enums are validated; malformed/oversized values and duplicate same-name cookies are invalid. Values are limited to 1024 encoded bytes. Unknown future versions are ignored for rendering and not automatically rewritten.

`createPreferencePolicy` enables `ztd.frontend.v1; Domain=ztd.me; Path=/; SameSite=Lax; Secure; Max-Age=31536000` only for an explicit HTTPS production deployment on ztd.me or its subdomains. Every sibling subdomain can read/overwrite that shared cookie: it is untrusted UI input and never an authorization source. Cookie policy does not modify existing auth cookies.

Development and preview use host-only `ztd.frontend.<environment>.<namespace>.v1`. HTTP localhost omits Secure; HTTPS preview retains it. Cookies are not port-isolated, so the project namespace keeps local consumers apart. A preview on a ztd.me subdomain ignores and does not write the production preference key.

SSR reads the policy's cookie and negotiates supported languages from ordered Accept-Language quality values, falling back to English. It renders the same initial snapshot as the first client render. Existing valid selections take priority over new defaults. Cookie rejection leaves controls usable in memory, with `persistence='unavailable'`; reload may restore the server defaults. Local storage is an optional notification mirror, never the server's preference source.

A valid shared cookie wins over stale legacy storage. When the cookie is missing, extract only valid UI fields from `ztd.home.v1`, `showcase.clock.v1`, or Memory's old `locale` cookie. Old keys are not deleted and clock/business fields remain untouched. Legacy local storage cannot influence the first SSR paint; migration runs after hydration.

Same-origin tabs can observe mirror notifications. Different subdomains do not receive localStorage/BroadcastChannel events; active pages re-read cookies on focus/visibility recovery. Sharing is eventual, with last-write-wins values, not atomic merging or instantaneous cross-origin broadcasting. Consumers must verify locale-sensitive CDN/Worker cache variations and their own business hydration/metadata behavior.

## Verification and staging

```sh
pnpm install --frozen-lockfile
pnpm run check
```

Checks retain strict ESLint, CSS, full TypeScript declaration checking, unit boundary tests, and independent packed-consumer installation/types/build/browser evidence under `.artifacts/`. Browser fixtures use React SSR/Vite with synthetic data and intercepted local subdomain origins; they do not contact production, verify authentication or certify all consuming frameworks.

The Select declaration compatibility patch is documented in [Radix compatibility](docs/radix-compatibility.md). The package retains the original Radix Select interaction and unmodified runtime. Consumers of this package's public declarations do not need the development patch.

`publish-frontend-shell.yml` automatically verifies and stages the immutable main artifact with the existing stage-only `NPM_TOKEN` route. It has no manual dispatch or confirmation fork and performs no npm promotion. The owner merges and approves the staged version using npm 2FA. Final consumer integration requires a fresh registry installation of the promoted exact version: run `node scripts/registry-smoke.mjs 0.1.0` after owner promotion. See [verification scope](docs/verification.md) for evidence and limitations.

Self-owned code is MIT; upstream dependencies and font licenses remain intact. See [third-party notices](THIRD_PARTY_NOTICES.md).
