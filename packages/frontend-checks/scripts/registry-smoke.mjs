import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import process from 'node:process'
import { consumerSmoke } from './consumer-smoke.mjs'

const [version, archive] = process.argv.slice(2)
if (!version || !/^\d+\.\d+\.\d+$/u.test(version) || version === '0.0.0' || !archive) {
  throw new Error('Usage: node scripts/registry-smoke.mjs <promoted-version> <verified-tarball>')
}
const metadata = JSON.parse(execFileSync('pnpm', [
  'view',
  `@ztd-me/frontend-checks@${version}`,
  '--json',
  '--registry=https://registry.npmjs.org/',
], { encoding: 'utf8' }))
assert.equal(metadata.name, '@ztd-me/frontend-checks')
assert.equal(metadata.version, version)
const integrity = `sha512-${createHash('sha512').update(await readFile(archive)).digest('base64')}`
assert.equal(metadata.dist.integrity, integrity, 'Registry tarball must match the reviewed CI artifact')
await consumerSmoke(version, `Registry @ztd-me/frontend-checks@${version}`)
