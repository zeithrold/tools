import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'
import { DEFAULT_PREFERENCES, resolveInitialPreferences } from '@ztd-me/ui'
import * as frontend from '@ztd-me/ui'
import { PublicShell, FrontendProvider, buttonVariants } from '@ztd-me/ui/client'

assert.equal(typeof PublicShell, 'function')
assert.equal(typeof FrontendProvider, 'function')
const override = buttonVariants({ size: 'sm', className: 'min-h-0 text-help' })
assert.ok(override.includes('min-h-0'))
assert.equal(override.split(' ').includes('min-h-9'), false)
assert.ok(override.includes('text-help'))
assert.equal(override.split(' ').includes('text-control'), false)
assert.equal(typeof resolveInitialPreferences, 'function')
assert.equal(DEFAULT_PREFERENCES.mode, 'system')
assert.equal('migrateLegacyPreferences' in frontend, false)
assert.equal('LEGACY_STORAGE_KEYS' in frontend, false)
const require = createRequire(import.meta.url)
const css = require.resolve('@ztd-me/ui/styles.css')
const source = await readFile(css, 'utf8')
assert.ok(source.includes('.ztd-frontend'))
assert.ok(source.includes('.border-b'))
assert.ok(source.includes('--ztd-background'))
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
const client = await readFile(new URL(import.meta.resolve('@ztd-me/ui/client')), 'utf8')
assert.ok(client.startsWith("'use client'"))
