import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { test } from 'node:test'
import { fileURLToPath, pathToFileURL } from 'node:url'

const require = createRequire(import.meta.url)

test('native Playwright all-hooks use wall time without erasing the actual test body', async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), 'native-timing-'))
  const outputFile = path.join(directory, 'test-timings.json')
  const api = pathToFileURL(require.resolve('@playwright/test')).href
  const reporter = fileURLToPath(new URL('../src/timing-reporter.mjs', import.meta.url))
  const config = path.join(directory, 'playwright.config.mjs')
  await writeFile(config, `export default {
    testDir: ${JSON.stringify(directory)},
    testMatch: 'native.spec.mjs',
    outputDir: ${JSON.stringify(path.join(directory, 'test-results'))},
    workers: 1,
    retries: 0,
    reporter: [['list'], [${JSON.stringify(reporter)}, { outputFile: ${JSON.stringify(outputFile)} }]],
  }\n`)
  await writeFile(path.join(directory, 'native.spec.mjs'), `
    import playwright from ${JSON.stringify(api)}
    import { setTimeout as delay } from 'node:timers/promises'
    const { test } = playwright
    test.beforeAll(async () => await delay(200))
    test.afterAll(async () => await delay(100))
    test('body has its own duration', async () => await delay(50))
  `)
  // No browser fixture: this validates the installed runner's clocks without browser binaries.
  execFileSync('pnpm', [
    'exec',
    'playwright',
    'test',
    '--config',
    config,
  ], {
    cwd: process.cwd(),
    stdio: 'inherit',
    timeout: 30000,
  })
  const report = JSON.parse(await readFile(outputFile, 'utf8'))
  assert.equal(report.status, 'passed')
  assert.equal(report.cases.length, 1)
  const timing = report.cases[0]
  assert.equal(timing.phaseTimingAvailable, true)
  assert.ok(timing.setupMs >= 200)
  assert.ok(timing.teardownMs >= 100)
  assert.ok(timing.bodyMs > 0, 'the beforeAll delay must not erase the real body duration')
  assert.ok(timing.wallDurationMs - timing.durationMs >= 200, 'all-hooks are wall time, not charged test time')
  assert.equal(timing.wallDurationMs, timing.setupMs + timing.bodyMs + timing.teardownMs)
})
