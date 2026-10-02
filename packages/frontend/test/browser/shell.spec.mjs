import { expect, test } from '@playwright/test'
import { assertAccessible, captureState } from '@ztd-me/frontend-checks/playwright'

const developmentKey = 'ztd.frontend.development.website.v1'
const preferences = { version: 1, mode: 'dark', palette: 'ocean', locale: 'zh-CN' }
const appbar = page => page.locator('.ztd-appbar')

async function chooseAppearance(page, name) {
  await appbar(page).getByRole('button', { name: /Appearance|外观/u }).press('Enter')
  await page.getByRole('menuitemradio', { name, exact: true }).focus()
  await page.keyboard.press('Enter')
}
async function chooseChinese(page) {
  await appbar(page).getByRole('combobox', { name: 'Language', exact: true }).press('Enter')
  await page.getByRole('option', { name: '简体中文', exact: true }).press('Enter')
  await expect(appbar(page).getByRole('combobox', { name: '语言', exact: true })).toBeFocused()
}

function watchErrors(page) {
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', (message) => {
    if (message.type() === 'error') {
      errors.push(message.text())
    }
  })
  return errors
}

test('SSR cookie snapshot matches hydrated locale and theme', async ({ context, page }, info) => {
  await context.addCookies([
    { name: developmentKey, value: encodeURIComponent(JSON.stringify(preferences)), url: 'http://127.0.0.1:4317' },
  ])
  const errors = watchErrors(page)
  const response = await page.goto('/')
  const html = await response.text()
  expect(html).toContain('data-frontend-mode="dark"')
  expect(html).toContain('data-frontend-palette="ocean"')
  expect(html).toContain('lang="zh-CN"')
  await expect(appbar(page).getByRole('combobox', { name: '语言', exact: true })).toHaveText(/简体中文/u)
  await expect(page.locator('#resolved')).toHaveText('dark')
  await assertAccessible(page, info)
  await captureState(page, info, 'ssr-chinese-dark-ocean')
  await page.evaluate(async () => document.fonts.ready)
  expect(await page.evaluate(() => document.fonts.check('14px "Inter Variable"', 'Appearance'))).toBe(true)
  expect(errors).toEqual([])
})

test('system first paint works under CSP before hydration; every explicit mode stays independent', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.route('**/assets/*.js', route => route.abort())
  await page.goto('/')
  const background = await appbar(page).evaluate(node => getComputedStyle(node).backgroundColor)
  expect(background).toBe('rgb(23, 23, 23)')
  await page.unroute('**/assets/*.js')
  await page.reload()
  await expect(page.locator('#resolved')).toHaveText('dark')
  await chooseAppearance(page, 'Light')
  await page.emulateMedia({ colorScheme: 'light' })
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('#resolved')).toHaveText('light')
  await chooseAppearance(page, 'System')
  await expect(page.locator('#resolved')).toHaveText('dark')
  await page.emulateMedia({ colorScheme: 'light' })
  await expect(page.locator('#resolved')).toHaveText('light')
})

test('keyboard navigation restores focus and inert, preserving business drafts', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.goto('/')
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.locator('#ztd-main')).toBeFocused()
  await page.getByRole('textbox', { name: 'Business draft' }).fill('Unsaved business draft')
  await appbar(page).getByRole('button', { name: 'Appearance' }).press('Enter')
  await expect(page.getByRole('menu')).toBeVisible()
  await page.keyboard.press('End')
  await expect(page.getByRole('menuitemradio', { name: 'Graphite', exact: true })).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(appbar(page).getByRole('button', { name: 'Appearance' })).toBeFocused()
  await appbar(page).getByRole('combobox', { name: 'Language' }).press('Enter')
  await expect(page.getByRole('listbox')).toBeVisible()
  await assertAccessible(page, info)
  await expect(page.locator('[data-aria-hidden="true"]').first()).toHaveJSProperty('inert', true)
  await page.keyboard.press('End')
  await page.keyboard.press('Enter')
  await expect(appbar(page).getByRole('combobox', { name: '语言' })).toBeFocused()
  await expect(page.locator('[inert]')).toHaveCount(0)
  await expect(page.getByRole('textbox', { name: 'Business draft' })).toHaveValue('Unsaved business draft')
  await page.getByRole('link', { name: 'Second route' }).click()
  await expect(appbar(page).getByRole('combobox', { name: '语言' })).toHaveText(/简体中文/u)
  expect(errors).toEqual([])
})

