import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  createPreferencePolicy,
  DEFAULT_PREFERENCES,
  frontendRootAttributes,
  MAX_PREFERENCE_COOKIE_BYTES,
  negotiateLocale,
  normalizePreferences,
  PALETTES,
  preferenceCookie,
  readPreferenceCookie,
  resolveInitialPreferences,
  serializePreferences,
} from '@ztd-me/frontend'

const policy = createPreferencePolicy({
  name: 'atelier.ui.v1',
  domain: 'atelier.example',
  mirrorKey: 'atelier.ui.events',
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

test('default persistence is host-only and secure without a storage mirror', () => {
  const defaults = createPreferencePolicy()
  assert.deepEqual(defaults, { name: 'frontend.preferences.v1', secure: true })
  const cookie = preferenceCookie(selected, defaults)
  assert.match(cookie, /; Path=\/; SameSite=Lax; Max-Age=31536000; Secure$/u)
  assert.ok(!cookie.includes('Domain='))
  const local = createPreferencePolicy({ name: 'atelier.local.v1', secure: false })
  assert.ok(!preferenceCookie(selected, local).includes('; Secure'))
})

test('domain, cookie name and optional mirror are independent consumer choices', () => {
  assert.equal(policy.domain, 'atelier.example')
  assert.equal(policy.mirrorKey, 'atelier.ui.events')
  assert.match(preferenceCookie(selected, policy), /Domain=atelier\.example; Secure/u)
  const existing = createPreferencePolicy({ name: 'ztd.frontend.v1', domain: 'ztd.me' })
  const header = `${existing.name}=${serializePreferences(selected)}`
  assert.deepEqual(readPreferenceCookie(header, existing).preferences, selected)
  assert.equal(createPreferencePolicy({ domain: 'CUSTOMER.EXAMPLE' }).domain, 'customer.example')
})

test('policy rejects injection, invalid cookie prefixes and unsafe domain options', () => {
  const invalid = [
    { name: 'ui; Path=/auth' },
    { name: '' },
    { name: null },
    { name: 'x'.repeat(129) },
    { domain: 'https://atelier.example' },
    { domain: 'atelier.example; HttpOnly' },
    { domain: 'atelier.example:443' },
    { domain: '.atelier.example' },
    { domain: 'localhost' },
    { domain: '127.0.0.1' },
    { domain: 'atelier.example', secure: false },
    { secure: null },
    { secure: 'false' },
    { name: '__Secure-ui', secure: false },
    { name: '__Host-ui', domain: 'atelier.example' },
    { mirrorKey: '' },
    { mirrorKey: 'auth\nkey' },
    { namespace: 'arbitrary' },
    null,
  ]
  for (const options of invalid) {
    assert.throws(() => createPreferencePolicy(options), TypeError)
  }
  assert.equal(createPreferencePolicy({ name: '__Host-ui' }).secure, true)
  assert.throws(() => preferenceCookie(selected, { name: 'bad=value', secure: true }), TypeError)
  assert.throws(() => readPreferenceCookie('', { name: 'bad=value', secure: true }), TypeError)
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
