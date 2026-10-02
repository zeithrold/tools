import type { PreferenceSnapshot, PreferenceStore } from './store-types.js'
import type { FrontendPreferences, PreferencePolicy } from './types.js'
import { persistBrowserPreferences, readBrowserPreferences, readLegacyPreference } from './browser-environment.js'
import { migrateLegacyPreferences } from './migration.js'
import { normalizePreferences, serializePreferences } from './preferences.js'

export function createPreferenceStore(initial: FrontendPreferences, policy: PreferencePolicy): PreferenceStore {
  const preferences = normalizePreferences(initial)
  const server: PreferenceSnapshot = {
    preferences,
    resolvedMode: preferences.mode === 'dark' ? 'dark' : 'light',
    persistence: 'unchanged',
  }
  let snapshot = server
  let systemDark = false
  const listeners = new Set<() => void>()
  function commit(next: FrontendPreferences, persistence = snapshot.persistence): void {
    const systemMode = systemDark ? 'dark' : 'light'
    const resolvedMode = next.mode === 'system' ? systemMode : next.mode
    snapshot = { preferences: next, resolvedMode, persistence }
    for (const listener of listeners) {
      listener()
    }
  }
  function change(next: FrontendPreferences): void {
    const normalized = normalizePreferences(next)
    const saved = persistBrowserPreferences(normalized, policy)
    commit(normalized, saved ? 'saved' : 'unavailable')
  }
  function restore(): void {
    restoreFromCookie(snapshot, policy, commit)
  }
  const actions = {
    setMode: (mode: FrontendPreferences['mode']) => change({ ...snapshot.preferences, mode }),
    setPalette: (palette: FrontendPreferences['palette']) => change({ ...snapshot.preferences, palette }),
    setLocale: (locale: FrontendPreferences['locale']) => change({ ...snapshot.preferences, locale }),
  }
  function connect(): () => void {
    return connectBrowser(policy, snapshot.preferences, {
      commit,
      restore,
      change,
      setSystem: (dark) => {
        systemDark = dark
        commit(snapshot.preferences)
      },
    })
  }
  return {
    ...actions,
    getSnapshot: () => snapshot,
    getServerSnapshot: () => server,
    subscribe: (listener) => {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
    connect,
  }
}
interface BrowserActions {
  commit: (preferences: FrontendPreferences) => void
  restore: () => void
  change: (preferences: FrontendPreferences) => void
  setSystem: (dark: boolean) => void
}
function connectBrowser(
  policy: PreferencePolicy,
  initial: FrontendPreferences,
  actions: BrowserActions,
): () => void {
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const refreshMedia = (): void => {
    actions.setSystem(media.matches)
    actions.restore()
  }
  actions.setSystem(media.matches)
  actions.commit(initial)
  const read = readBrowserPreferences(policy)
  if (read.status === 'missing') {
    const migrated = migrateLegacyPreferences(readLegacyPreference(policy), policy.namespace, initial)
    if (serializePreferences(migrated) !== serializePreferences(initial)) {
      actions.change(migrated)
    }
  }
  actions.restore()
  const onStorage = (event: StorageEvent): void => {
    if (event.key === policy.name) {
      actions.restore()
    }
  }
  const onVisible = (): void => {
    if (document.visibilityState === 'visible') {
      actions.restore()
    }
  }
  window.addEventListener('focus', actions.restore)
  window.addEventListener('storage', onStorage)
  document.addEventListener('visibilitychange', onVisible)
  media.addEventListener('change', refreshMedia)
  return () => {
    window.removeEventListener('focus', actions.restore)
    window.removeEventListener('storage', onStorage)
    document.removeEventListener('visibilitychange', onVisible)
    media.removeEventListener('change', refreshMedia)
  }
}

function restoreFromCookie(
  snapshot: PreferenceSnapshot,
  policy: PreferencePolicy,
  commit: (preferences: FrontendPreferences, persistence: PreferenceSnapshot['persistence']) => void,
): void {
  const read = readBrowserPreferences(policy)
  if (read.status === 'unavailable') {
    commit(snapshot.preferences, 'unavailable')
    return
  }
  if (read.preferences === null) {
    return
  }
  const recovered = snapshot.persistence === 'unavailable'
  const changed = serializePreferences(read.preferences) !== serializePreferences(snapshot.preferences)
  if (changed || recovered) {
    commit(read.preferences, recovered ? 'saved' : snapshot.persistence)
  }
}
