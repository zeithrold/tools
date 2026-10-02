import assert from 'node:assert/strict'
import { expect, test } from '@playwright/test'
import { assertAccessible, captureState } from '../../src/playwright.mjs'

const safe = `<!doctype html><html lang="en"><head><title>Verification fixture</title></head>
<body><a href="#content">Skip to content</a><main id="content" tabindex="-1"><h1>Account</h1>
<label for="name">Name</label><input id="name"><button type="button">Save</button></main></body></html>`

test('real Chromium scans and retains artifacts for an accessible keyboard flow', async ({ page }, info) => {
  await page.setContent(safe)
  await assertAccessible(page, info, { label: 'account' })
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('main')).toBeFocused()
  await captureState(page, info, 'account-keyboard')
})

test('real Axe violation fails the assertion and retains the scan evidence', async ({ page }, info) => {
  await page.setContent(safe.replace('Save</button>', '</button>'))
  await assert.rejects(assertAccessible(page, info, { label: 'broken-button' }), /button-name/)
  assert.ok(info.attachments.some(attachment => attachment.name === 'a11y-broken-button'))
})

test('dialog state and translated accessible name are scanned after interaction', async ({ page }, info) => {
  await page.setContent(`${safe}<dialog aria-labelledby="dialog-name"><h2 id="dialog-name">確認</h2>
<button type="button" onclick="this.closest('dialog').close()">閉じる</button></dialog>
<script>document.querySelector('button').onclick = () => document.querySelector('dialog').showModal()</script>`)
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByRole('dialog', { name: '確認' })).toBeVisible()
  await assertAccessible(page, info, { label: 'dialog' })
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).not.toBeVisible()
  await expect(page.getByRole('button', { name: 'Save' })).toBeFocused()
})
