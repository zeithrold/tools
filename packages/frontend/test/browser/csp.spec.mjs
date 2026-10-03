import process from 'node:process'
import { expect } from '@playwright/test'
import { assertAccessible, test, watchErrors } from './helpers.mjs'

async function attemptFontCssRead(page, url) {
  return page.evaluate(source => new Promise((resolve) => {
    const request = new XMLHttpRequest()
    request.open('GET', source)
    request.addEventListener('load', () => resolve(true))
    request.addEventListener('error', () => resolve(false))
    request.send()
  }), url)
}

test('real font CSS loads while application XHR remains blocked before and after Axe', async ({ page }, info) => {
  test.skip(Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW), 'Requires actual Google Fonts browser delivery')
  const errors = watchErrors(page)
  await page.goto('/fonts')
  await page.evaluate(async () => document.fonts.ready)
  const url = await page.evaluate(() => {
    const entries = performance.getEntriesByType('resource')
    const stylesheet = entries.find(entry => new URL(entry.name).origin === 'https://fonts.googleapis.com')
    return stylesheet?.name
  })
  expect(url).toBeTruthy()
  expect(errors).toEqual([])
  expect(await attemptFontCssRead(page, url)).toBe(false)
  expect(errors).toHaveLength(1)
  expect(errors[0]).toContain('connect-src')
  errors.length = 0
  const results = await assertAccessible(page, info)
  expect(results.incomplete).toEqual([])
  expect(errors).toEqual([])
  expect(await attemptFontCssRead(page, url)).toBe(false)
  expect(errors).toHaveLength(1)
  expect(errors[0]).toContain('connect-src')
})
