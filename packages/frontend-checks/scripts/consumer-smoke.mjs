import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'

export async function consumerSmoke(spec, label) {
  const consumer = await mkdtemp(path.join(os.tmpdir(), 'ztd-frontend-consumer-'))
  await writeConsumerFiles(consumer, spec)
  execFileSync('pnpm', ['install'], { cwd: consumer, stdio: 'inherit' })
  execFileSync('pnpm', ['install', '--frozen-lockfile'], { cwd: consumer, stdio: 'inherit' })
  execFileSync('node', ['smoke.mjs'], { cwd: consumer, stdio: 'inherit' })
  const report = JSON.parse(execFileSync('pnpm', [
    'exec',
    'ztd-css',
    './css-check.config.mjs',
  ], { cwd: consumer, encoding: 'utf8' }))
  assert.equal(report.status, 'passed')
  await checkTypes(consumer)
  await checkBrowserPass(consumer)
  await checkBrowserFailure(consumer)
  process.stdout.write(`${label} imports, types, CSS CLI and browser pass/failure evidence verified in ${consumer}\n`)
  return consumer
}

async function writeConsumerFiles(consumer, spec) {
  await writeFile(path.join(consumer, 'package.json'), JSON.stringify({
    private: true,
    type: 'module',
    packageManager: 'pnpm@11.22.0',
    dependencies: { '@ztd-me/frontend-checks': spec, '@playwright/test': '1.62.0' },
    devDependencies: { 'typescript': '6.0.3', '@types/node': '24.12.0' },
  }))
  await writeFile(path.join(consumer, 'pnpm-workspace.yaml'), `packages:
  - .
minimumReleaseAge: 1440
minimumReleaseAgeStrict: true
minimumReleaseAgeExcludePrune: true
minimumReleaseAgeExclude:
  - '@ztd-me/*'
shellEmulator: true
trustPolicy: no-downgrade
`)
  await writeCssFixtures(consumer)
  await writeFile(path.join(consumer, 'smoke.mjs'), `
import assert from 'node:assert/strict'
import { checkCss } from '@ztd-me/frontend-checks/css'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'
const options = { files: ['app.css'], classFiles: ['app.tsx'], tokenFiles: ['tokens.css'] }
assert.equal((await checkCss(options)).status, 'passed')
const invalid = await checkCss({ ...options, files: ['app.css', 'invalid.css'], classFiles: ['invalid.tsx'] })
assert.equal(invalid.status, 'failed')
for (const rule of [
  'property-no-unknown', 'ztd/semantic-color',
  'ztd/semantic-class-color', 'ztd/defined-class-custom-property',
]) {
  assert.ok(invalid.warnings.some(warning => warning.rule === rule), rule)
}
assert.equal(verificationArtifacts('out').outputDir, 'out/test-results')
`)
}

async function writeCssFixtures(consumer) {
  await writeFile(path.join(consumer, 'tokens.css'), ':root { --paint: #234567; }\n')
  await writeFile(path.join(consumer, 'app.tsx'), 'const classes = "text-foreground bg-[var(--paint)] h-8";\n')
  await writeFile(path.join(consumer, 'invalid.tsx'), 'const classes = "bg-red-500 text-(--missing)";\n')
  await writeFile(path.join(consumer, 'invalid.css'), `@utility invalid-utility {
  & > a { unknown-ztd-property: 1; background: red; }
}
`)
  await writeFile(path.join(consumer, 'app.css'), `@import "./tokens.css";

@theme inline {
  --color-foreground: var(--paint);
  --text-body: 1rem;
  --text-body--line-height: 1.75;
}

@utility consumer-utility {
  & > a { color: var(--paint); }
}

a { color: var(--paint); }

@custom-variant dark {
  :root[data-frontend-mode="dark"] & { @slot; }

  @media (prefers-color-scheme: dark) {
    :root[data-frontend-mode="system"] & { @slot; }
  }
}
`)
  await writeFile(path.join(consumer, 'css-check.config.mjs'), `export default {
  files: ['app.css'], classFiles: ['app.tsx'], tokenFiles: ['tokens.css'],
}
`)
}

async function checkTypes(consumer) {
  await writeFile(path.join(consumer, 'api.ts'), await readFile('test/types/api.ts', 'utf8'))
  await writeFile(path.join(consumer, 'tsconfig.json'), JSON.stringify({
    compilerOptions: {
      module: 'NodeNext',
      moduleResolution: 'NodeNext',
      strict: true,
      noUncheckedIndexedAccess: true,
      noEmit: true,
      lib: ['ESNext', 'DOM'],
      types: ['node'],
      skipLibCheck: false,
    },
    include: ['api.ts'],
  }))
  execFileSync('pnpm', ['exec', 'tsc'], { cwd: consumer, stdio: 'inherit' })
}

async function checkBrowserPass(consumer) {
  await mkdir(path.join(consumer, 'browser-positive'))
  await writeFile(path.join(consumer, 'positive.config.mjs'), `
import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'
export default defineConfig({
  ...verificationArtifacts('positive-artifacts'), testDir: './browser-positive', workers: 1,
})
`)
  const source = await readFile('test/browser/accessibility.spec.mjs', 'utf8')
  assert.ok(source.includes('from \'../../src/playwright.mjs\''))
  const imported = source.replace('from \'../../src/playwright.mjs\'', 'from \'@ztd-me/frontend-checks/playwright\'')
  const specFile = path.join(consumer, 'browser-positive', 'accessibility.spec.mjs')
  await writeFile(specFile, imported)
  execFileSync('pnpm', [
    'exec',
    'playwright',
    'test',
    '-c',
    'positive.config.mjs',
  ], { cwd: consumer, stdio: 'inherit' })
  const reportFile = path.join(consumer, 'positive-artifacts', 'playwright.json')
  const browserReport = JSON.parse(await readFile(reportFile, 'utf8'))
  assert.equal(browserReport.stats.expected, 3)
  assert.equal(browserReport.stats.unexpected, 0)
}

async function checkBrowserFailure(consumer) {
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
  const reportFile = path.join(consumer, 'negative-artifacts', 'playwright.json')
  const browserReport = JSON.parse(await readFile(reportFile, 'utf8'))
  assert.equal(browserReport.stats.unexpected, 1)
  const failed = browserReport.suites[0].specs[0].tests[0].results[0]
  assert.ok(failed.error.message.includes('button-name'))
  assert.ok(failed.attachments.some(attachment => attachment.name === 'a11y-state'))
}