test('all six palettes and both modes are accessible, including open appearance menu', async ({ page }, info) => {
  test.setTimeout(60_000)
  await page.goto('/')
  const palettes = [
    'Neutral',
    'Terracotta',
    'Moss',
    'Ocean',
    'Plum',
    'Graphite',
  ]
  for (const mode of ['Light', 'Dark']) {
    await chooseAppearance(page, mode)
    for (const palette of palettes) {
      await chooseAppearance(page, palette)
      await expect(page.locator('html')).toHaveAttribute('data-frontend-palette', palette.toLowerCase())
      await assertAccessible(page, info)
      await captureState(page, info, `${mode.toLowerCase()}-${palette.toLowerCase()}`)
      await appbar(page).getByRole('button', { name: 'Appearance' }).press('Enter')
      await assertAccessible(page, info, { label: 'appearance' })
      await page.keyboard.press('Escape')
      await appbar(page).getByRole('combobox', { name: 'Language' }).press('Enter')
      await assertAccessible(page, info, { label: 'locale' })
      await page.keyboard.press('Escape')
    }
  }
  await appbar(page).getByRole('button', { name: 'Appearance' }).press('Enter')
  await assertAccessible(page, info)
  await captureState(page, info, 'appearance-menu')
})

test('Chinese controls and fixed footer reflow at narrow widths without adding navigation', async ({ page }, info) => {
  await page.goto('/')
  await chooseChinese(page)
  for (const width of [
    320,
    390,
    768,
    1024,
    1440,
  ]) {
    await page.setViewportSize({ width, height: 900 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
    await expect(page.locator('.ztd-appbar nav')).toHaveCount(0)
    await expect(page.locator('footer')).toHaveText(/© ZeithroldGitHubhello@ztd\.me/u)
  }
  await page.setViewportSize({ width: 390, height: 844 })
  await assertAccessible(page, info)
  await captureState(page, info, 'chinese-mobile')
})

test('storage rejection leaves choices usable in memory and reports failed persistence', async ({ context, page }) => {
  await context.addInitScript(() => {
    Object.defineProperty(Document.prototype, 'cookie', { configurable: true, get: () => '', set: () => {} })
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get: () => {
        throw new Error('Denied')
      },
    })
  })
  await page.goto('/')
  await chooseAppearance(page, 'Dark')
  await expect(page.locator('#resolved')).toHaveText('dark')
  await expect(page.locator('#persistence')).toHaveText('unavailable')
  await chooseChinese(page)
  await expect(appbar(page).getByRole('combobox', { name: '语言' })).toHaveText(/简体中文/u)
})

test('legacy migration preserves business storage', async ({ context, page }) => {
  const old = {
    theme: 'dark',
    palette: 'moss',
    locale: 'zh-CN',
    timezone: 'UTC',
    seconds: true,
    auth: 'opaque',
  }
  await context.addInitScript((value) => {
    localStorage.setItem('showcase.clock.v1', JSON.stringify(value))
  }, old)
  await page.goto('/?project=showcase')
  await expect(page.locator('html')).toHaveAttribute('data-frontend-palette', 'moss')
  const cookies = await context.cookies()
  const shared = cookies.find(cookie => cookie.name === 'ztd.frontend.development.showcase.v1')
  expect(JSON.parse(decodeURIComponent(shared.value))).toEqual({
    version: 1,
    mode: 'dark',
    palette: 'moss',
    locale: 'zh-CN',
  })
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem('showcase.clock.v1')))).toEqual(old)
})

