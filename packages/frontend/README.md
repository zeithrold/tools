# @ztd-me/frontend

The selected successor is **`@ztd-me/ui` source delivery**, not a renamed npm UI package. See the
[local registry candidate](../../registry/README.md). Existing npm installs remain available until their
source replacements are reviewed and validated; installed registry source has no frontend package dependency.
This directory temporarily hosts the canonical source and verification harness during that transition.

Generic shadcn new-york/Radix chrome for React 19 projects. The default is **Neutral + System**; mode and palette are independent. All six palettes support light and dark. Brand, footer content, deployment/storage policy, business navigation, identity and application state stay with the consumer.

```sh
pnpm add @ztd-me/frontend@0.2.0
```

Version 0.2.0 is published and verified. Requires Node >=22.14 and React/react-dom >=19.2 <20. The verified package uses React 19.3, TypeScript 6.0.3 and pnpm 11.22.0.

## Exports

| Entry | Contract |
| --- | --- |
| `@ztd-me/frontend` | Server-safe types, defaults, normalization, cookie policy/serialization and locale negotiation; no DOM or React runtime import |
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
  name: 'harbor.ui.v1',
  domain: 'harbor.example', // Optional: omit for a host-only cookie.
  secure: true,
  mirrorKey: 'harbor.ui.notification', // Optional same-origin change notification.
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
    brand={{ label: 'Harbor Studio', homeHref: '/' }}
    footer={{
      copyright: '© Harbor Studio',
      links: [
        { label: 'Source', href: 'https://codeberg.org/example/studio' },
        { label: 'Contact', href: 'mailto:support@harbor.example' },
      ],
    }}
  >
    {existingContent}
  </PublicShell>
