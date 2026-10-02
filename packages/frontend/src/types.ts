export const MODES = [
  'system',
  'light',
  'dark',
] as const
export const LOCALES = ['en', 'zh-CN'] as const
export const PALETTES = [
  'neutral',
  'terracotta',
  'moss',
  'ocean',
  'plum',
  'graphite',
] as const

export type Mode = (typeof MODES)[number]
export type Locale = (typeof LOCALES)[number]
export type Palette = (typeof PALETTES)[number]
export type Project = 'website' | 'showcase' | 'memory'
export interface FrontendPreferences {
  version: 1
  mode: Mode
  palette: Palette
  locale: Locale
}
export interface PreferencePolicyOptions {
  environment: 'production' | 'preview' | 'development'
  namespace: Project
  hostname: string
  protocol: 'http:' | 'https:'
}
export interface PreferencePolicy {
  name: string
  domain?: 'ztd.me'
  secure: boolean
  namespace: Project
}
export interface PreferenceCookieResult {
  status: 'valid' | 'missing' | 'invalid' | 'future'
  preferences: FrontendPreferences | null
}
export interface InitialPreferenceOptions {
  policy: PreferencePolicy
  cookieHeader?: string | undefined
  acceptLanguage?: string | undefined
}
