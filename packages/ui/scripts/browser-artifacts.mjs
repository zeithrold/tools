import assert from 'node:assert/strict'
import { readFile, stat } from 'node:fs/promises'
import path from 'node:path'

const checker = '@ztd-me/frontend-checks'

async function readJson(file) {
  return JSON.parse(await readFile(file, 'utf8'))
}

export function browserArtifactRoot(mode, localFontPreview = false) {
  assert.ok([
    'pack',
    'source',
    'public-source',
    'manual',
  ].includes(mode), `Unknown browser verification mode: ${mode}`)
  const parent = localFontPreview ? '.artifacts/browser-local-font-preview' : '.artifacts/browser'
  return `${parent}/${mode}`
}

/** Only the pinned, installed checker artifact may produce the retained evidence. */
async function verifyCheckerPin(base) {
  const pinned = (await readJson(path.join(base, 'package.json'))).devDependencies?.[checker]
  assert.match(pinned ?? '', /^\d+\.\d+\.\d+$/u, `${checker} must stay pinned to one exact version`)
  const installed = await readJson(path.join(base, 'node_modules', ...checker.split('/'), 'package.json'))
  assert.equal(installed.version, pinned, `${checker} ${installed.version} is not the pinned ${pinned}`)
}

/** Every result's scans and captures must reference retained files, not inline payloads. */
function retainedAttachments(report) {
  const attachments = []
  const visit = (suites = []) => {
    for (const suite of suites) {
      for (const spec of suite.specs ?? []) {
        for (const testCase of spec.tests ?? []) {
          for (const result of testCase.results ?? []) {
            attachments.push(...result.attachments ?? [])
          }
        }
      }
      visit(suite.suites)
    }
  }
  visit(report.suites)
  return attachments
}

async function verifyAttachments({ mode, base, root, report }) {
  const evidenceRoot = path.resolve(base, root)
  let scans = 0
  let captures = 0
  for (const attachment of retainedAttachments(report)) {
    const name = attachment.name ?? ''
    const scan = name.startsWith('a11y-')
    const capture = attachment.contentType === 'image/png'
    if (!scan && !capture) {
      continue
    }
    assert.ok(attachment.path, `${mode}: ${name} must be attached from a retained file`)
    assert.equal(attachment.body, undefined, `${mode}: ${name} still embeds its evidence inline`)
    const file = path.resolve(base, attachment.path)
    assert.ok(file.startsWith(`${evidenceRoot}${path.sep}`), `${mode}: ${name} evidence is outside ${root}`)
    assert.ok((await stat(file)).size > 0, `${mode}: ${name} evidence file is empty`)
    if (scan) {
      scans += 1
      const result = await readJson(file)
      for (const section of [
        'violations',
        'passes',
        'incomplete',
        'inapplicable',
      ]) {
        assert.ok(Array.isArray(result[section]), `${mode}: ${name} lacks the complete Axe ${section} section`)
      }
    }
    else {
      captures += 1
    }
  }
  assert.ok(scans > 0, `${mode}: no file-backed Axe scans`)
  assert.ok(captures > 0, `${mode}: no file-backed captures`)
}

function specTimingCases(base, rootDir, titles, spec) {
  const repeats = new Map()
  return (spec.tests ?? []).flatMap((testCase) => {
    const project = testCase.projectName ?? ''
    const repeatEachIndex = repeats.get(project) ?? 0
    repeats.set(project, repeatEachIndex + 1)
    return (testCase.results ?? []).map(result => ({
      file: path.relative(base, path.resolve(rootDir, spec.file)),
      project,
      title: [
        project,
        ...titles,
        spec.title,
      ].filter(Boolean).join(' › '),
      retry: result.retry,
      repeatEachIndex,
      status: result.status,
      durationMs: result.duration,
    }))
  })
}

