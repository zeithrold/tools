# Generic API upgrade to 0.2

Version 0.2 removes consumer-specific identity, deployment decisions and legacy mappings from the reusable package. Its version-1 preference schema, server root attributes, mode/palette/locale values, provider hooks, generic slots and CSS entry remain compatible.

## Cookie policy

Replace the old `environment`, `namespace`, `hostname` and `protocol` options with explicit `name`, optional `domain`, `secure` and optional `mirrorKey`. The removed `Project` type has no replacement allowlist. Any consumer may configure its own policy.

```ts
import { createPreferencePolicy } from '@ztd-me/frontend'

const policy = createPreferencePolicy({
  name: 'harbor.ui.v1',
  domain: 'harbor.example',
  secure: true,
  mirrorKey: 'harbor.ui.notification',
})
```

Choose these values in the project's trusted deployment configuration. To preserve existing version-1 selections, keep the existing cookie name and domain. This is explicit policy configuration, not storage migration. The package never reads old keys or unrelated cookies, and never removes business/authentication data. Do not reintroduce the rejected legacy mappings in consumer adapters.

No-argument defaults are a secure, host-only `frontend.preferences.v1` cookie without a storage mirror. HTTP development needs explicit `secure: false`. Previews that should stay isolated need their own name and no domain. An optional mirror writes change notifications and never reads persisted values; omit `mirrorKey` to avoid localStorage entirely.

The `LEGACY_STORAGE_KEYS` and `migrateLegacyPreferences` exports are removed, with no replacement.

## Consumer identity

Replace the removed `repositoryUrl` shell prop with optional `footer`:

```tsx
<PublicShell
  brand={{ label: 'Harbor Studio', homeHref: 'https://harbor.example/' }}
  footer={{
    copyright: '© Harbor Studio',
    links: [{ label: 'Source', href: 'https://codeberg.org/example/studio' }],
  }}
>
  {children}
</PublicShell>
```

`FooterLink` and `SiteFooterProps` are exported as types from the root and client entries. `SiteFooter` is exported from the client entry. Both shells omit the footer when no configuration is supplied; a standalone footer accepts the same optional fields. HTTP/HTTPS and relative links are valid, with `mailto:` also supported in footer links. Consumer identity has no fixed default or repository-host restriction.

Validate an exact public npm version using the registry smoke script before final consumer dependency
locks. For the selected source delivery path, follow the [registry installation and reviewed-update
recipe](../../../registry/README.md), keeping the provider, hooks and stylesheet together.
