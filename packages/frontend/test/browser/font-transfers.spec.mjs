import process from 'node:process'
import { expect } from '@playwright/test'
import { transferProfile } from './font-transfer-profile.mjs'
import { test, watchErrors } from './helpers.mjs'

const scenarios = [
  [
    'english-shell',
    '/review',
    'en',
  ],
  [
    'chinese-page',
    '/review-zh',
    'zh-CN',
  ],
  [
    'full-specimen',
    '/fonts',
    'en',
  ],
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

test('bounded cold and warm font profiles for English, Chinese and full specimen', async ({ browser }, info) => {
  const profiles = []
  for (const [scenario, route, locale] of scenarios) {
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
        expect(fonts.every(resource => resource.faces.length)).toBe(true)
        const profile = {
          scenario,
          phase,
          localPreview: Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW),
          byFamily: summarize(resources),
          resources,
        }
        profiles.push(profile)
        // Bounded diagnostic evidence, not a proposed product budget or a font/network substitute.
        process.stdout.write(`FONT_TRANSFER_PROFILE ${JSON.stringify(profile)}\n`)
      }
    }
    finally {
      await context.close()
    }
  }
  await info.attach('cold-warm-font-profiles', { body: JSON.stringify(profiles), contentType: 'application/json' })
})
