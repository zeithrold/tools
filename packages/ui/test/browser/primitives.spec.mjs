import { expect } from '@playwright/test'
import { assertAccessible, chooseAppearance, chooseChinese, settleOverlay, test, watchErrors } from './helpers.mjs'

test('native fields submit and injected locale changes preserve input', async ({ page }) => {
  const errors = watchErrors(page)
  await page.goto('/primitives')
  await expect(page.locator('#injected-locale')).toHaveText('Ready 1')
  await page.getByLabel('Fixture name', { exact: true }).fill('Retained draft')
  await page.getByLabel('Native choice').selectOption('one')
  await page.getByRole('checkbox', { name: 'Accepted' }).press('Space')
  await page.getByRole('button', { name: 'Submit fixture' }).click()
  await expect(page.locator('#submitted')).toHaveText('{"name":"Retained draft","note":"Draft","choice":"one"}')
  await chooseChinese(page)
  await expect(page.locator('#injected-locale')).toHaveText('已就绪 1')
  await expect(page.getByLabel('Fixture name', { exact: true })).toHaveValue('Retained draft')
  expect(errors).toEqual([])
})
test('button variants show a semantic hover surface', async ({ page }) => {
  await page.goto('/primitives')
  for (const name of [
    'Submit fixture',
    'Outline fixture',
    'Ghost fixture',
  ]) {
    const button = page.getByRole('button', { name, exact: true })
    const rest = await button.evaluate(node => getComputedStyle(node).backgroundColor)
    await button.hover()
    await expect.poll(() => button.evaluate(node => getComputedStyle(node).backgroundColor)).not.toBe(rest)
    expect(await button.evaluate(node => getComputedStyle(node).filter)).toBe('none')
    await page.mouse.move(0, 0)
  }
})

test('disclosures and tabs expose keyboard state and associated content', async ({ page }) => {
  await page.goto('/primitives')
  const details = page.getByRole('button', { name: 'Fixture details' })
  await details.press('Enter')
  await expect(details).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByText('Expanded details', { exact: true })).toBeVisible()
  await details.press('Space')
  await expect(details).toHaveAttribute('aria-expanded', 'false')
  await page.getByRole('button', { name: 'Toggle section' }).press('Enter')
  await expect(page.getByText('Visible section', { exact: true })).toBeVisible()
  const tab = page.getByRole('tab', { name: 'First tab' })
  expect(await tab.evaluate(node => getComputedStyle(node).display)).toMatch(/^(?:inline-)?flex$/u)
  expect(await tab.evaluate(node => getComputedStyle(node).flexDirection)).toBe('row')
  const icon = await tab.locator('svg').boundingBox()
  const label = await tab.locator('span').boundingBox()
  expect(icon.y + icon.height / 2).toBeCloseTo(label.y + label.height / 2, 0)
  expect(icon.x + icon.width).toBeLessThan(label.x)
  await tab.focus()
  await page.keyboard.press('ArrowRight')
  await expect(page.getByRole('tab', { name: 'Second tab' })).toBeFocused()
  await expect(page.getByRole('tabpanel')).toHaveText('Second panel')
})
test('modal and confirmation focus, escape and portal ownership work under nonce CSP', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.goto('/primitives')
  const trigger = page.getByRole('button', { name: 'Open dialog' })
  await trigger.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'Fixture dialog' })
  await expect(dialog).toBeVisible()
  await expect(dialog.locator('[data-slot=dialog-title]')).toHaveText('Fixture dialog')
  await expect(dialog.locator('[data-slot=dialog-description]')).toHaveText('Focusable content')
  await expect(page.getByLabel('Dialog input')).toBeFocused()
  expect(await dialog.evaluate(node => Boolean(node.closest('#fullscreen-shell')))).toBe(true)
  await page.keyboard.press('Shift+Tab')
  await expect(page.getByRole('button', { name: 'Close dialog' })).toBeFocused()
  await dialog.evaluate(async node => Promise.all(node.getAnimations().map(animation => animation.finished)))
  await assertAccessible(page, info)
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(trigger).toBeFocused()
  await page.getByRole('button', { name: 'Open confirmation' }).press('Enter')
  await expect(page.getByRole('button', { name: 'Cancel action' })).toBeFocused()
  await page.getByRole('button', { name: 'Cancel action' }).press('Enter')
  await expect(page.getByRole('button', { name: 'Open confirmation' })).toBeFocused()
  await page.getByRole('button', { name: 'Open sheet' }).press('Enter')
  await expect(page.getByRole('dialog', { name: 'Fixture sheet' })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('button', { name: 'Open sheet' })).toBeFocused()
  expect(errors).toEqual([])
})
test('enhanced select, tooltip and native scrolling retain keyboard and CSP behavior', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.goto('/primitives')
  const select = page.getByRole('combobox', { name: 'Enhanced choice' })
  await select.press('Enter')
  const selected = page.getByRole('option', { name: 'First choice' })
  await settleOverlay(page)
  const gap = await selected.evaluate((node) => {
    const row = node.getBoundingClientRect()
    const indicator = node.querySelector('.ztd-indicator').getBoundingClientRect()
    return row.right - indicator.right
  })
  expect(gap).toBeGreaterThanOrEqual(11)
  expect(gap).toBeLessThanOrEqual(13)
  expect(await selected.evaluate(node => getComputedStyle(node).outlineStyle)).toBe('none')
  await page.getByRole('option', { name: 'Second choice' }).press('Enter')
  await expect(select).toHaveText('Second choice')
  await expect(select).toBeFocused()
  const viewport = page.getByRole('region', { name: 'Scrollable sample' })
  expect(await viewport.evaluate(node => node.clientHeight)).toBe(120)
  expect(await viewport.evaluate(node => node.scrollHeight > node.clientHeight)).toBe(true)
  await viewport.focus()
  await page.keyboard.press('PageDown')
  await expect.poll(() => viewport.evaluate(node => node.scrollTop)).toBeGreaterThan(0)
  const nonce = await page.locator('#initial').evaluate(node => JSON.parse(node.textContent).styleNonce)
  expect(await page.locator('.ztd-scroll-area style').evaluate(node => node.nonce)).toBe(nonce)
  await page.getByRole('button', { name: 'Tooltip anchor' }).focus()
  await expect(page.getByRole('tooltip')).toHaveText('Fixture tooltip')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('tooltip')).toBeHidden()
  await assertAccessible(page, info)
  await page.screenshot({ path: info.outputPath('primitives.png'), fullPage: true })
  expect(errors).toEqual([])
})

