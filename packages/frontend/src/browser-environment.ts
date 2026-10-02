import type { FrontendPreferences, PreferencePolicy } from './types.js'
import { preferenceCookie, readPreferenceCookie } from './cookies.js'
import { LEGACY_STORAGE_KEYS } from './migration.js'
import { serializePreferences } from './preferences.js'

export function readLegacyPreference(policy: PreferencePolicy): unknown {
  if (policy.namespace === 'memory') {
    const values = document.cookie.split(';').map(value => value.trim())
    return values.find(value => value.startsWith('locale='))?.slice(7)
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
    const read = readPreferenceCookie(document.cookie, policy)
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
