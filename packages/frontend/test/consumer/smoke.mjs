import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'
import { DEFAULT_PREFERENCES, resolveInitialPreferences } from '@ztd-me/frontend'
import * as frontend from '@ztd-me/frontend'
import { PublicShell, FrontendProvider } from '@ztd-me/frontend/client'

assert.equal(typeof PublicShell, 'function')
assert.equal(typeof FrontendProvider, 'function')
assert.equal(typeof resolveInitialPreferences, 'function')
assert.equal(DEFAULT_PREFERENCES.mode, 'system')
assert.equal('migrateLegacyPreferences' in frontend, false)
assert.equal('LEGACY_STORAGE_KEYS' in frontend, false)
const require = createRequire(import.meta.url)
const css = require.resolve('@ztd-me/frontend/styles.css')
const source = await readFile(css, 'utf8')
assert.ok(source.includes('.ztd-appbar'))
const fonts = [...source.matchAll(/url\((?:['"])?\.\/([^)'"\s]+)/gu)]
assert.equal(fonts.length, 0)
assert.ok(source.includes('https://fonts.googleapis.com/css2?family=Noto+Sans'))
assert.ok(source.includes('family=Noto+Color+Emoji&display=swap'))
assert.equal(source.includes('family=Noto+Emoji'), false)
for (const family of [
  'SANS',
  'SANS-SC',
  'SANS-JP',
  'SANS-KR',
  'EMOJI',
  'COLOR-EMOJI',
]) {
  const license = path.join(path.dirname(css), `assets/NOTO-${family}-OFL.txt`)
  assert.ok((await readFile(license, 'utf8')).includes('SIL OPEN FONT LICENSE Version 1.1'))
}
assert.ok((await readFile(path.join(path.dirname(css), 'assets/SHADCN-MIT.txt'), 'utf8')).includes('MIT'))
const client = await readFile(new URL(import.meta.resolve('@ztd-me/frontend/client')), 'utf8')
assert.ok(client.startsWith("'use client'"))