test('shared controls retain state, touch sizing and class overrides', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.goto('/primitives')
  const setting = page.getByRole('button', { name: /Settings sample/u })
  await setting.press('Enter')
  await expect(page.getByLabel('Settings input')).toHaveValue('Preserved setting')
  await page.getByRole('switch', { name: 'Shared switch' }).press('Space')
  await expect(page.getByRole('switch', { name: 'Shared switch' })).not.toBeChecked()
  const sample = page.getByRole('group', { name: 'Sample colors' })
  await sample.getByRole('button', { name: 'Accent sample' }).press('Enter')
  await expect(page.locator('#selected-color')).toHaveText('accent')
  await expect(sample.getByRole('button', { name: 'Accent sample' })).toHaveAttribute('aria-pressed', 'true')
  const automatic = sample.getByRole('button', { name: 'Automatic' })
  expect(await automatic.evaluate(n => n.getBoundingClientRect().height)).toBe(44)
  const compact = page.getByRole('button', { name: 'Compact override' })
  expect(await compact.evaluate(n => n.getBoundingClientRect().height)).toBe(32)
  expect(await compact.evaluate(n => getComputedStyle(n).fontSize)).toBe('14px')
  const wrapped = page.getByRole('textbox', { name: 'Wrapped field' })
  expect(await wrapped.evaluate(n => getComputedStyle(n).fontWeight)).toBe('400')
  await assertAccessible(page, info)
  expect(errors).toEqual([])
})
test('shared drawer retains focus, CSP, fullscreen portal and reduced motion', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.setViewportSize({ width: 320, height: 844 })
  await page.goto('/primitives')
  const trigger = page.getByRole('button', { name: 'Open drawer' })
  await trigger.press('Enter')
  const drawer = page.getByRole('dialog', { name: 'Fixture drawer' })
  await expect(drawer).toBeVisible()
  await expect(drawer.locator('[data-slot=drawer-handle]')).toHaveCount(1)
  await expect(page.getByLabel('Drawer input')).toBeFocused()
  const inheritedPortal = await drawer.evaluate(n => Boolean(n.closest('#fullscreen-shell')))
  expect(inheritedPortal).toBe(true)
  expect(await drawer.evaluate(n => getComputedStyle(n).animationName)).toBe('none')
  await page.keyboard.press('Shift+Tab')
  await expect(page.getByRole('button', { name: 'Close drawer' })).toBeFocused()
  await assertAccessible(page, info)
  await page.keyboard.press('Escape')
  await expect(drawer).toBeHidden()
  await expect(trigger).toBeFocused()
  expect(errors).toEqual([])
})

test('destructive notices and controls remain accessible in system and explicit dark', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.goto('/primitives')
  const button = page.getByRole('button', { name: 'Destructive fixture' })
  const notice = page.getByText('Destructive notice', { exact: true })
  for (const [mode, background, foreground] of [
    [
      'System',
      'rgb(252, 165, 165)',
      'rgb(24, 24, 24)',
    ],
    [
      'Light',
      'rgb(185, 28, 28)',
      'rgb(255, 255, 255)',
    ],
    [
      'Dark',
      'rgb(252, 165, 165)',
      'rgb(24, 24, 24)',
    ],
  ]) {
    await chooseAppearance(page, mode)
    await expect(button).toHaveCSS('background-color', background)
    await expect(button).toHaveCSS('color', foreground)
    await expect(notice).toHaveCSS('color', background)
    await assertAccessible(page, info)
  }
  expect(errors).toEqual([])
})
