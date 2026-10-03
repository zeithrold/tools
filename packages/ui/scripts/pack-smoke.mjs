import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { prepareConsumer } from './prepare-consumer.mjs'

await mkdir('.artifacts', { recursive: true })
const directory = await mkdtemp(path.resolve('.artifacts/pack-'))
const archive = path.join(directory, 'package.tgz')
execFileSync('pnpm', [
  'pack',
  '--out',
  archive,
], { stdio: 'inherit' })
const consumer = await prepareConsumer(`file:${archive}`)
const sha256 = createHash('sha256').update(await readFile(archive)).digest('hex')
await writeFile('.artifacts/packed-consumer.json', JSON.stringify({
  archive,
  consumer,
  sha256,
  fontVerification: process.env.ZTD_LOCAL_FONT_PREVIEW ? 'local-preview-only' : 'google-fonts-api',
}, null, 2))
execFileSync('pnpm', [
  'exec',
  'playwright',
  'test',
], { stdio: 'inherit', env: { ...process.env, ZTD_UI_CONSUMER: consumer } })
