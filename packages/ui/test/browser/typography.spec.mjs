import { expect } from '@playwright/test'
import { assertAccessible, settleOverlay, test, watchErrors } from './helpers.mjs'

async function sizeRole(element, fontSize, lineHeight) {
  await expect(element).toHaveCSS('font-size', `${fontSize}px`)
  await expect(element).toHaveCSS('line-height', `${lineHeight}px`)
}

async function mergedRole(element, expected, replaced) {
  const classes = await element.evaluate(node => Array.from(node.classList))
  expect(classes).toContain(expected)
  expect(classes).not.toContain(replaced)
}

async function fieldRoles(page) {
  await sizeRole(page.getByLabel('Fixture name', { exact: true }), 16, 24)
  await sizeRole(page.getByLabel('Fixture note', { exact: true }), 16, 24)
  await sizeRole(page.getByLabel('Wrapped field', { exact: true }), 16, 24)
  await expect(page.getByLabel('Wrapped field', { exact: true })).toHaveCSS('font-weight', '400')
  const native = page.getByLabel('Native choice')
  await expect(native).toHaveCSS('font-size', '14px')
  // Chromium native select keeps its platform line height even when the role declares 1.5.
  await expect(native).toHaveCSS('line-height', 'normal')
  expect(await native.evaluate(node => node.getBoundingClientRect().height)).toBeGreaterThanOrEqual(44)
  await sizeRole(page.getByRole('combobox', { name: 'Enhanced choice' }), 14, 21)
  await expect(page.getByLabel('Disabled name')).toBeDisabled()
  await sizeRole(page.getByLabel('Disabled name'), 16, 24)
  await expect(page.getByRole('button', { name: 'Disabled submit' })).toBeDisabled()
  await page.getByRole('button', { name: 'Disabled submit' }).evaluate(node => node.click())
  await expect(page.locator('#submitted')).toBeEmpty()
}

async function controlTargets(page, coarse) {
  const defaults = page.getByRole('button', { name: 'Submit fixture', exact: true })
  await sizeRole(defaults, 14, 21)
  expect(await defaults.evaluate(node => node.getBoundingClientRect().height)).toBeGreaterThanOrEqual(44)
  for (const [name, fineSize] of [
    ['Extra small target', 32],
    ['Small target', 36],
    ['Extra small icon target', 32],
    ['Small icon target', 36],
  ]) {
    const button = page.getByRole('button', { name, exact: true })
    expect(await button.evaluate(node => node.getBoundingClientRect().height)).toBe(coarse ? 44 : fineSize)
    expect(await button.evaluate(node => node.getBoundingClientRect().width)).toBeGreaterThanOrEqual(44)
  }
  await page.getByRole('combobox', { name: 'Enhanced choice' }).press('Enter')
  await settleOverlay(page)
  for (const item of await page.getByRole('option').all()) {
    await sizeRole(item, 14, 21)
    if (coarse) {
      expect(await item.evaluate(node => node.getBoundingClientRect().height)).toBeGreaterThanOrEqual(44)
    }
  }
  await page.keyboard.press('Escape')
  await page.getByRole('listbox').waitFor({ state: 'detached' })
  await expect(page.getByRole('combobox', { name: 'Enhanced choice' })).toBeFocused()
  await expect(page.locator('[inert]')).toHaveCount(0)
}

