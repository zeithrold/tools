import { expect, test } from '@playwright/test'
import { assertAccessible } from '@ztd-me/frontend-checks/playwright'
import { appbar, chooseAppearance, chooseChinese, watchErrors } from './helpers.mjs'

async function denyStorage(context, afterFirstRead = false) {
  await context.addInitScript((delayed) => {
    const nativeCookie = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie')
    let reads = 0
    Object.defineProperty(Document.prototype, 'cookie', {
      configurable: true,
      get: () => {
        reads += 1
        if (delayed && reads === 1) {
          return ''
        }
        throw new DOMException('Cookie access denied', 'SecurityError')
      },
      set: () => {
        throw new DOMException('Cookie access denied', 'SecurityError')
      },
    })
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get: () => {
        throw new DOMException('Storage access denied', 'SecurityError')
      },
    })
    window.restoreCookieAccess = () => Object.defineProperty(Document.prototype, 'cookie', nativeCookie)
  }, afterFirstRead)
}

for (const width of [390, 1280]) {
  test(`denied cookie reads preserve SSR, controls and recovery at width ${width}`, async ({ context, page }, info) => {
    await context.addCookies([
      {
        name: 'harbor.ui.v1',
        value: encodeURIComponent(JSON.stringify({ version: 1, mode: 'dark', palette: 'ocean', locale: 'en' })),
        url: 'http://127.0.0.1:4317',
      },
    ])
    await denyStorage(context)
    await page.setViewportSize({ width, height: 900 })
    const errors = watchErrors(page)
    const response = await page.goto('/application')
    expect(await response.text()).toContain('data-frontend-palette="ocean"')
    await expect(page.getByRole('heading', { name: 'Shared shell fixture' })).toBeVisible()
    await expect(page.locator('#persistence')).toHaveText('unavailable')
    await expect(page.locator('#persistence-errors')).toHaveText('1')
    await page.getByRole('textbox', { name: 'Business draft' }).fill('Unsaved recovery draft')
    await chooseAppearance(page, 'Light')
    await chooseChinese(page)
    await expect(page.locator('#resolved')).toHaveText('light')
    await page.evaluate(() => {
      window.dispatchEvent(new Event('focus'))
      window.dispatchEvent(new StorageEvent('storage', { key: 'harbor.ui.notification' }))
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await page.emulateMedia({ colorScheme: 'dark' })
    await expect(appbar(page).getByRole('combobox', { name: '语言' })).toHaveText(/简体中文/u)
    await expect(page.getByRole('textbox', { name: 'Business draft' })).toHaveValue('Unsaved recovery draft')
    await assertAccessible(page, info)
    await page.evaluate(() => {
      window.restoreCookieAccess()
      window.dispatchEvent(new Event('focus'))
    })
    await expect(page.locator('#persistence')).toHaveText('saved')
    await expect(page.locator('html')).toHaveAttribute('data-frontend-palette', 'ocean')
    await chooseAppearance(page, 'Moss')
    await chooseChinese(page)
    await expect(page.locator('#persistence')).toHaveText('saved')
    await expect(page.getByRole('textbox', { name: 'Business draft' })).toHaveValue('Unsaved recovery draft')
    expect(errors).toEqual([])
  })
}

test('later cookie denial preserves the application', async ({ context, page }) => {
  await denyStorage(context, true)
  const errors = watchErrors(page)
  await page.goto('/application')
  await page.evaluate(() => window.dispatchEvent(new Event('focus')))
  await expect(page.getByRole('heading', { name: 'Shared shell fixture' })).toBeVisible()
  await chooseAppearance(page, 'Dark')
  await expect(page.locator('#resolved')).toHaveText('dark')
  await expect(page.locator('#persistence')).toHaveText('unavailable')
  expect(errors).toEqual([])
})
