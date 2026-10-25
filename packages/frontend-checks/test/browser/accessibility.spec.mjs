import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { access, mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { expect, test } from '@playwright/test'
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

function attachmentsNamed(info, name) {
  return info.attachments.filter(attachment => attachment.name === name)
}

/** Evidence is attached by path, stays inside the test output directory and stays complete. */
async function attachedContent(info, attachment, contentType) {
  assert.equal(attachment.contentType, contentType)
  assert.equal(attachment.body, undefined, `${attachment.name} must be attached by path`)
  assert.ok(attachment.path, `${attachment.name} keeps a path`)
  const relative = path.relative(info.outputDir, attachment.path)
  assert.ok(relative && !relative.startsWith('..') && !path.isAbsolute(relative), relative)
  const content = await readFile(attachment.path)
  assert.ok(content.length > 0, `${attachment.name} stays readable`)
  return content
}

function assertCompleteScan(scan) {
  for (const section of [
    'violations',
    'passes',
    'incomplete',
    'inapplicable',
  ]) {
    assert.ok(Array.isArray(scan[section]), section)
  }
  assert.equal(scan.testEngine.name, 'axe-core')
  assert.ok(scan.passes.length > 0, 'passing rules stay in the evidence')
}

/** The helper removes only its own source file; the reporter-accessible copy remains. */
async function assertSourceRemoved(info, ...names) {
  for (const name of names) {
    await assert.rejects(access(path.join(info.outputDir, name)), { code: 'ENOENT' }, name)
  }
}

test('real Chromium scans and retains artifacts for an accessible keyboard flow', async ({ page }, info) => {
  await page.setContent(safe)
  await assertAccessible(page, info, { label: 'account' })
  // Repeating a label must retain both scans instead of overwriting the first one.
  await assertAccessible(page, info, { label: 'account' })
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('main')).toBeFocused()
  await captureState(page, info, 'account-keyboard')
  await captureState(page, info, 'account-keyboard')

  const scans = attachmentsNamed(info, 'a11y-account')
  assert.equal(scans.length, 2)
  assert.notEqual(scans[0].path, scans[1].path)
  for (const scan of scans) {
    const results = JSON.parse(await attachedContent(info, scan, 'application/json'))
    assertCompleteScan(results)
    assert.equal(results.violations.length, 0)
  }

  const shots = attachmentsNamed(info, 'account-keyboard')
  assert.equal(shots.length, 2)
  assert.notEqual(shots[0].path, shots[1].path)
  for (const shot of shots) {
    const png = await attachedContent(info, shot, 'image/png')
    assert.deepEqual(png.subarray(0, pngSignature.length), pngSignature)
  }

  await assertSourceRemoved(
    info,
    'a11y-account.json',
    'a11y-account-2.json',
    'account-keyboard.png',
    'account-keyboard-2.png',
  )
})

test('real Axe violation fails the assertion and retains the scan evidence', async ({ page }, info) => {
  const callerFile = path.join(info.outputDir, 'a11y-broken-button.json')
  await mkdir(info.outputDir, { recursive: true })
  await writeFile(callerFile, 'caller evidence')
  await page.setContent(safe.replace('Save</button>', '</button>'))
  await assert.rejects(assertAccessible(page, info, { label: 'broken-button' }), /button-name/)
  const [scan] = attachmentsNamed(info, 'a11y-broken-button')
  assert.ok(scan, 'the scan is attached before the assertion fails')
  const results = JSON.parse(await attachedContent(info, scan, 'application/json'))
  assertCompleteScan(results)
  assert.ok(results.violations.some(violation => violation.id === 'button-name'))
  // A same-named caller file keeps its bytes; the scan reserves the next name instead.
  assert.deepEqual(await readFile(callerFile), Buffer.from('caller evidence'))
  await assertSourceRemoved(info, 'a11y-broken-button-2.json')
})

test('dialog state and translated accessible name are scanned after interaction', async ({ page }, info) => {
  await page.setContent(`${safe}<dialog aria-labelledby="dialog-name"><h2 id="dialog-name">確認</h2>
<button type="button" onclick="this.closest('dialog').close()">閉じる</button></dialog>
<script>document.querySelector('button').onclick = () => document.querySelector('dialog').showModal()</script>`)
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByRole('dialog', { name: '確認' })).toBeVisible()
  // Non-ASCII and traversal-looking labels keep their evidence inside the test output directory.
  await assertAccessible(page, info, { label: '../確認' })
  await captureState(page, info, '../確認')
  const [scan] = attachmentsNamed(info, 'a11y-../確認')
  assert.ok(scan)
  assertCompleteScan(JSON.parse(await attachedContent(info, scan, 'application/json')))
  const [shot] = attachmentsNamed(info, '../確認')
  assert.ok(shot)
  const png = await attachedContent(info, shot, 'image/png')
  assert.deepEqual(png.subarray(0, pngSignature.length), pngSignature)
  await assertSourceRemoved(info, 'a11y-確認.json', '確認.png')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).not.toBeVisible()
  await expect(page.getByRole('button', { name: 'Save' })).toBeFocused()
})
