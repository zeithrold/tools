import type { Page, TestInfo } from '@playwright/test'
import { checkCss } from '@ztd-me/frontend-checks/css'
import { assertAccessible, captureState, verificationArtifacts } from '@ztd-me/frontend-checks/playwright'

const report = await checkCss({ files: ['src/**/*.css'], tokenFiles: ['tokens.css'], externalCustomProperties: ['--runtime'] })
const status: 'passed' | 'failed' = report.status
void status
declare const page: Page
declare const info: TestInfo
await assertAccessible(page, info, { label: 'dialog' })
await captureState(page, info, 'dialog')
verificationArtifacts('artifacts/browser')
// @ts-expect-error CSS file selection is mandatory.
await checkCss({})
