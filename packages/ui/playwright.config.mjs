import process from 'node:process'
import { defineConfig } from '@playwright/test'
import { verificationArtifacts } from '@ztd-me/frontend-checks/playwright'

const consumer = process.env.ZTD_UI_CONSUMER
const artifacts = process.env.ZTD_LOCAL_FONT_PREVIEW
  ? '.artifacts/browser-local-font-preview'
  : '.artifacts/browser'
if (!consumer) {
  throw new Error('Run pnpm test:pack to verify an independent packed consumer')
}
export default defineConfig({
  ...verificationArtifacts(artifacts),
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