function reportTimingCases(base, report) {
  const cases = []
  const visit = (suites = [], parents = []) => {
    for (const suite of suites) {
      const titles = [
        ...parents,
        suite.title,
      ]
      for (const spec of suite.specs ?? []) {
        cases.push(...specTimingCases(base, report.config.rootDir, titles, spec))
      }
      visit(suite.suites, titles)
    }
  }
  visit(report.suites)
  return cases
}

/** Reconcile native case/attempt observations and full-run duration, not only counts. */
async function verifyTimings({ mode, base, root, report }) {
  const timings = await readJson(path.join(base, root, 'test-timings.json'))
  assert.equal(timings.schemaVersion, 1, `${mode}: unknown timing report schema`)
  assert.equal(timings.summary.executed, report.stats.expected, `${mode}: timing report covers another run`)
  assert.equal(timings.summary.skipped, report.stats.skipped, `${mode}: timing report covers another run`)
  assert.ok(Number.isFinite(timings.elapsedMs), `${mode}: no observed run duration`)
  assert.equal(timings.elapsedMs, report.stats.duration, `${mode}: timing report run duration differs`)
  const fields = [
    'file',
    'project',
    'title',
    'retry',
    'repeatEachIndex',
    'status',
    'durationMs',
  ]
  const identities = cases => cases.map(item => JSON.stringify(fields.map(key => item[key]))).sort()
  assert.deepEqual(
    identities(timings.cases),
    identities(reportTimingCases(base, report)),
    `${mode}: timing case/attempt evidence differs from the browser report`,
  )
  assert.ok(timings.summary.durationSamples > 0, `${mode}: no measured case durations`)
  assert.ok(timings.summary.phaseTimingCases > 0, `${mode}: no observed setup/body/teardown phases`)
  for (const item of timings.cases) {
    for (const key of [
      'durationMs',
      'wallDurationMs',
      'setupMs',
      'bodyMs',
      'teardownMs',
    ]) {
      const value = item[key]
      const observed = value === null || (Number.isFinite(value) && value >= 0)
      assert.ok(observed, `${mode}: ${item.title} records an invalid ${key}`)
    }
    if (item.wallDurationMs !== null && item.durationMs !== null) {
      assert.ok(item.wallDurationMs >= item.durationMs, `${mode}: ${item.title} wall time must include hooks`)
    }
  }
}

export async function verifyBrowserEvidence({ base = '.', localFontPreview = false } = {}) {
  await verifyCheckerPin(base)
  const verified = []
  for (const mode of ['pack', 'source']) {
    const root = browserArtifactRoot(mode, localFontPreview)
    const receiptName = mode === 'pack' ? 'packed-consumer.json' : 'source-consumer.json'
    const receipt = JSON.parse(await readFile(path.join(base, '.artifacts', receiptName), 'utf8'))
    const report = JSON.parse(await readFile(path.join(base, root, 'playwright.json'), 'utf8'))
    assert.equal(receipt.browserArtifacts, root, `${mode}: receipt must identify its browser artifacts`)
    assert.equal(report.config.metadata.deliveryMode, mode, `${mode}: report belongs to another gate`)
    assert.equal(report.config.metadata.consumer, receipt.consumer, `${mode}: report belongs to another consumer`)
    const fontVerification = localFontPreview ? 'local-preview-only' : 'google-fonts-api'
    assert.equal(report.config.metadata.fontVerification, fontVerification, `${mode}: another font verification mode`)
    assert.ok(report.stats.expected > 0, `${mode}: no successful browser tests`)
    assert.equal(report.stats.unexpected, 0, `${mode}: browser failures remain`)
    const html = await stat(path.join(base, root, 'playwright-report', 'index.html'))
    assert.ok(html.size > 0, `${mode}: HTML evidence is empty`)
    assert.ok((await stat(path.join(base, root, 'test-results'))).isDirectory(), `${mode}: browser results are missing`)
    await verifyAttachments({ mode, base, root, report })
    await verifyTimings({ mode, base, root, report })
    verified.push(root)
  }
  return verified
}
