import type { FrontendPreferences, Project } from './types.js'
import { isLocale, isMode, isPalette, preferenceRecord } from './preferences.js'

export const LEGACY_STORAGE_KEYS = { website: 'ztd.home.v1', showcase: 'showcase.clock.v1', memory: 'locale' } as const

export function migrateLegacyPreferences(
  value: unknown,
  project: Project,
  initial: FrontendPreferences,
): FrontendPreferences {
  if (project === 'memory') {
    return { ...initial, locale: isLocale(value) ? value : initial.locale }
  }
  const record = preferenceRecord(value)
  if (record === null) {
    return initial
  }
  return {
    version: 1,
    mode: isMode(record.theme) ? record.theme : initial.mode,
    palette: isPalette(record.palette) ? record.palette : initial.palette,
    locale: isLocale(record.locale) ? record.locale : initial.locale,
  }
}
