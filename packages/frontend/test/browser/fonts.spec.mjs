import process from 'node:process'
import { expect } from '@playwright/test'
import { captureState } from '@ztd-me/frontend-checks/playwright'
import { assertAccessible, test, watchErrors } from './helpers.mjs'

const localPreview = Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW)

async function renderedFonts(session, selector) {
  const { root } = await session.send('DOM.getDocument')
  const { nodeId } = await session.send('DOM.querySelector', { nodeId: root.nodeId, selector })
  return (await session.send('CSS.getPlatformFontsForNode', { nodeId })).fonts
}

async function renderedWeight(page) {
  return page.locator('#font-bold strong').evaluate((node) => {
    const canvas = document.createElement('canvas').getContext('2d')
    const widths = [400, 600].map((weight) => {
      canvas.font = `${weight} 24px "Noto Sans"`
      return canvas.measureText('English Noto weight').width
    })
    const faces = Array.from(document.fonts).filter(face => face.weight === '600' && face.status === 'loaded')
    return {
      weight: getComputedStyle(node).fontWeight,
      synthesis: getComputedStyle(node).fontSynthesis,
      faces: faces.map(face => ({ family: face.family, weight: face.weight })),
      widths,
    }
  })
}

test('English and CJK use Google Noto with real weights and bounded downloads', async ({ page, context }, info) => {
  const errors = watchErrors(page)
  const requests = []
  page.on('request', request => requests.push(request.url()))
  await page.goto('/fonts')
  await page.evaluate(async () => document.fonts.ready)
  const session = await context.newCDPSession(page)
  await session.send('DOM.enable')
  await session.send('CSS.enable')
  const evidence = []
  for (const [id, family] of [
    ['font-en', 'Noto Sans'],
    ['font-zh', 'Noto Sans SC'],
    ['font-ja', 'Noto Sans JP'],
    ['font-ko', 'Noto Sans KR'],
    ['font-bold strong', 'Noto Sans SC'],
  ]) {
    const fonts = await renderedFonts(session, `#${id}`)
    await info.attach(id, { body: JSON.stringify(fonts), contentType: 'application/json' })
    expect(fonts.length).toBeGreaterThan(0)
    expect(fonts.every(font => font.isCustomFont && font.familyName.startsWith('Noto Sans'))).toBe(true)
    expect(fonts.some(font => font.familyName.startsWith(family) && font.glyphCount > 0)).toBe(true)
    evidence.push({ id, fonts })
  }
  const weight = await renderedWeight(page)
  expect(weight).toMatchObject({ weight: '600', synthesis: 'none' })
  expect(weight.faces.some(face => face.family.includes('Noto Sans SC'))).toBe(true)
  expect(weight.widths[1]).not.toBe(weight.widths[0])
  const transfers = await page.evaluate(() => performance.getEntriesByType('resource')
    .filter(entry => entry.name.endsWith('.woff2'))
    .map(entry => ({ url: entry.name, bytes: entry.transferSize })))
  expect(transfers.length).toBeGreaterThanOrEqual(4)
  expect(transfers.length).toBeLessThan(80)
  expect(transfers.reduce((total, entry) => total + entry.bytes, 0)).toBeLessThan(2_000_000)
  if (localPreview) {
    expect(requests.some(url => url.includes('/__local-noto-preview/fonts.css'))).toBe(true)
    expect(requests.some(url => new URL(url).origin.startsWith('https://fonts.'))).toBe(false)
  }
  else {
    expect(requests.some(url => new URL(url).origin === 'https://fonts.googleapis.com')).toBe(true)
    expect(requests.some(url => new URL(url).origin === 'https://fonts.gstatic.com')).toBe(true)
  }
  await info.attach('noto-font-evidence', {
    body: JSON.stringify({
      localPreview,
      actualGoogleFontsBrowserLoad: !localPreview,
      evidence,
      weight,
      transfers,
    }, null, 2),
    contentType: 'application/json',
  })
  await assertAccessible(page, info)
  await captureState(page, info, 'noto-english-cjk')
  expect(errors).toEqual([])
})

test('Noto Emoji renders complete sequences without platform emoji fallback', async ({ page, context }, info) => {
  const errors = watchErrors(page)
  await page.goto('/fonts')
  await page.evaluate(async () => document.fonts.ready)
  const session = await context.newCDPSession(page)
  await session.send('DOM.enable')
  await session.send('CSS.enable')
  const evidence = []
  for (const id of [
    'face',
    'heart',
    'technologist',
    'family',
    'rainbow',
    'flag',
  ]) {
    const fonts = await renderedFonts(session, `#emoji-${id}`)
    evidence.push({ id, fonts })
    await info.attach(`emoji-${id}`, { body: JSON.stringify(fonts), contentType: 'application/json' })
    expect(fonts).toHaveLength(1)
    expect(fonts[0].isCustomFont).toBe(true)
    expect(fonts[0].familyName).toMatch(/^Noto Emoji/u)
    expect(fonts[0].glyphCount).toBe(1)
  }
  const mixed = await renderedFonts(session, '#font-mixed')
  expect(mixed.every(font => font.isCustomFont && /^Noto (?:Sans|Emoji)/u.test(font.familyName))).toBe(true)
  expect(mixed.some(font => font.familyName.startsWith('Noto Emoji'))).toBe(true)
  expect(mixed.some(font => font.familyName === 'Noto Sans')).toBe(true)
  await info.attach('mixed-text-fonts', { body: JSON.stringify({ mixed, evidence }), contentType: 'application/json' })
  await assertAccessible(page, info)
  await captureState(page, info, 'noto-emoji-sequences')
  expect(errors).toEqual([])
})
