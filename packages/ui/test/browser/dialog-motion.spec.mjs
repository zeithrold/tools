import { expect } from '@playwright/test'
import { assertAccessible, test, watchErrors } from './helpers.mjs'

async function inspectActualMotion(page) {
  await page.evaluate(() => {
    // Pause genuine CSS animations at insertion/state changes, before their short timeline finishes.
    const observer = new MutationObserver(() => {
      for (const node of document.querySelectorAll('.ztd-dialog')) {
        for (const animation of node.getAnimations()) {
          if (['ztd-dialog-enter', 'ztd-dialog-exit'].includes(animation.animationName)) {
            animation.pause()
          }
        }
      }
    })
    observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: [
      'data-state',
    ] })
  })
}

async function sampleMotion(panel, name) {
  return panel.evaluate((node, expectedName) => {
    const animation = node.getAnimations().find(item => item.animationName === expectedName)
    if (!animation) {
      throw new Error(`Missing native CSS animation: ${expectedName}`)
    }
    animation.pause()
    const duration = animation.effect.getTiming().duration
    function bounds(phase, time) {
      const rect = node.getBoundingClientRect()
      const style = getComputedStyle(node)
      return {
        phase,
        time,
        centerX: rect.x + rect.width / 2,
        centerY: rect.y + rect.height / 2,
        left: rect.left,
        top: rect.top,
        right: rect.right,
        bottom: rect.bottom,
        opacity: Number(style.opacity),
        transform: style.transform,
        translate: style.translate,
      }
    }
    const samples = []
    for (const [phase, time] of [
      ['start', 0],
      [
        'half',
        duration / 2,
      ],
      [
        'near-end',
        duration - 1,
      ],
    ]) {
      animation.currentTime = time
      samples.push(bounds(phase, time))
    }
    animation.finish()
    samples.push(bounds('settled', duration))
    return { name: animation.animationName, duration, samples }
  }, name)
}

function centered(samples, viewport) {
  for (const sample of samples) {
    expect(Math.abs(sample.centerX - viewport.width / 2)).toBeLessThanOrEqual(0.5)
    expect(Math.abs(sample.centerY - viewport.height / 2)).toBeLessThanOrEqual(0.5)
    expect(sample.left).toBeGreaterThanOrEqual(15.5)
    expect(sample.top).toBeGreaterThanOrEqual(15.5)
    expect(sample.right).toBeLessThanOrEqual(viewport.width - 15.5)
    expect(sample.bottom).toBeLessThanOrEqual(viewport.height - 15.5)
  }
}

async function staticMotion(panel, viewport) {
  const sample = await panel.evaluate((node) => {
    const rect = node.getBoundingClientRect()
    return {
      phase: 'reduced',
      centerX: rect.x + rect.width / 2,
      centerY: rect.y + rect.height / 2,
      left: rect.left,
      top: rect.top,
      right: rect.right,
      bottom: rect.bottom,
      name: getComputedStyle(node).animationName,
      animations: node.getAnimations().length,
    }
  })
  centered([sample], viewport)
  expect(sample.name).toBe('none')
  expect(sample.animations).toBe(0)
  return sample
}

async function verifyModal(page, info, { kind, viewport, reduced }) {
  const confirmation = kind === 'confirmation'
  const trigger = page.getByRole('button', { name: `Open ${kind}`, exact: true })
  const panel = page.getByRole(confirmation ? 'alertdialog' : 'dialog', { name: `Fixture ${kind}`, exact: true })
  await trigger.press('Enter')
  const records = []
  if (reduced) {
    records.push(await staticMotion(panel, viewport))
  }
  else {
    const enter = await sampleMotion(panel, 'ztd-dialog-enter')
    expect(enter.duration).toBe(150)
    centered(enter.samples, viewport)
    expect(enter.samples[0].opacity).toBe(0)
    expect(enter.samples[2].opacity).toBeGreaterThan(0.99)
    records.push(enter)
  }
  await expect(panel).toHaveAttribute('data-state', 'open')
  const firstControl = confirmation
    ? panel.getByRole('button', { name: 'Cancel action' })
    : panel.getByRole('textbox')
  await expect(firstControl).toBeFocused()
  await assertAccessible(page, info)
  await page.screenshot({ path: info.outputPath(`dialog-motion-${kind}-${viewport.width}.png`) })
  await page.keyboard.press('Escape')
  if (!reduced) {
    await expect(panel).toHaveAttribute('data-state', 'closed')
    const exit = await sampleMotion(panel, 'ztd-dialog-exit')
    expect(exit.duration).toBe(100)
    centered(exit.samples, viewport)
    expect(exit.samples[0].opacity).toBe(1)
    expect(exit.samples[1].opacity).toBeLessThan(exit.samples[0].opacity)
    expect(exit.samples[2].opacity).toBeLessThan(exit.samples[1].opacity)
    records.push(exit)
  }
  await panel.waitFor({ state: 'detached' })
  await expect(trigger).toBeFocused()
  await expect(page.locator('[inert]')).toHaveCount(0)
  await info.attach(`${kind}-motion-geometry`, {
    body: JSON.stringify(records, null, 2),
    contentType: 'application/json',
  })
}

for (const width of [320, 1440]) {
  for (const motion of ['no-preference', 'reduce']) {
    test.describe(`modal centering at ${width}px with ${motion} motion`, () => {
      test.use({ viewport: { width, height: 900 }, contextOptions: { reducedMotion: motion }, hasTouch: width === 320 })
      test('Dialog and AlertDialog preserve center during native motion and restore focus', async ({ page }, info) => {
        const errors = watchErrors(page)
        await page.goto('/primitives')
        const prefersReduced = await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)
        expect(prefersReduced).toBe(motion === 'reduce')
        if (motion !== 'reduce') {
          await inspectActualMotion(page)
        }
        for (const kind of ['dialog', 'confirmation']) {
          await verifyModal(page, info, { kind, viewport: page.viewportSize(), reduced: motion === 'reduce' })
        }
        expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
        expect(errors).toEqual([])
      })
    })
  }
}
