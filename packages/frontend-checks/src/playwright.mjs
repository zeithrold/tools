import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { readFile, rm, writeFile } from 'node:fs/promises'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import AxeBuilder from '@axe-core/playwright'

const defaultTags = [
  'wcag2a',
  'wcag2aa',
  'wcag21aa',
  'wcag22aa',
]

/**
 * Playwright names its attachment copy after the display name and appends its own hash and
 * extension, so path attachments must leave room for them in a file name.
 */
const attachmentNameByteLimit = 180

/** Source file names already reserved by this test, so scans cannot reuse one another's name. */
const usedSourceNames = new WeakMap()

/** Labels are developer-supplied, so keep only portable filename characters. */
function sourceStem(label) {
  const stem = String(label)
    .normalize('NFKC')
    .replace(/[^\p{L}\p{N}._-]+/gu, '-')
    .slice(0, 60)
    .replace(/^[.-]+|[.-]+$/g, '')
  // Windows device names remain reserved even when followed by an extension.
  if (/^(?:con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)/i.test(stem)) {
    return `state-${stem}`
  }
  return stem || 'state'
}

/**
 * Reserve this helper's own source file. Exclusive creation never overwrites a caller's
 * same-named evidence, and names already handed out in this test stay reserved for the next
 * scan.
 */
async function reserveSourcePath(testInfo, prefix, label, extension) {
  const stem = `${prefix}${sourceStem(label)}`
  let used = usedSourceNames.get(testInfo)
  if (!used) {
    used = new Set()
    usedSourceNames.set(testInfo, used)
  }
  for (let index = 1; ; index += 1) {
    const name = index === 1 ? `${stem}${extension}` : `${stem}-${index}${extension}`
    if (used.has(name)) {
      continue
    }
    const file = testInfo.outputPath(name)
    try {
      await writeFile(file, '', { flag: 'wx' })
    }
    catch (error) {
      if (error.code === 'EEXIST') {
        continue
      }
      throw error
    }
    used.add(name)
    return file
  }
}

/**
 * Prefer path attachments to avoid large inline payloads. Playwright copies the file into the
 * test output directory before `attach()` resolves and documents that the source may then be
 * removed. A label too long for Playwright's name-derived copy keeps the previous inline form
 * so the display name stays intact. Only this helper's own source file is removed; if attaching
 * fails it stays on disk for diagnosis and the failure propagates.
 */
async function attachSourceFile(testInfo, name, file, contentType) {
  if (Buffer.byteLength(name) > attachmentNameByteLimit) {
    await testInfo.attach(name, { body: await readFile(file), contentType })
  }
  else {
    await testInfo.attach(name, { path: file, contentType })
  }
  await rm(file)
}

/** Scan after each relevant UI state is established, not only the initial route. */
export async function assertAccessible(page, testInfo, options = {}) {
  if (options.tags && !options.tags.length) {
    throw new TypeError('Accessibility tags must not be empty')
  }
  let builder = new AxeBuilder({ page }).withTags(options.tags ?? defaultTags)
  if (options.include) {
    builder = builder.include(options.include)
  }
  const results = await builder.analyze()
  const label = options.label ?? 'state'
  const file = await reserveSourcePath(testInfo, 'a11y-', label, '.json')
  await writeFile(file, JSON.stringify(results, null, 2))
  await attachSourceFile(testInfo, `a11y-${label}`, file, 'application/json')
  assert.equal(results.violations.length, 0, JSON.stringify(results.violations, null, 2))
  return results
}

/** Screenshots are captured evidence; this helper does not maintain baselines. */
export async function captureState(page, testInfo, label) {
  const file = await reserveSourcePath(testInfo, '', label, '.png')
  await page.screenshot({ path: file, fullPage: true })
  await attachSourceFile(testInfo, label, file, 'image/png')
}

/** Apply these paths to the project's existing config; retain its webServer/projects. */
export function verificationArtifacts(root = process.env.ZT_ARTIFACTS_DIR ?? '.zt/browser') {
  return {
    outputDir: `${root}/test-results`,
    reporter: [
      ['list'],
      [
        'html',
        { outputFolder: `${root}/playwright-report`, open: 'never' },
      ],
      [
        'json',
        { outputFile: `${root}/playwright.json` },
      ],
      [
        fileURLToPath(new URL('./timing-reporter.mjs', import.meta.url)),
        { outputFile: `${root}/test-timings.json` },
      ],
    ],
    use: { trace: 'retain-on-failure', screenshot: 'only-on-failure', video: 'retain-on-failure' },
  }
}
