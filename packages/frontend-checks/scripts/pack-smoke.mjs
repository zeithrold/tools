import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdtemp, readFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { consumerSmoke } from './consumer-smoke.mjs'

const directory = await mkdtemp(path.join(os.tmpdir(), 'ztd-frontend-pack-'))
const { version } = JSON.parse(await readFile('package.json', 'utf8'))
execFileSync('pnpm', [
  'pack',
  '--pack-destination',
  directory,
], { stdio: 'inherit' })
const archive = path.join(directory, `ztd-me-frontend-checks-${version}.tgz`)
const sha256 = createHash('sha256').update(await readFile(archive)).digest('hex')
const consumer = await consumerSmoke(`file:${archive}`, 'Packed')
process.stdout.write(`${JSON.stringify({ archive, consumer, sha256 })}\n`)
