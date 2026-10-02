import assert from 'node:assert/strict'
import process from 'node:process'
import AxeBuilder from '@axe-core/playwright'

/** Scan after each relevant UI state is established, not only the initial route. */
export async function assertAccessible(page, testInfo, options = {}) {
  if (options.tags && !options.tags.length) {
    throw new TypeError('Accessibility tags must not be empty')
  }
  let builder = new AxeBuilder({ page }).withTags(options.tags ?? [
    'wcag2a',
    'wcag2aa',
    'wcag21aa',
    'wcag22aa',
  ])
  if (options.include) {
    builder = builder.include(options.include)
  }
  const results = await builder.analyze()
  await testInfo.attach(`a11y-${options.label ?? 'state'}`, {
    body: JSON.stringify(results, null, 2),
    contentType: 'application/json',
  })
  assert.equal(results.violations.length, 0, JSON.stringify(results.violations, null, 2))
  return results
}

/** Screenshots are captured evidence; this helper does not maintain baselines. */
export async function captureState(page, testInfo, label) {
  await testInfo.attach(label, { body: await page.screenshot({ fullPage: true }), contentType: 'image/png' })
}

/** Apply these paths to the project's existing config; retain its webServer/projects. */
export function verificationArtifacts(root = process.env.ZT_ARTIFACTS_DIR ?? '.zt/browser') {
  return {
    outputDir: `${root}/test-results`,
    reporter: [
      ['list'],
      [
        'html',
        { outputFolder: `${root}/playwright-report`, open: 'never' },
      ],
      [
        'json',
        { outputFile: `${root}/playwright.json` },
      ],
    ],
    use: { trace: 'retain-on-failure', screenshot: 'only-on-failure', video: 'retain-on-failure' },
  }
}
