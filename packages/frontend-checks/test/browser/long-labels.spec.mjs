import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { readdir } from 'node:fs/promises'
import { test } from '@playwright/test'
import { assertAccessible, captureState } from '../../src/playwright.mjs'

const safe = `<!doctype html><html lang="en"><head><title>Verification fixture</title></head>
<body><a href="#content">Skip to content</a><main id="content" tabindex="-1"><h1>Account</h1>
<label for="name">Name</label><input id="name"><button type="button">Save</button></main></body></html>`

const pngSignature = Buffer.from([
  0x89,
  0x50,
  0x4E,
  0x47,
  0x0D,
  0x0A,
  0x1A,
  0x0A,
])

/** Longer than Playwright can derive a path-attachment file name from. */
const longAscii = 'a'.repeat(220)
const longMultibyte = '確'.repeat(120)

function attachmentsNamed(info, name) {
  return info.attachments.filter(attachment => attachment.name === name)
}

function assertInline(attachment, name) {
  assert.equal(attachment.name, name)
  assert.equal(attachment.path, undefined)
  assert.ok(Buffer.isBuffer(attachment.body))
  assert.ok(Buffer.byteLength(name) > 180)
}

async function assertOnlyAttachmentsRemain(info) {
  assert.deepEqual(await readdir(info.outputDir), ['attachments'])
}

test('very long scan labels keep their display name and fall back to an inline body', async ({ page }, info) => {
  await page.setContent(safe)
  await assertAccessible(page, info, { label: longAscii })
  await assertAccessible(page, info, { label: longMultibyte })
  await assertAccessible(page, info, { label: 'short' })

  for (const label of [longAscii, longMultibyte]) {
    const name = `a11y-${label}`
    const [scan] = attachmentsNamed(info, name)
    assert.ok(scan, name)
    assertInline(scan, name)
    const results = JSON.parse(scan.body)
    for (const section of [
      'violations',
      'passes',
      'incomplete',
      'inapplicable',
    ]) {
      assert.ok(Array.isArray(results[section]), section)
    }
    assert.equal(results.violations.length, 0)
  }

  const [short] = attachmentsNamed(info, 'a11y-short')
  assert.ok(short, 'a11y-short')
  assert.ok(short.path, 'a normal label stays path-backed')
  assert.equal(short.body, undefined)
  await assertOnlyAttachmentsRemain(info)
})

test('very long screenshot labels keep their display name and fall back to an inline body', async ({ page }, info) => {
  await page.setContent(safe)
  await captureState(page, info, longAscii)
  await captureState(page, info, longMultibyte)
  await captureState(page, info, 'short')

  for (const label of [longAscii, longMultibyte]) {
    const [shot] = attachmentsNamed(info, label)
    assert.ok(shot, label)
    assertInline(shot, label)
    assert.deepEqual(shot.body.subarray(0, pngSignature.length), pngSignature)
  }

  const [short] = attachmentsNamed(info, 'short')
  assert.ok(short, 'short')
  assert.ok(short.path, 'a normal label stays path-backed')
  assert.equal(short.body, undefined)
  await assertOnlyAttachmentsRemain(info)
})
