import { resolve } from 'node:path'
import process from 'node:process'
import { ESLint } from 'eslint'
// eslint-disable-next-line antfu/no-import-dist -- These integration tests exercise the published build.
import config from '../dist/index.js'

// This harness repeatedly lints the same TS files with different configs, so it needs watch programs.
// Immutable single-run CI programs are validated separately in a fresh consumer process.
process.env.TSESTREE_SINGLE_RUN = 'false'

export const fixtureRoot = resolve('test/fixtures')

export async function makeESLint(options = {}, extra = {}) {
  return new ESLint({
    overrideConfigFile: true,
    overrideConfig: await config({ typescript: false, gitignore: false, ...options }),
    ...extra,
  })
}

export async function lintText(eslint, text, filePath = 'boundary.js') {
  const [result] = await eslint.lintText(text, { filePath })
  return result
}

export function messagesFor(result, rule) {
  return result.messages.filter(message => message.ruleId === rule)
}
