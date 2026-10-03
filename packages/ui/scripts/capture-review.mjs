import { spawn } from 'node:child_process'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { chromium } from '@playwright/test'

const [consumer, label] = process.argv.slice(2)
if (!consumer || !label) {
  throw new Error('Provide a built consumer directory and evidence label')
}
if (process.env.ZTD_LOCAL_FONT_PREVIEW && !label.includes('local-font-preview')) {
  throw new Error('Local-font preview captures must be explicitly labeled local-font-preview')
}
const directory = path.resolve('.artifacts', `visual-${label}`)
await mkdir(directory, { recursive: true })
const server = spawn('node', ['dist/server/server.js'], { cwd: consumer, stdio: 'inherit' })
const browser = await chromium.launch()
const evidence = []

async function openReview(viewport, preferences, colorScheme, record = false) {
  const context = await browser.newContext({
    viewport,
    colorScheme,
    ...(record ? { recordVideo: { dir: directory, size: viewport } } : {}),
  })
  await context.addCookies([
    {
      name: 'harbor.ui.v1',
      value: encodeURIComponent(JSON.stringify({ version: 1, ...preferences })),
      url: 'http://127.0.0.1:4317',
    },
  ])
  const page = await context.newPage()
  await page.goto('http://127.0.0.1:4317/review')
  await page.evaluate(async () => document.fonts.ready)
  return { page, context }
}

async function capture(name, viewport, preferences, colorScheme = 'light') {
  const { page, context } = await openReview(viewport, preferences, colorScheme)
  await page.screenshot({ path: path.join(directory, `${name}.png`), fullPage: true })
  const trigger = page.locator('.ztd-appbar button').first()
  const controls = await page.locator('.ztd-appbar .ztd-control').evaluateAll(nodes => nodes.map((node) => {
    const style = getComputedStyle(node)
    const bounds = node.getBoundingClientRect()
    return { width: bounds.width, height: bounds.height, fontSize: style.fontSize, border: style.borderColor }
  }))
  await trigger.press('Enter')
  const menu = page.getByRole('menu')
  const animation = await menu.evaluate(node => ({
    name: getComputedStyle(node).animationName,
    animations: node.getAnimations().length,
  }))
  await page.waitForTimeout(200)
  await page.screenshot({ path: path.join(directory, `${name}-appearance.png`), fullPage: true })
  await page.keyboard.press('Escape')
  await menu.waitFor({ state: 'detached' })
  await page.locator('.ztd-appbar [role="combobox"]').press('Enter')
  await page.waitForTimeout(200)
  await page.screenshot({ path: path.join(directory, `${name}-locale.png`), fullPage: true })
  evidence.push({ name, viewport, preferences, colorScheme, controls, animation })
  await context.close()
}

async function recordMotion() {
  const { page, context } = await openReview({ width: 1500, height: 860 }, {
    mode: 'light',
    palette: 'ocean',
    locale: 'zh-CN',
  }, 'light', true)
  await page.waitForTimeout(300)
  await page.locator('.ztd-appbar button').first().press('Enter')
  await page.waitForTimeout(600)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  await page.locator('.ztd-appbar [role="combobox"]').press('Enter')
  await page.waitForTimeout(600)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  const video = page.video()
  await context.close()
  await video.saveAs(path.join(directory, 'motion.webm'))
}

try {
  for (let attempt = 0; attempt < 30; attempt++) {
    try {
      await fetch('http://127.0.0.1:4317/review')
      break
    }
    catch {
      await new Promise((resolve) => {
        setTimeout(resolve, 100)
      })
    }
  }
  const ocean = { mode: 'light', palette: 'ocean', locale: 'zh-CN' }
  await capture('screenshot-ocean-light', { width: 1500, height: 860 }, ocean)
  await capture('mobile-ocean-light', { width: 390, height: 844 }, ocean)
  await capture('desktop-neutral-system-light', { width: 1920, height: 1080 }, {
    mode: 'system',
    palette: 'neutral',
    locale: 'en',
  })
  await capture('screenshot-neutral-dark', { width: 1500, height: 860 }, {
    mode: 'dark',
    palette: 'neutral',
    locale: 'en',
  })
  await capture('screenshot-ocean-system-dark', { width: 1500, height: 860 }, {
    mode: 'system',
    palette: 'ocean',
    locale: 'zh-CN',
  }, 'dark')
  await recordMotion()
  await writeFile(path.join(directory, 'measurements.json'), JSON.stringify(evidence, null, 2))
}
finally {
  await browser.close()
  server.kill()
}
