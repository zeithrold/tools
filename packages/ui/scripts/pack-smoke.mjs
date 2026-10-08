import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { assertPackedLicense } from '../../../scripts/license-coverage.mjs'
import { browserArtifactRoot } from './browser-artifacts.mjs'
import { prepareConsumer } from './prepare-consumer.mjs'

await mkdir('.artifacts', { recursive: true })
const directory = await mkdtemp(path.resolve('.artifacts/pack-'))
const archive = path.join(directory, 'package.tgz')
execFileSync('pnpm', [
  'pack',
  '--out',
  archive,
], { stdio: 'inherit' })
const notices = Object.fromEntries((await readdir('third-party')).map(file => [
  `dist/assets/${file}`,
  `third-party/${file}`,
]))
notices['THIRD_PARTY_NOTICES.md'] = 'THIRD_PARTY_NOTICES.md'
await assertPackedLicense(archive, process.cwd(), notices)
const consumer = await prepareConsumer(`file:${archive}`)
const sha256 = createHash('sha256').update(await readFile(archive)).digest('hex')
await writeFile('.artifacts/packed-consumer.json', JSON.stringify({
  archive,
  consumer,
  sha256,
  browserArtifacts: browserArtifactRoot('pack', Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW)),
  fontVerification: process.env.ZTD_LOCAL_FONT_PREVIEW ? 'local-preview-only' : 'google-fonts-api',
}, null, 2))
execFileSync('pnpm', [
  'exec',
  'playwright',
  'test',
], { stdio: 'inherit', env: { ...process.env, ZTD_UI_CONSUMER: consumer, ZTD_UI_VERIFICATION_MODE: 'pack' } })
