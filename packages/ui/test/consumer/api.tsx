import { createI18nAdapter } from '@ztd-me/ui'
import type {} from '@ztd-me/ui/styles.css'
import type { FrontendPreferences, PreferencePolicy, ShellProps } from '@ztd-me/ui'
import { FrontendProvider, PublicShell, ApplicationShell, useFrontendPreferences } from '@ztd-me/ui/client'
// @ts-expect-error The package no longer enumerates its consuming projects.
export type { Project } from '@ztd-me/ui'

const initialPreferences: FrontendPreferences = { version: 1, mode: 'system', palette: 'neutral', locale: 'en' }
const policy: PreferencePolicy = { name: 'studio.preferences', secure: false, mirrorKey: 'studio.events' }
const shell: ShellProps = {
  brand: { label: 'Harbor Studio', homeHref: 'https://harbor.example/' },
  footer: { copyright: '© Harbor Studio', links: [{ label: 'Source', href: 'https://codeberg.org/example/studio' }] },
  children: null,
}
export const publicShell = <PublicShell {...shell} />
export const applicationShell = <ApplicationShell {...shell} businessNavigation={<nav />} />
export const provider = <FrontendProvider initialPreferences={initialPreferences} policy={policy}>{publicShell}</FrontendProvider>
// @ts-expect-error The shared preference schema excludes authentication.
export const sensitivePreferences: FrontendPreferences = { ...initialPreferences, authToken: 'forbidden' }
// @ts-expect-error The public shell has no navigation ownership.
export const publicNavigation = <PublicShell {...shell} businessNavigation={<nav />} />
// @ts-expect-error Invalid palette values cannot enter the typed API.
export const invalidPalette: FrontendPreferences = { ...initialPreferences, palette: 'brand-new' }
// @ts-expect-error Project-specific policy is no longer part of the package API.
export const projectPolicy: PreferencePolicy = { ...policy, namespace: 'arbitrary-project' }
// @ts-expect-error Repository identity is supplied through generic footer links.
export const fixedRepository = <PublicShell {...shell} repositoryUrl="https://codeberg.org/example/studio" />
export function setters(): void {
  const state = useFrontendPreferences()
  state.setMode('system')
  state.setPalette('graphite')
  state.setLocale('zh-CN')
  // @ts-expect-error Only supported locales are accepted.
  state.setLocale('zh-TW')
}

const catalog = { common: { retry: 'Retry' }, editor: { save: 'Save' } }
const adapter = createI18nAdapter({
  resources: { en: catalog, 'zh-CN': catalog },
  instance: { translate: request => request.key },
})
adapter.t('en', 'editor', 'save')
// @ts-expect-error Unknown namespaces must be rejected by the injected typed adapter.
adapter.t('en', 'absent', 'save')
// @ts-expect-error Keys belong to their declared namespace rather than a global string dictionary.
adapter.t('en', 'common', 'save')
