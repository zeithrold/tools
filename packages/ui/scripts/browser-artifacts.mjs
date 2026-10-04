import assert from 'node:assert/strict'
import { readFile, stat } from 'node:fs/promises'
import path from 'node:path'

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

export async function verifyBrowserEvidence({ base = '.', localFontPreview = false } = {}) {
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
    verified.push(root)
  }
  return verified
}
