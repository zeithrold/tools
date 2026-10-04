import { expect } from '@playwright/test'
import { appbar, assertAccessible, chooseAppearance, chooseChinese, test, watchErrors } from './helpers.mjs'

test('unowned consumer storage and business data stay untouched', async ({ context, page }, info) => {
  await context.addInitScript(() => {
    const old = { theme: 'dark', palette: 'moss', locale: 'zh-CN', workspace: 'private', auth: 'opaque' }
    for (const key of [
      'consumer.preferences.v0',
      'consumer.clock.v0',
      'consumer.auth.session',
    ]) {
      localStorage.setItem(key, JSON.stringify(old))
    }
    document.cookie = 'locale=zh-CN; Path=/'
    document.cookie = 'business.session=opaque; Path=/'
    const read = Storage.prototype.getItem
    const remove = Storage.prototype.removeItem
    window.storageReads = []
    window.storageRemovals = []
    Storage.prototype.getItem = function (key) {
      window.storageReads.push(key)
      return read.call(this, key)
    }
    Storage.prototype.removeItem = function (key) {
      window.storageRemovals.push(key)
      return remove.call(this, key)
    }
    window.readStored = key => read.call(localStorage, key)
  })
  await page.goto('/?mirror=off')
  await expect(page.locator('html')).toHaveAttribute('data-frontend-palette', 'neutral')
  await expect(appbar(page).getByRole('combobox', { name: 'Language' })).toBeVisible()
  expect((await context.cookies()).some(cookie => cookie.name === 'harbor.ui.v1')).toBe(false)
  await chooseAppearance(page, 'Dark')
  await chooseChinese(page)
  const values = await page.evaluate(() => ({
    reads: window.storageReads,
    removals: window.storageRemovals,
    clock: JSON.parse(window.readStored('consumer.clock.v0')),
    preferences: JSON.parse(window.readStored('consumer.preferences.v0')),
    auth: JSON.parse(window.readStored('consumer.auth.session')),
  }))
  expect(values.reads).toEqual([])
  expect(values.removals).toEqual([])
  expect(values.clock).toEqual(values.preferences)
  expect(values.auth).toEqual(values.clock)
  expect(values.clock).toEqual({
    theme: 'dark',
    palette: 'moss',
    locale: 'zh-CN',
    workspace: 'private',
    auth: 'opaque',
  })
  const cookies = await context.cookies()
  expect(cookies.find(cookie => cookie.name === 'locale').value).toBe('zh-CN')
  expect(cookies.find(cookie => cookie.name === 'business.session').value).toBe('opaque')
  expect(JSON.parse(decodeURIComponent(cookies.find(cookie => cookie.name === 'harbor.ui.v1').value)).mode).toBe('dark')
  await assertAccessible(page, info)
})

test('no mirror means no localStorage access while cookies and controls still work', async ({ context, page }) => {
  await context.addInitScript(() => {
    window.storageAccesses = 0
    Object.defineProperty(window, 'localStorage', {
      get: () => {
        window.storageAccesses += 1
        throw new DOMException('Unavailable storage', 'SecurityError')
      },
    })
  })
  const errors = watchErrors(page)
  await page.goto('/?mirror=off&footer=none')
  await expect(page.locator('footer')).toHaveCount(0)
  await chooseAppearance(page, 'Plum')
  await expect(page.locator('#persistence')).toHaveText('saved')
  expect(await page.evaluate(() => window.storageAccesses)).toBe(0)
  expect(errors).toEqual([])
})

test('external consumer owns footer links without a repository-host restriction', async ({ page }, info) => {
  await page.goto('/')
  await expect(page.getByRole('link', { name: 'Studio source' })).toHaveAttribute('href', 'https://codeberg.org/example/studio')
  const contact = page.getByRole('link', { name: 'support@harbor.example' })
  await expect(contact).toHaveAttribute('href', 'mailto:support@harbor.example')
  await assertAccessible(page, info)
})
