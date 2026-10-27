import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdtemp, readFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { assertPackedLicense } from '../../../scripts/license-coverage.mjs'
import { consumerSmoke } from './consumer-smoke.mjs'

const options = process.argv.slice(2)
const unknown = options.filter(option => option !== '--static')
if (unknown.length > 0) {
  throw new Error(`Unknown option(s): ${unknown.join(' ')}; --static keeps the packed checks without browser evidence`)
}
const browser = !options.includes('--static')

const directory = await mkdtemp(path.join(os.tmpdir(), 'ztd-frontend-pack-'))
const { version } = JSON.parse(await readFile('package.json', 'utf8'))
execFileSync('pnpm', [
  'pack',
  '--pack-destination',
  directory,
], { stdio: 'inherit' })
const archive = path.join(directory, `ztd-me-frontend-checks-${version}.tgz`)
await assertPackedLicense(archive, process.cwd())
const sha256 = createHash('sha256').update(await readFile(archive)).digest('hex')
const result = await consumerSmoke(`file:${archive}`, 'Packed', { browser })
const receipt = { archive, consumer: result.consumer, sha256, browserEvidence: result.browserEvidence }
process.stdout.write(`${JSON.stringify(receipt)}\n`)