for (const [width, coarse] of [
  [1440, false],
  [390, true],
]) {
  test.describe(`typography roles at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 }, hasTouch: coarse, isMobile: coarse })
    test(`roles preserve reading, inputs and ${coarse ? 'coarse' : 'fine'} targets`, async ({ page }, info) => {
      const errors = watchErrors(page)
      await page.goto('/primitives')
      await expect(page.locator('html')).toHaveCSS('font-size', '16px')
      await sizeRole(page.locator('#typography-body'), 16, 26.4)
      await sizeRole(page.locator('#typography-reading'), 18, 29.7)
      const sample = page.getByRole('region', { name: 'Typography roles', exact: true })
      await sizeRole(sample.locator('[data-slot="card-title"]'), 20, 27)
      await sizeRole(sample.locator('[data-slot="card-description"]'), 13, 19.5)
      await sizeRole(page.locator('#reading-class-override'), 18, 29.7)
      await sizeRole(page.locator('#input-class-override'), 16, 24)
      await sizeRole(page.locator('#reading-input-override'), 18, 29.7)
      await mergedRole(page.locator('#reading-class-override'), 'text-reading', 'text-control')
      await mergedRole(page.locator('#input-class-override'), 'text-editable', 'text-control')
      await mergedRole(page.locator('#reading-input-override'), 'text-reading', 'text-editable')
      await expect(page.locator('#reading-class-override')).toHaveCSS('color', 'rgb(23, 23, 23)')
      await fieldRoles(page)
      await controlTargets(page, coarse)
      await page.getByLabel('Fixture name', { exact: true }).fill('Preserved 中文')
      await expect(page.getByLabel('Fixture name', { exact: true })).toHaveValue('Preserved 中文')
      await page.getByRole('button', { name: 'Submit fixture', exact: true }).click()
      await expect(page.locator('#submitted')).toHaveText(JSON.stringify({
        name: 'Preserved 中文',
        note: 'Draft',
        choice: 'two',
        accepted: 'on',
      }))
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
      await assertAccessible(page, info)
      await page.screenshot({ path: info.outputPath(`typography-roles-${width}.png`), fullPage: true })
      expect(errors).toEqual([])
    })
  })
}

test('consumer typography tokens override each semantic role without changing rem or targets', async ({ page }) => {
  await page.goto('/primitives')
  await page.evaluate(() => {
    for (const [role, size] of [
      ['body', 17],
      ['control', 15],
      ['help', 14],
      ['heading', 21],
      ['reading', 19],
      ['input', 18],
    ]) {
      document.documentElement.style.setProperty(`--ztd-font-${role}`, `${size}px`)
    }
  })
  await expect(page.locator('html')).toHaveCSS('font-size', '16px')
  await sizeRole(page.locator('#typography-body'), 17, 28.05)
  await sizeRole(page.locator('#typography-reading'), 19, 31.35)
  const sample = page.getByRole('region', { name: 'Typography roles', exact: true })
  await sizeRole(sample.locator('[data-slot="card-title"]'), 21, 28.35)
  await sizeRole(sample.locator('[data-slot="card-description"]'), 14, 21)
  await sizeRole(page.getByLabel('Fixture name', { exact: true }), 18, 27)
  await sizeRole(page.getByRole('button', { name: 'Submit fixture', exact: true }), 15, 22.5)
  await sizeRole(page.getByRole('combobox', { name: 'Enhanced choice' }), 15, 22.5)
  await sizeRole(page.locator('#reading-class-override'), 19, 31.35)
  await sizeRole(page.locator('#input-class-override'), 18, 27)
  await sizeRole(page.locator('#reading-input-override'), 19, 31.35)
  for (const element of [
    page.getByRole('button', { name: 'Submit fixture', exact: true }),
    page.getByLabel('Fixture name', { exact: true }),
  ]) {
    expect(await element.evaluate(node => node.getBoundingClientRect().height)).toBeGreaterThanOrEqual(44)
  }
})

test('dialog and drawer use compact interface titles while editable text stays 16px', async ({ page }, info) => {
  const errors = watchErrors(page)
  await page.setViewportSize({ width: 390, height: 900 })
  await page.goto('/primitives')
  for (const [kind, name] of [
    ['dialog', 'Fixture dialog'],
    ['drawer', 'Fixture drawer'],
  ]) {
    const trigger = page.getByRole('button', { name: `Open ${kind}`, exact: true })
    await trigger.press('Enter')
    const panel = page.getByRole('dialog', { name, exact: true })
    await sizeRole(panel, 16, 26.4)
    await sizeRole(panel.locator(`[data-slot="${kind}-title"]`), 20, 27)
    await sizeRole(panel.locator(`[data-slot="${kind}-description"]`), 13, 19.5)
    await sizeRole(panel.getByRole('textbox'), 16, 24)
    await expect(panel.getByRole('textbox')).toBeFocused()
    expect(await panel.evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true)
    await assertAccessible(page, info)
    await page.screenshot({ path: info.outputPath(`typography-${kind}-390.png`) })
    await page.keyboard.press('Escape')
    await expect(panel).toBeHidden()
    await expect(trigger).toBeFocused()
  }
  expect(errors).toEqual([])
})
