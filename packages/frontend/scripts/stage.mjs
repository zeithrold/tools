import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { stagePackage } from '../../../scripts/npm-staging.mjs'

const tarball = process.argv[2]
if (!tarball) {
  throw new Error('Usage: node scripts/stage.mjs <verified-tarball>')
}
const packageInfo = JSON.parse(readFileSync('package.json', 'utf8'))
if (packageInfo.name !== '@ztd-me/frontend' || packageInfo.private) {
  throw new Error('Only the public frontend shell package may use this workflow')
}
const packedInfo = JSON.parse(execFileSync('tar', [
  '-xOf',
  tarball,
  'package/package.json',
], { encoding: 'utf8' }))
assert.equal(packedInfo.name, packageInfo.name)
assert.equal(packedInfo.version, packageInfo.version)
assert.ok(!packedInfo.private)
const sourceCommit = execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim()
assert.equal(sourceCommit, process.env.GITHUB_SHA, 'Stage only the immutable automatic workflow source')
const sha256 = createHash('sha256').update(readFileSync(tarball)).digest('hex')
const result = await stagePackage(tarball, packageInfo)
const receipt = `${JSON.stringify({ ...result, sourceCommit, sha256 }, null, 2)}\n`
writeFileSync('stage-result.json', receipt)
process.stdout.write(receipt)
