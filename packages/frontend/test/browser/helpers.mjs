import { test as baseTest, expect } from '@playwright/test'
import { assertAccessible as scan } from '@ztd-me/frontend-checks/playwright'

const loadedFontSheets = new WeakMap()

function observeFontSheets(page) {
  const sheets = new Map()
  loadedFontSheets.set(page, sheets)
  page.on('response', (response) => {
    if (response.status() === 200 && new URL(response.url()).origin === 'https://fonts.googleapis.com') {
      // Passive capture of the actual browser stylesheet response; never supply a font/network fixture.
      sheets.set(response.url(), response.text().catch(() => null))
    }
  })
}

export const test = baseTest.extend({
  capturedFontSheets: [
    async ({ page }, use) => {
      observeFontSheets(page)
      await use(undefined)
      loadedFontSheets.delete(page)
    },
    { auto: true },
  ],
})

export async function assertAccessible(page, info, options) {
  const nonce = await page.locator('#initial').evaluate(node => JSON.parse(node.textContent).styleNonce)
  const stylesheets = (await Promise.all(Array.from(loadedFontSheets.get(page) ?? [], async ([url, body]) => [
    url,
    await body,
  ]))).filter(([, body]) => body !== null)
  // Axe copies CSS into a temporary document. Give only this analysis step the response's nonce.
  // Its CSSOM reader reuses already loaded responses; UI requests still use the browser and real CSP.
  const restore = await page.evaluateHandle(({ styleNonce, sheets }) => {
    const original = Document.prototype.createElement
    const originalFetch = window.fetch
    const snapshot = new Map(sheets)
    Document.prototype.createElement = function (name, ...args) {
      const element = original.call(this, name, ...args)
      if (name.toLowerCase() === 'style') {
        element.nonce = styleNonce
      }
      return element
    }
    window.fetch = function (input, ...args) {
      const url = typeof input === 'string' ? input : input.url ?? String(input)
      if (snapshot.has(url)) {
        return Promise.resolve(new Response(snapshot.get(url), { headers: { 'Content-Type': 'text/css' } }))
      }
      return originalFetch.call(this, input, ...args)
    }
    return () => {
      Document.prototype.createElement = original
      window.fetch = originalFetch
    }
  }, { styleNonce: nonce, sheets: stylesheets })
  try {
    return await scan(page, info, options)
  }
  finally {
    await restore.evaluate(callback => callback())
    await restore.dispose()
  }
}

export const appbar = page => page.locator('.ztd-appbar')

export async function settleOverlay(page) {
  await page.locator('.ztd-menu').evaluateAll(async nodes => Promise.all(nodes.flatMap(node =>
    node.getAnimations().map(animation => animation.finished.catch(() => undefined)),
  )))
}

export async function chooseAppearance(page, name) {
  await appbar(page).getByRole('button', { name: /Appearance|外观/u }).press('Enter')
  await page.getByRole('menuitemradio', { name, exact: true }).focus()
  await page.keyboard.press('Enter')
  await page.getByRole('menu').waitFor({ state: 'detached' })
}
export async function chooseChinese(page) {
  await appbar(page).getByRole('combobox', { name: 'Language', exact: true }).press('Enter')
  await page.getByRole('option', { name: '简体中文', exact: true }).press('Enter')
  await expect(appbar(page).getByRole('combobox', { name: '语言', exact: true })).toBeFocused()
}

export function watchErrors(page) {
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', (message) => {
    if (message.type() === 'error') {
      errors.push(message.text())
    }
  })
  return errors
}
