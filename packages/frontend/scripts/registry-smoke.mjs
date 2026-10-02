import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile, writeFile } from 'node:fs/promises'
import process from 'node:process'
import { prepareConsumer } from './prepare-consumer.mjs'

const packageInfo = JSON.parse(await readFile('package.json', 'utf8'))
const version = process.argv[2]
assert.equal(version, packageInfo.version, 'Verify this source revision against its exact promoted version')
const spec = `${packageInfo.name}@${version}`
const registry = 'https://registry.npmjs.org/'
const metadata = JSON.parse(execFileSync('pnpm', [
  'view',
  spec,
  '--registry',
  registry,
  '--json',
], { encoding: 'utf8' }))
assert.equal(metadata.version, version)
assert.ok(metadata.dist?.integrity, 'A stage placeholder is insufficient public-release evidence')
const consumer = await prepareConsumer(version)
execFileSync('pnpm', [
  'exec',
  'playwright',
  'test',
], { stdio: 'inherit', env: { ...process.env, ZTD_FRONTEND_CONSUMER: consumer } })
await writeFile(`.artifacts/registry-${version}.json`, JSON.stringify({
  spec,
  registry,
  integrity: metadata.dist.integrity,
  consumer,
  status: 'passed',
}, null, 2))
