import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'

const consumer = await mkdtemp(path.join(os.tmpdir(), 'ztd-frontend-pack-'))
execFileSync('pnpm', [
  'pack',
  '--pack-destination',
  consumer,
], { stdio: 'inherit' })
const archive = path.join(consumer, 'ztd-me-frontend-checks-0.1.0.tgz')
await writeFile(path.join(consumer, 'package.json'), JSON.stringify({
  private: true,
  type: 'module',
  packageManager: 'pnpm@11.22.0',
  dependencies: { '@ztd-me/frontend-checks': `file:${archive}`, '@playwright/test': '1.62.0' },
  devDependencies: { 'typescript': '6.0.3', '@types/node': '24.12.0' },
}))
await writeFile(path.join(consumer, 'pnpm-workspace.yaml'), `packages:
  - .
minimumReleaseAge: 1440
minimumReleaseAgeExcludePrune: true
shellEmulator: true
trustPolicy: no-downgrade
`)
await writeFile(path.join(consumer, 'app.css'), ':root { --paint: #234567; }\na { color: var(--paint); }\n')
await writeFile(path.join(consumer, 'css-check.config.mjs'), 'export default { files: ["app.css"] }\n')
await writeFile(path.join(consumer, 'smoke.mjs'), `
import assert from 'node:assert/strict'
import { checkCss } from '@ztd-me/frontend-checks/css'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'
assert.equal((await checkCss({ files: ['app.css'] })).status, 'passed')
assert.equal(verificationArtifacts('out').outputDir, 'out/test-results')
`)
execFileSync('pnpm', ['install'], { cwd: consumer, stdio: 'inherit' })
execFileSync('pnpm', ['install', '--frozen-lockfile'], { cwd: consumer, stdio: 'inherit' })
execFileSync('node', ['smoke.mjs'], { cwd: consumer, stdio: 'inherit' })
const report = JSON.parse(execFileSync('pnpm', [
  'exec',
  'ztd-css',
  './css-check.config.mjs',
], { cwd: consumer, encoding: 'utf8' }))
assert.equal(report.status, 'passed')
const sha256 = createHash('sha256').update(await readFile(archive)).digest('hex')
process.stdout.write(`${JSON.stringify({ archive, consumer, sha256 })}\n`)

await writeFile(path.join(consumer, 'api.ts'), await readFile('test/types/api.ts', 'utf8'))
await writeFile(path.join(consumer, 'tsconfig.json'), JSON.stringify({
  compilerOptions: {
    module: 'NodeNext',
    moduleResolution: 'NodeNext',
    strict: true,
    noEmit: true,
    lib: ['ESNext', 'DOM'],
    types: ['node'],
    skipLibCheck: false,
  },
  include: ['api.ts'],
}))
execFileSync('pnpm', ['exec', 'tsc'], { cwd: consumer, stdio: 'inherit' })
await mkdir(path.join(consumer, 'browser-negative'))
await writeFile(path.join(consumer, 'negative.config.mjs'), `
import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'
export default defineConfig({
  ...verificationArtifacts('negative-artifacts'), testDir: './browser-negative', workers: 1,
})
`)
await writeFile(path.join(consumer, 'browser-negative', 'negative.spec.mjs'), `
import { test } from '@playwright/test'
import { assertAccessible } from '@ztd-me/frontend-checks/playwright'
test('broken accessible name fails', async ({ page }, info) => {
  await page.setContent(
    '<html lang="en"><head><title>Fixture</title></head><body><main><button></button></main></body></html>',
  )
  await assertAccessible(page, info)
})
`)
assert.throws(() => execFileSync('pnpm', [
  'exec',
  'playwright',
  'test',
  '-c',
  'negative.config.mjs',
], {
  cwd: consumer,
  stdio: 'pipe',
}), /Command failed/)
const browserReport = JSON.parse(await readFile(path.join(consumer, 'negative-artifacts', 'playwright.json'), 'utf8'))
assert.equal(browserReport.stats.unexpected, 1)
const failed = browserReport.suites[0].specs[0].tests[0].results[0]
assert.ok(failed.error.message.includes('button-name'))
assert.ok(failed.attachments.some(attachment => attachment.name === 'a11y-state'))
process.stdout.write(`Packed types and browser failure evidence verified in ${consumer}\n`)
