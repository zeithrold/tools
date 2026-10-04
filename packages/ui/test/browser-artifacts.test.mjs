import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { browserArtifactRoot, verifyBrowserEvidence } from '../scripts/browser-artifacts.mjs'

async function fixture(t, localFontPreview = false) {
  const base = await mkdtemp(path.join(tmpdir(), 'ui-browser-evidence-'))
  t.after(() => rm(base, { recursive: true, force: true }))
  for (const mode of ['pack', 'source']) {
    const root = browserArtifactRoot(mode, localFontPreview)
    const consumer = `synthetic-${mode}-consumer`
    const report = {
      config: {
        metadata: {
          consumer,
          deliveryMode: mode,
          fontVerification: localFontPreview ? 'local-preview-only' : 'google-fonts-api',
        },
      },
      stats: { expected: 1, unexpected: 0 },
    }
    await mkdir(path.join(base, root, 'test-results'), { recursive: true })
    await mkdir(path.join(base, root, 'playwright-report'), { recursive: true })
    await writeFile(path.join(base, root, 'playwright-report', 'index.html'), '<html>synthetic report</html>')
    await writeFile(path.join(base, root, 'playwright.json'), JSON.stringify(report))
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
