import type { FrontendPreferences, PreferenceCookieResult, PreferencePolicy } from './types.js'
import { preferenceCookie, readPreferenceCookie } from './cookies.js'
import { LEGACY_STORAGE_KEYS } from './migration.js'
import { serializePreferences } from './preferences.js'

function browserCookieHeader(): string | null {
  try {
    return document.cookie
  }
  catch {
    return null
  }
}
export function readBrowserPreferences(policy: PreferencePolicy):
  PreferenceCookieResult | { status: 'unavailable', preferences: null } {
  const header = browserCookieHeader()
  return header === null
    ? { status: 'unavailable', preferences: null }
    : readPreferenceCookie(header, policy)
}
export function readLegacyPreference(policy: PreferencePolicy): unknown {
  if (policy.namespace === 'memory') {
    const values = browserCookieHeader()?.split(';').map(value => value.trim())
    return values?.find(value => value.startsWith('locale='))?.slice(7) ?? null
  }
  try {
    const stored = localStorage.getItem(LEGACY_STORAGE_KEYS[policy.namespace])
    return stored !== null && stored !== '' ? JSON.parse(stored) as unknown : null
  }
  catch {
    return null
  }
}
export function persistBrowserPreferences(preferences: FrontendPreferences, policy: PreferencePolicy): boolean {
  try {
    document.cookie = preferenceCookie(preferences, policy)
    const read = readBrowserPreferences(policy)
    if (read.preferences === null || serializePreferences(read.preferences) !== serializePreferences(preferences)) {
      return false
    }
    try {
      localStorage.setItem(policy.name, serializePreferences(preferences))
    }
    catch {
      // Cookie persistence still works when the optional same-origin notification mirror is unavailable.
    }
    return true
  }
  catch {
    return false
  }
}
