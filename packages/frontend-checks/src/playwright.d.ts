import type { Page, TestInfo, PlaywrightTestConfig } from '@playwright/test'
import type { AxeBuilder } from '@axe-core/playwright'
export type AccessibilityOptions = {
  label?: string
  /** Restricts the scan; full-page scans remain necessary for page-level checks. */
  include?: string
  tags?: string[]
}
export function assertAccessible(page: Page, testInfo: TestInfo, options?: AccessibilityOptions): ReturnType<AxeBuilder['analyze']>
export function captureState(page: Page, testInfo: TestInfo, label: string): Promise<void>
export function verificationArtifacts(root?: string): Pick<PlaywrightTestConfig, 'outputDir' | 'reporter' | 'use'>