test('production subdomains share preferences while preview stays isolated', async ({ context, page }) => {
  await context.route('https://*.ztd.me/**', async (route) => {
    const url = new URL(route.request().url())
    const local = new URL(url.pathname + url.search, 'http://127.0.0.1:4317')
    local.searchParams.set('environment', url.hostname === 'preview.ztd.me' ? 'preview' : 'production')
    local.searchParams.set('hostname', url.hostname)
    local.searchParams.set('project', url.hostname === 'showcase.ztd.me' ? 'showcase' : 'website')
    const response = await route.fetch({ url: local.toString() })
    await route.fulfill({ response })
  })
  await page.goto('https://website.ztd.me/')
  await chooseAppearance(page, 'Dark')
  await chooseAppearance(page, 'Ocean')
  await chooseChinese(page)
  const second = await context.newPage()
  await second.goto('https://showcase.ztd.me/')
  await expect(second.locator('html')).toHaveAttribute('data-frontend-palette', 'ocean')
  await expect(appbar(second).getByRole('combobox', { name: '语言' })).toHaveText(/简体中文/u)
  await chooseAppearance(second, '莓紫')
  await page.evaluate(() => window.dispatchEvent(new Event('focus')))
  await expect(page.locator('html')).toHaveAttribute('data-frontend-palette', 'plum')
  const preview = await context.newPage()
  await preview.goto('https://preview.ztd.me/')
  await expect(preview.locator('html')).toHaveAttribute('data-frontend-palette', 'neutral')
  await chooseAppearance(preview, 'Moss')
  const cookies = await context.cookies()
  const production = cookies.find(cookie => cookie.name === 'ztd.frontend.v1')
  expect(production.domain).toBe('.ztd.me')
  expect(JSON.parse(decodeURIComponent(production.value)).palette).toBe('plum')
  expect(cookies.find(cookie => cookie.name === 'ztd.frontend.preview.website.v1').domain).toBe('preview.ztd.me')
})

test('same-origin tabs respond to the optional mirror notification', async ({ context, page }) => {
  await page.goto('/')
  const second = await context.newPage()
  await second.goto('/')
  await chooseAppearance(page, 'Moss')
  await expect(second.locator('html')).toHaveAttribute('data-frontend-palette', 'moss')
})

test('fullscreen portal stays in the shell and restores Select focus', async ({ page }, info) => {
  await page.goto('/')
  await page.getByRole('button', { name: 'Fullscreen', exact: true }).click()
  await expect(page.locator('#fullscreen-shell')).toHaveJSProperty('id', 'fullscreen-shell')
  expect(await page.evaluate(() => document.fullscreenElement?.id)).toBe('fullscreen-shell')
  await appbar(page).getByRole('combobox', { name: 'Language', exact: true }).press('Enter')
  await expect(page.locator('#fullscreen-shell [role="listbox"]')).toBeVisible()
  await assertAccessible(page, info)
  await page.keyboard.press('Escape')
  await expect(appbar(page).getByRole('combobox', { name: 'Language', exact: true })).toBeFocused()
  await expect(page.locator('[inert]')).toHaveCount(0)
  await page.evaluate(async () => document.exitFullscreen())
})

test('application slots preserve project navigation and one main landmark', async ({ page }, info) => {
  await page.goto('/application')
  await expect(page.getByRole('navigation', { name: 'Business' })).toHaveText('Workspace')
  await expect(page.getByRole('complementary', { name: 'Context' })).toHaveText('Project sidebar')
  await expect(page.getByRole('main')).toHaveCount(1)
  await expect(page.getByRole('link', { name: 'Fixture', exact: true })).toHaveAttribute('data-router-link', 'fixture')
  await expect(page.locator('p[role="status"]')).toHaveText('Synthetic service notice')
  await assertAccessible(page, info)
})
