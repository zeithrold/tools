import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from './src/playwright.mjs'

export default defineConfig({
  ...verificationArtifacts(),
  testDir: './test/browser',
  retries: 0,
  workers: 1,
  use: { ...verificationArtifacts().use, browserName: 'chromium' },
})
