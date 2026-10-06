import { cp } from 'node:fs/promises'
import path from 'node:path'
import { retainBundledLicenses } from '../../../scripts/license-coverage.mjs'

export async function copyConsumerFixture(consumer) {
  await cp('test/consumer', consumer, { recursive: true })
  await cp('third-party', path.join(consumer, 'third-party'), { recursive: true })
}

export async function retainConsumerLicenses(consumer) {
  // This upstream npm artifact omits LICENSE; retain its author's original notice separately.
  await retainBundledLicenses(consumer, {
    'react-remove-scroll-bar@2.3.8': 'third-party/REACT-REMOVE-SCROLL-BAR-MIT.txt',
  })
}
