import { execFileSync } from 'node:child_process'
import { mkdtemp } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { consumerSmoke, packageRoot } from './smoke.mjs'

const directory = await mkdtemp(join(tmpdir(), 'ztd-eslint-pack-'))
const tarball = join(directory, 'package.tgz')
execFileSync('pnpm', [
  'pack',
  '--out',
  tarball,
], { cwd: packageRoot, stdio: 'inherit' })
await consumerSmoke(`file:${tarball}`, 'Packed artifact')
