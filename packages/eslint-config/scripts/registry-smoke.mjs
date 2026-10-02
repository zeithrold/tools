import process from 'node:process'
import { consumerSmoke } from './smoke.mjs'

const version = process.argv[2]
if (version === undefined || !/^\d+\.\d+\.\d+(?:-[\w.-]+)?$/u.test(version)) {
  throw new Error('Usage: node scripts/registry-smoke.mjs <exact-version>')
}
await consumerSmoke(version, `Registry @ztd-me/eslint@${version}`)
