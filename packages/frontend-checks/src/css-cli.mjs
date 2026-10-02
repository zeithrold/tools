#!/usr/bin/env node
import { writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'
import { checkCss } from './css.mjs'

async function main() {
  if (process.argv.length !== 3) {
    throw new Error('usage: ztd-css ./css-check.config.mjs; see README for the config schema')
  }
  const loaded = await import(pathToFileURL(path.resolve(process.argv[2])).href)
  const report = await checkCss(loaded.default)
  const data = `${JSON.stringify(report, null, 2)}\n`
  if (process.env.ZT_ARTIFACTS_DIR) {
    await writeFile(path.join(process.env.ZT_ARTIFACTS_DIR, 'css.json'), data)
  }
  process.stdout.write(data)
  if (report.status !== 'passed') {
    process.exitCode = 1
  }
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`)
  process.exitCode = 1
})