</FrontendProvider>
```

`brand` accepts `label`, a relative or absolute HTTP/HTTPS `homeHref` and optional `mark`. Optional shell slots are `projectActions` and `identity`; `mainId` defaults to `ztd-main`, with a skip link and focusable main. `linkComponent` provides the consumer's routing link. Public shells have no navigation slot. `ApplicationShell` adds `businessNavigation`, `contextSidebar` and `serviceNotice`; the consumer determines their contents, destinations and responsive behavior. Slots must not create a second main landmark.

Footer configuration is optional. Without `footer`, the shell renders no footer. `SiteFooterProps` contains optional `copyright: ReactNode` and `links: readonly FooterLink[]`; each link has `label`, `href` and optional `ariaLabel`. Relative paths, HTTP/HTTPS and `mailto:` links are supported without a repository-host restriction. Unsafe schemes, protocol-relative URLs, control characters and backslashes throw. The package supplies no brand, contact address, repository or copyright text.

`useFrontendPreferences()` returns `preferences`, `resolvedMode`, `persistence` and setters `setMode`, `setPalette`, `setLocale`. The provider accepts `onPreferencesChange` for an existing project translation/theme bridge, `onPersistenceError` for meaningful recovery feedback, and `portalContainer` for fullscreen overlays. Optional `styleNonce` passes the current document's style nonce to Radix's upstream scroll-lock injector and Select viewport styles. Hooks require the same provider instance. Authentication remains an opaque project-owned UI slot, never a preference field.

## Appearance and CSS

Modes: `system`, `light`, `dark`. Palettes: `neutral`, `terracotta`, `moss`, `ocean`, `plum`, `graphite`. Locales: `en`, `zh-CN`, with complete shared control labels and native language names.

CSS uses root attributes `data-frontend-mode` and `data-frontend-palette`, with `lang`. System colors resolve through CSS media queries before hydration, without an inline prepaint script. Radix uses inline style attributes for positioning/hidden controls and injects scroll-lock style elements. A consumer CSP must account for these: the packed fixture keeps `script-src 'self'`, permits style attributes, and allows style elements only from self or a per-response nonce supplied as `styleNonce`. This does not promise compatibility with `style-src-attr 'none'`; the package does not change any consumer CSP. Client media changes update `resolvedMode`; an explicit mode stays fixed. The package does not add `.dark` or the old `data-palette` attribute. Consumers bridge their own business tokens or legacy attributes through the hook; shared root tokens are `--ztd-background`, `--ztd-foreground`, `--ztd-surface`, `--ztd-muted`, `--ztd-muted-foreground`, `--ztd-border`, `--ztd-accent`, `--ztd-focus`, `--ztd-action`, `--ztd-action-foreground`, and `--ztd-color-scheme`.

Chrome is at most 1280px wide, 72px high on desktop and 64px below 1024px, with 16/24/32px gutters and 44px control targets. Body width, content, typography and layout remain project-owned. Import CSS once. Include its actual declaration file as a `tokenFiles` source in the existing CSS checker when business CSS references its tokens; do not exempt arbitrary unknown variables. Font licenses ship with the assets; CJK uses system fallbacks.

## Preferences and boundaries

Version 1 contains exactly `{ version: 1, mode, palette, locale }`. No auth, account, business data, workspace IDs, drafts or clock settings enter shared persistence. Unknown keys are discarded. Enums are validated; malformed/oversized values and duplicate same-name cookies are invalid. Values are limited to 1024 encoded bytes. Unknown future versions are ignored for rendering and not automatically rewritten.

`createPreferencePolicy({ name?, domain?, secure?, mirrorKey? })` accepts explicit consumer configuration and returns a frozen validated policy. The default cookie is `frontend.preferences.v1; Path=/; SameSite=Lax; Secure; Max-Age=31536000`, host-only, with no localStorage access. Cookie names must be nonempty tokens of at most 128 characters. An explicit domain must be a DNS name without a scheme, port, leading dot or IP address and requires Secure. Standard `__Secure-`/`__Host-` restrictions are enforced. Browsers still enforce host matching and public-suffix restrictions; the package does not infer a deployment or permit a domain based on a project name. Every sibling subdomain can read/overwrite an explicitly shared domain cookie: it is untrusted UI input and never an authorization source.

For HTTP development, explicitly pass `secure: false` and omit `domain`. Consumers choose separate cookie names for environments or applications that should remain isolated; cookies are not port-isolated. A preview requiring isolation must use its own host-only policy/name. There are no project enums, hostname branches, automatic domain decisions or fixed storage namespaces.

SSR reads the policy's cookie and negotiates supported languages from ordered Accept-Language quality values, falling back to English. It renders the same initial snapshot as the first client render. Existing valid selections take priority over new defaults. Throwing or rejected cookie reads/writes leave the SSR/current snapshot and controls usable in memory, with `persistence='unavailable'` and observable `onPersistenceError` feedback. Focus, visibility, storage and media refreshes use the same guarded reader. When reads recover, a valid persisted cookie becomes authoritative and clears unavailable status; a successful later write also restores `saved`. Missing/unreadable storage never resets an unsaved in-memory selection. Reload may restore the server defaults. Local storage is an optional notification mirror, never the server's preference source.

Only the explicitly configured cookie is read. There is no legacy extraction or migration, and no reading/deleting old localStorage keys or unrelated cookies. Missing cookies use negotiated/default preferences until the user makes a choice. Existing valid version-1 preferences stay compatible when a consumer supplies the same cookie name and domain. Removed mappings must not be relocated into consumer projects. See [the 0.2 API upgrade](docs/upgrade-0.2.md).

An explicit `mirrorKey` enables localStorage writes solely to notify same-origin tabs, which re-read the cookie on matching storage events. The key must be nonempty, at most 128 characters and free of control characters. Mirror values are never read and mirror failure does not invalidate a successful cookie write. Omit it to avoid all localStorage access. Different subdomains do not receive localStorage events; active pages re-read cookies on focus/visibility recovery. Sharing is eventual with last-write-wins values. Consumers must verify locale-sensitive CDN/Worker cache variations and their own business hydration/metadata behavior.

## Verification and staging

```sh
pnpm install --frozen-lockfile
pnpm run check
```

Checks retain strict ESLint, CSS, full TypeScript declaration checking, unit boundary tests, and independent packed-consumer installation/types/build/browser evidence under `.artifacts/`. Browser fixtures use React SSR/Vite with synthetic data and intercepted local subdomain origins; they do not contact production, verify authentication or certify all consuming frameworks.

The Select declaration compatibility patch is documented in [Radix compatibility](docs/radix-compatibility.md). The package retains the original Radix Select interaction and unmodified runtime. Consumers of this package's public declarations do not need the development patch.

`publish-frontend-shell.yml` automatically verifies and stages the immutable main artifact through the existing stage-only route. It has no manual dispatch and performs no npm promotion. The owner merges and approves the staged version using npm 2FA. The development CSS gate uses the published `@ztd-me/frontend-checks@0.1.1`, whose native scoping exception is limited to Tailwind `custom-variant`. Final consumer integration requires a fresh registry installation of the promoted exact version: run `node scripts/registry-smoke.mjs 0.2.0` after owner promotion. See [verification scope](docs/verification.md) for evidence and limitations.

Self-owned code is MIT; upstream dependencies and font licenses remain intact. See [third-party notices](THIRD_PARTY_NOTICES.md).
