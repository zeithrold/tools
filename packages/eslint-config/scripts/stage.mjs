import { readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { stagePackage } from './staging.mjs'

const tarball = process.argv[2]
if (tarball === undefined) {
  throw new Error('Usage: node scripts/stage.mjs <verified-tarball>')
}
const packageInfo = JSON.parse(readFileSync('package.json', 'utf8'))
const result = await stagePackage(tarball, packageInfo)
const receipt = `${JSON.stringify(result, null, 2)}\n`
writeFileSync('stage-result.json', receipt)
console.log(receipt)
