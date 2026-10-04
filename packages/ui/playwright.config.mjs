import process from 'node:process'
import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'
import { browserArtifactRoot } from './scripts/browser-artifacts.mjs'

const consumer = process.env.ZTD_UI_CONSUMER
const deliveryMode = process.env.ZTD_UI_VERIFICATION_MODE ?? 'manual'
const localFontPreview = Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW)
const artifacts = browserArtifactRoot(deliveryMode, localFontPreview)
if (!consumer) {
  throw new Error('Run pnpm test:pack to verify an independent packed consumer')
}
export default defineConfig({
  ...verificationArtifacts(artifacts),
  metadata: { consumer, deliveryMode, fontVerification: localFontPreview ? 'local-preview-only' : 'google-fonts-api' },
  testDir: './test/browser',
  workers: 1,
  retries: 0,
  use: {
    ...verificationArtifacts(artifacts).use,
    browserName: 'chromium',
    baseURL: 'http://127.0.0.1:4317',
  },
  webServer: {
    command: 'node dist/server/server.js',
    cwd: consumer,
    url: 'http://127.0.0.1:4317',
    reuseExistingServer: false,
  },
})
