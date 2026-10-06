import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { assertPackedLicense } from '../../../scripts/license-coverage.mjs'
import { consumerSmoke, packageRoot } from './smoke.mjs'

await mkdir('.artifacts', { recursive: true })
const directory = await mkdtemp(resolve('.artifacts/pack-'))
const tarball = join(directory, 'package.tgz')
execFileSync('pnpm', [
  'pack',
  '--out',
  tarball,
], { cwd: packageRoot, stdio: 'inherit' })
await assertPackedLicense(tarball, packageRoot)
const consumer = await consumerSmoke(`file:${tarball}`, 'Packed artifact')
const sha256 = createHash('sha256').update(await readFile(tarball)).digest('hex')
await writeFile('.artifacts/packed-consumer.json', JSON.stringify({ tarball, consumer, sha256 }, null, 2))
