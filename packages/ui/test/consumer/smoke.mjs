import assert from 'node:assert/strict'
import { readFile, readdir } from 'node:fs/promises'
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
// Font software stays with Google; the package ships one original MIT notice per copied or patched
// dependency and no OFL copy or font binary of its own.
const assets = path.join(path.dirname(css), 'assets')
assert.deepEqual((await readdir(assets)).sort(), [
  'RADIX-MIT.txt',
  'REACT-REMOVE-SCROLL-BAR-MIT.txt',
  'SHADCN-MIT.txt',
  'VAUL-MIT.txt',
])
for (const notice of await readdir(assets)) {
  assert.ok((await readFile(path.join(assets, notice), 'utf8')).includes('MIT License'), notice)
}
const client = await readFile(new URL(import.meta.resolve('@ztd-me/ui/client')), 'utf8')
assert.ok(client.startsWith("'use client'"))
