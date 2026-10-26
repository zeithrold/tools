import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { browserArtifactRoot, verifyBrowserEvidence } from '../scripts/browser-artifacts.mjs'

const checker = '@ztd-me/frontend-checks'
const checkerVersion = '9.9.9'

async function retainEvidence(root) {
  const attachments = path.join(root, 'test-results', 'synthetic-case', 'attachments')
  await mkdir(attachments, { recursive: true })
  const scan = path.join(attachments, 'a11y-state.json')
  const capture = path.join(attachments, 'state.png')
  await writeFile(scan, JSON.stringify({
    violations: [],
    passes: [],
    incomplete: [],
    inapplicable: [],
  }))
  await writeFile(capture, 'synthetic png')
  return {
    suites: [
      {
        title: 'synthetic.spec.mjs',
        specs: [
          {
            title: 'synthetic case',
            file: 'synthetic.spec.mjs',
            tests: [
              {
                projectName: '',
                results: [
                  {
                    retry: 0,
                    status: 'passed',
                    duration: 8,
                    attachments: [
                      { name: 'a11y-state', contentType: 'application/json', path: scan },
                      { name: 'state', contentType: 'image/png', path: capture },
                    ],
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
  }
}

async function retainTimings(root, executed = 1) {
  await writeFile(path.join(root, 'test-timings.json'), JSON.stringify({
    schemaVersion: 1,
    status: 'passed',
    elapsedMs: 12,
    configuration: { projects: [''] },
    summary: {
      executed,
      skipped: 0,
      durationSamples: executed,
      phaseTimingCases: executed,
    },
    projects: {},
    cases: [
      {
        title: 'synthetic.spec.mjs › synthetic case',
        project: '',
        file: 'test/browser/synthetic.spec.mjs',
        status: 'passed',
        retry: 0,
        repeatEachIndex: 0,
        durationMs: 8,
        wallDurationMs: 10,
        setupMs: 1,
        bodyMs: 8,
        teardownMs: 1,
        stepsAvailable: true,
        phaseTimingAvailable: true,
        slowestSteps: [],
      },
    ],
  }))
}

async function fixture(t, localFontPreview = false) {
  const base = await mkdtemp(path.join(tmpdir(), 'ui-browser-evidence-'))
  t.after(() => rm(base, { recursive: true, force: true }))
  await mkdir(path.join(base, 'node_modules', ...checker.split('/')), { recursive: true })
  await writeFile(path.join(base, 'package.json'), JSON.stringify({
    devDependencies: { [checker]: checkerVersion },
  }))
  await writeFile(
    path.join(base, 'node_modules', ...checker.split('/'), 'package.json'),
    JSON.stringify({ name: checker, version: checkerVersion }),
  )
  for (const mode of ['pack', 'source']) {
    const root = browserArtifactRoot(mode, localFontPreview)
    const consumer = `synthetic-${mode}-consumer`
    const report = {
      config: {
        rootDir: path.join(base, 'test', 'browser'),
        metadata: {
          consumer,
          deliveryMode: mode,
          fontVerification: localFontPreview ? 'local-preview-only' : 'google-fonts-api',
        },
      },
      stats: { expected: 1, skipped: 0, unexpected: 0, duration: 12 },
    }
    await mkdir(path.join(base, root, 'playwright-report'), { recursive: true })
    await writeFile(path.join(base, root, 'playwright-report', 'index.html'), '<html>synthetic report</html>')
    await retainTimings(path.join(base, root))
    const evidence = await retainEvidence(path.join(base, root))
    await writeFile(
      path.join(base, root, 'playwright.json'),
      JSON.stringify({ ...report, ...evidence }),
    )
    const receiptName = mode === 'pack' ? 'packed-consumer.json' : 'source-consumer.json'
    await writeFile(path.join(base, '.artifacts', receiptName), JSON.stringify({ consumer, browserArtifacts: root }))
  }
  return base
}

test('delivery and font modes keep separate browser artifacts', () => {
  const roots = [
    'pack',
    'source',
    'public-source',
    'manual',
  ].flatMap(mode => [
    browserArtifactRoot(mode),
    browserArtifactRoot(mode, true),
  ])
  assert.equal(new Set(roots).size, roots.length)
  assert.throws(() => browserArtifactRoot('../other'), /Unknown browser verification mode/)
})

test('both independent consumer reports and HTML results must be retained', async (t) => {
  const base = await fixture(t)
  assert.equal((await verifyBrowserEvidence({ base })).length, 2)
  await rm(path.join(base, browserArtifactRoot('pack'), 'playwright.json'))
  await assert.rejects(verifyBrowserEvidence({ base }), /ENOENT/)
})

test('a later source report cannot stand in for pack evidence', async (t) => {
  const base = await fixture(t)
  const source = await readFile(path.join(base, browserArtifactRoot('source'), 'playwright.json'))
  await writeFile(path.join(base, browserArtifactRoot('pack'), 'playwright.json'), source)
  await assert.rejects(verifyBrowserEvidence({ base }), /report belongs to another gate/)
})

test('stale consumer evidence does not satisfy a fresh run', async (t) => {
  const base = await fixture(t)
  const filename = path.join(base, browserArtifactRoot('source'), 'playwright.json')
  const report = JSON.parse(await readFile(filename, 'utf8'))
  report.config.metadata.consumer = 'another-synthetic-consumer'
  await writeFile(filename, JSON.stringify(report))
  await assert.rejects(verifyBrowserEvidence({ base }), /report belongs to another consumer/)
})

test('preview evidence must identify its font verification mode', async (t) => {
  const base = await fixture(t, true)
  assert.equal((await verifyBrowserEvidence({ base, localFontPreview: true })).length, 2)
  const filename = path.join(base, browserArtifactRoot('source', true), 'playwright.json')
  const report = JSON.parse(await readFile(filename, 'utf8'))
  report.config.metadata.fontVerification = 'google-fonts-api'
  await writeFile(filename, JSON.stringify(report))
  await assert.rejects(verifyBrowserEvidence({ base, localFontPreview: true }), /another font verification mode/)
})

test('scans and captures must be attached from retained files', async (t) => {
  const base = await fixture(t)
  const filename = path.join(base, browserArtifactRoot('pack'), 'playwright.json')
  const report = JSON.parse(await readFile(filename, 'utf8'))
  const scanPath = report.suites[0].specs[0].tests[0].results[0].attachments[0]
  scanPath.body = 'e30='
  scanPath.path = undefined
  await writeFile(filename, JSON.stringify(report))
  await assert.rejects(verifyBrowserEvidence({ base }), /must be attached from a retained file/)
})

test('timings must describe the same run as the retained report', async (t) => {
  const base = await fixture(t)
  await retainTimings(path.join(base, browserArtifactRoot('source')), 2)
  await assert.rejects(verifyBrowserEvidence({ base }), /timing report covers another run/)
})

test('matching summary counts cannot hide missing or unrelated timing cases', async (t) => {
  for (const mutation of [
    (timings) => { timings.cases = [] },
    (timings) => { timings.cases[0].title = 'unrelated case' },
    (timings) => { timings.cases[0].retry = 1 },
  ]) {
    const base = await fixture(t)
    const filename = path.join(base, browserArtifactRoot('source'), 'test-timings.json')
    const timings = JSON.parse(await readFile(filename, 'utf8'))
    mutation(timings)
    await writeFile(filename, JSON.stringify(timings))
    await assert.rejects(verifyBrowserEvidence({ base }), /timing case\/attempt evidence differs/)
  }
})

test('timings with the same cases must match the observed run duration', async (t) => {
  const base = await fixture(t)
  const filename = path.join(base, browserArtifactRoot('source'), 'test-timings.json')
  const timings = JSON.parse(await readFile(filename, 'utf8'))
  timings.elapsedMs = 13
  await writeFile(filename, JSON.stringify(timings))
  await assert.rejects(verifyBrowserEvidence({ base }), /timing report run duration differs/)
})

test('the installed checker must be the pinned artifact', async (t) => {
  const base = await fixture(t)
  await writeFile(
    path.join(base, 'node_modules', ...checker.split('/'), 'package.json'),
    JSON.stringify({ name: checker, version: '9.9.8' }),
  )
  await assert.rejects(verifyBrowserEvidence({ base }), /is not the pinned/)
})
