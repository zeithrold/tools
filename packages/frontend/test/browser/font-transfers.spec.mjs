import process from 'node:process'
import { expect } from '@playwright/test'
import { transferProfile } from './font-transfer-profile.mjs'
import { test, watchErrors } from './helpers.mjs'

const localPreview = Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW)
const scenarios = [
  { scenario: 'english-shell', route: '/review', locale: 'en', coldCap: 500_000 },
  { scenario: 'chinese-page', route: '/review-zh', locale: 'zh-CN', coldCap: 1_000_000 },
  { scenario: 'full-specimen', route: '/fonts', locale: 'en' },
]

function summarize(resources) {
  const byFamily = {}
  for (const resource of resources) {
    const family = [
      ...new Set(resource.faces.map(face => face.family)),
    ].join(', ') || resource.kind
    const row = byFamily[family] ?? { requests: 0, cached: 0, httpResponseBytes: 0, httpDecodedBodyBytes: 0 }
    row.requests++
    row.cached += Number(resource.cached)
    row.httpResponseBytes += resource.httpResponseBytes
    row.httpDecodedBodyBytes += resource.httpDecodedBodyBytes ?? 0
    byFamily[family] = row
  }
  return byFamily
}

test('representative cold and warm font budgets with full specimen reporting', async ({ browser }, info) => {
  for (const { scenario, route, locale, coldCap } of scenarios) {
    const context = await browser.newContext()
    try {
      await context.addCookies([
        {
          name: 'harbor.ui.v1',
          value: encodeURIComponent(JSON.stringify({ version: 1, mode: 'light', palette: 'neutral', locale })),
          url: 'http://127.0.0.1:4317',
        },
      ])
      const page = await context.newPage()
      const errors = watchErrors(page)
      const sample = await transferProfile(page)
      for (const phase of ['cold', 'warm']) {
        const resources = await sample(phase, () => phase === 'cold'
          ? page.goto(`http://127.0.0.1:4317${route}`)
          : page.reload())
        expect(errors).toEqual([])
        expect(resources.every(resource => resource.status === 200 || resource.status === 304)).toBe(true)
        const fonts = resources.filter(resource => resource.kind === 'font')
        expect(fonts.length).toBeGreaterThan(0)
        expect(fonts.every(resource => resource.faces.length)).toBe(true)
        expect(resources.some(resource => resource.kind === 'font-css')).toBe(true)
        const httpResponseBytes = resources.reduce((total, resource) => total + resource.httpResponseBytes, 0)
        const phaseCap = phase === 'cold' ? coldCap : 10_000
        const cap = coldCap === undefined ? undefined : phaseCap
        const profile = {
          scenario,
          phase,
          localPreview,
          httpResponseBytes,
          budgetBytes: cap ?? null,
          remoteBudgetEnforced: !localPreview && cap !== undefined,
          byFamily: summarize(resources),
          resources,
        }
        await info.attach(`${scenario}-${phase}-font-profile`, {
          body: JSON.stringify(profile),
          contentType: 'application/json',
        })
        process.stdout.write(`FONT_TRANSFER_PROFILE ${JSON.stringify(profile)}\n`)
        // Local preview lacks Google's compression/cache headers and cannot verify remote budgets.
        if (!localPreview && cap !== undefined) {
          expect(httpResponseBytes).toBeLessThanOrEqual(cap)
        }
      }
    }
    finally {
      await context.close()
    }
  }
})
