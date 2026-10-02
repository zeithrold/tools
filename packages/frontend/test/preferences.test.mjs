import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  createPreferencePolicy,
  DEFAULT_PREFERENCES,
  frontendRootAttributes,
  MAX_PREFERENCE_COOKIE_BYTES,
  migrateLegacyPreferences,
  negotiateLocale,
  normalizePreferences,
  PALETTES,
  preferenceCookie,
  readPreferenceCookie,
  resolveInitialPreferences,
  serializePreferences,
} from '@ztd-me/frontend'

const policy = createPreferencePolicy({
  environment: 'production',
  namespace: 'website',
  hostname: 'ztd.me',
  protocol: 'https:',
})
const selected = { version: 1, mode: 'dark', palette: 'ocean', locale: 'zh-CN' }

test('default mode and palette are independent; every approved palette survives', () => {
  assert.deepEqual(normalizePreferences(null), DEFAULT_PREFERENCES)
  assert.equal(PALETTES.length, 6)
  for (const palette of PALETTES) {
    const value = normalizePreferences({ ...selected, palette })
    assert.equal(value.palette, palette)
    assert.equal(value.mode, 'dark')
  }
})

test('unknown inputs are reduced to the exact non-sensitive schema', () => {
  const input = { ...selected, auth: 'token', account: { id: 'user' }, timezone: 'secret-zone' }
  assert.deepEqual(normalizePreferences(input), selected)
  const invalid = { version: 1, mode: 'invalid', palette: 'x', locale: 'fr' }
  assert.deepEqual(normalizePreferences(invalid), DEFAULT_PREFERENCES)
  for (const value of [
    null,
    [],
    'invalid',
    { version: 2 },
  ]) {
    assert.deepEqual(normalizePreferences(value), DEFAULT_PREFERENCES)
  }
})

test('production sharing is restricted to trusted HTTPS ztd.me deployment', () => {
  assert.equal(policy.domain, 'ztd.me')
  assert.match(preferenceCookie(selected, policy), /Domain=ztd\.me; Secure/u)
  for (const hostname of [
    'ztd.me.evil.test',
    'localhost',
    '127.0.0.1',
  ]) {
    const options = { ...policy, environment: 'production', hostname, protocol: 'https:' }
    assert.equal(createPreferencePolicy(options).domain, undefined)
  }
  const preview = createPreferencePolicy({
    environment: 'preview',
    namespace: 'website',
    hostname: 'preview.ztd.me',
    protocol: 'https:',
  })
  const memory = createPreferencePolicy({
    environment: 'development',
    namespace: 'memory',
    hostname: 'localhost',
    protocol: 'http:',
  })
  assert.equal(preview.domain, undefined)
  assert.equal(preview.secure, true)
  assert.equal(memory.secure, false)
  assert.notEqual(preview.name, policy.name)
  assert.notEqual(preview.name, memory.name)
})

test('valid, duplicate, malformed and future cookie versions are explicit', () => {
  const value = `${policy.name}=${serializePreferences(selected)}`
  assert.deepEqual(readPreferenceCookie(`auth=opaque; ${value}`, policy).preferences, selected)
  assert.equal(readPreferenceCookie(`${value}; ${value}`, policy).status, 'invalid')
  for (const payload of [
    '%broken',
    encodeURIComponent('[]'),
    encodeURIComponent('{"version":0}'),
  ]) {
    assert.equal(readPreferenceCookie(`${policy.name}=${payload}`, policy).status, 'invalid')
  }
  const future = `${policy.name}=${encodeURIComponent('{"version":2}')}`
  assert.equal(readPreferenceCookie(future, policy).status, 'future')
  assert.equal(readPreferenceCookie('', policy).status, 'missing')
})

test('cookie encoded length boundary is inclusive and oversized values are rejected', () => {
  const prefix = '{"version":1,"padding":"'
  const suffix = '"}'
  const padding = MAX_PREFERENCE_COOKIE_BYTES - encodeURIComponent(prefix + suffix).length
  const exact = encodeURIComponent(prefix + 'x'.repeat(padding) + suffix)
  assert.equal(exact.length, MAX_PREFERENCE_COOKIE_BYTES)
  assert.equal(readPreferenceCookie(`${policy.name}=${exact}`, policy).status, 'valid')
  assert.equal(readPreferenceCookie(`${policy.name}=${exact}x`, policy).status, 'invalid')
})

test('SSR cookie and ordered quality negotiation produce a deterministic first snapshot', () => {
  const cookieHeader = `${policy.name}=${serializePreferences(selected)}`
  assert.deepEqual(resolveInitialPreferences({ policy, cookieHeader, acceptLanguage: 'en-US' }), selected)
  assert.equal(negotiateLocale('en-US;q=0.5,zh-CN;q=0.9'), 'zh-CN')
  assert.equal(negotiateLocale('zh-CN;q=0,en-US;q=0.5'), 'en')
  assert.equal(negotiateLocale('zh-CN;q=2,en;q=1'), 'en')
  assert.equal(negotiateLocale('en;q=0.2,fr;q=invalid,zh-CN;q=0.9'), 'zh-CN')
  assert.equal(negotiateLocale('en-US,zh-CN'), 'en')
  assert.equal(negotiateLocale('zh-TW,fr'), 'en')
  assert.equal(resolveInitialPreferences({ policy, acceptLanguage: 'zh-CN' }).locale, 'zh-CN')
  assert.deepEqual(frontendRootAttributes(selected), {
    'lang': 'zh-CN',
    'data-frontend-mode': 'dark',
    'data-frontend-palette': 'ocean',
  })
})

test('legacy extraction preserves valid UI choices and excludes business data', () => {
  const legacy = { theme: 'dark', palette: 'moss', locale: 'zh-CN', timezone: 'UTC', seconds: true, auth: 'token' }
  assert.deepEqual(migrateLegacyPreferences(legacy, 'showcase', DEFAULT_PREFERENCES), {
    version: 1,
    mode: 'dark',
    palette: 'moss',
    locale: 'zh-CN',
  })
  assert.deepEqual(legacy.timezone, 'UTC')
  assert.equal(migrateLegacyPreferences('zh-CN', 'memory', DEFAULT_PREFERENCES).locale, 'zh-CN')
})
