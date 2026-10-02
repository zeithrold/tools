import type {} from '@ztd-me/frontend/styles.css'
import type { FrontendPreferences, PreferencePolicy, ShellProps } from '@ztd-me/frontend'
import { FrontendProvider, PublicShell, ApplicationShell, useFrontendPreferences } from '@ztd-me/frontend/client'

const initialPreferences: FrontendPreferences = { version: 1, mode: 'system', palette: 'neutral', locale: 'en' }
const policy: PreferencePolicy = { name: 'fixture', secure: false, namespace: 'website' }
const shell: ShellProps = {
  brand: { label: 'Fixture', homeHref: '/' }, repositoryUrl: 'https://github.com/zeithrold/website', children: null,
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
export function setters(): void {
  const state = useFrontendPreferences()
  state.setMode('system')
  state.setPalette('graphite')
  state.setLocale('zh-CN')
  // @ts-expect-error Only supported locales are accepted.
  state.setLocale('zh-TW')
}
