import type {
  FrontendPreferences,
  InitialPreferenceOptions,
  PreferenceCookieResult,
  PreferencePolicy,
  PreferencePolicyOptions,
} from './types.js'
import { negotiateLocale } from './locale.js'
import { normalizePreferences, preferenceRecord, serializePreferences } from './preferences.js'

export const MAX_PREFERENCE_COOKIE_BYTES = 1024

export function createPreferencePolicy(options: PreferencePolicyOptions): PreferencePolicy {
  const trustedHost = options.hostname === 'ztd.me' || options.hostname.endsWith('.ztd.me')
  const shared = options.environment === 'production' && trustedHost && options.protocol === 'https:'
  const localName = `ztd.frontend.${options.environment}.${options.namespace}.v1`
  return {
    name: shared ? 'ztd.frontend.v1' : localName,
    ...(shared ? { domain: 'ztd.me' as const } : {}),
    secure: options.protocol === 'https:',
    namespace: options.namespace,
  }
}
function decodeCookie(value: string): PreferenceCookieResult {
  if (value.length > MAX_PREFERENCE_COOKIE_BYTES) {
    return { status: 'invalid', preferences: null }
  }
  try {
    const decoded: unknown = JSON.parse(decodeURIComponent(value))
    const record = preferenceRecord(decoded)
    if (record === null || typeof record.version !== 'number') {
      return { status: 'invalid', preferences: null }
    }
    if (record.version > 1) {
      return { status: 'future', preferences: null }
    }
    if (record.version !== 1) {
      return { status: 'invalid', preferences: null }
    }
    return { status: 'valid', preferences: normalizePreferences(record) }
  }
  catch {
    return { status: 'invalid', preferences: null }
  }
}
export function readPreferenceCookie(header: string, policy: PreferencePolicy): PreferenceCookieResult {
  const values = header.split(';').map(part => part.trim()).filter(part => part.startsWith(`${policy.name}=`))
  if (values.length === 0) {
    return { status: 'missing', preferences: null }
  }
  if (values.length !== 1) {
    return { status: 'invalid', preferences: null }
  }
  return decodeCookie(values[0]?.slice(policy.name.length + 1) ?? '')
}
export function preferenceCookie(preferences: FrontendPreferences, policy: PreferencePolicy): string {
  const domain = policy.domain !== undefined ? '; Domain=ztd.me' : ''
  const secure = policy.secure ? '; Secure' : ''
  return `${policy.name}=${serializePreferences(preferences)}; Path=/; SameSite=Lax; Max-Age=31536000${domain}${secure}`
}
export function resolveInitialPreferences(options: InitialPreferenceOptions): FrontendPreferences {
  const cookie = readPreferenceCookie(options.cookieHeader ?? '', options.policy)
  return cookie.preferences ?? normalizePreferences(null, negotiateLocale(options.acceptLanguage))
}
export function frontendRootAttributes(preferences: FrontendPreferences): {
  'lang': string
  'data-frontend-mode': string
  'data-frontend-palette': string
} {
  const normalized = normalizePreferences(preferences)
  return {
    'lang': normalized.locale,
    'data-frontend-mode': normalized.mode,
    'data-frontend-palette': normalized.palette,
  }
}
