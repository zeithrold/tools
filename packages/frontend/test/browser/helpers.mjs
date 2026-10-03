import { expect } from '@playwright/test'
import { assertAccessible as scan } from '@ztd-me/frontend-checks/playwright'

export async function assertAccessible(page, info, options) {
  const nonce = await page.locator('#initial').evaluate(node => JSON.parse(node.textContent).styleNonce)
  // Axe copies CSS into a temporary document. Give only this analysis step the response's nonce.
  // Restore the native method before further interaction; real UI injection stays independently checked.
  const restore = await page.evaluateHandle((styleNonce) => {
    const original = Document.prototype.createElement
    Document.prototype.createElement = function (name, ...args) {
      const element = original.call(this, name, ...args)
      if (name.toLowerCase() === 'style') {
        element.nonce = styleNonce
      }
      return element
    }
    return () => {
      Document.prototype.createElement = original
    }
  }, nonce)
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
