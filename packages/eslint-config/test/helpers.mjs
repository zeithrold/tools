import { resolve } from 'node:path'
import { ESLint } from 'eslint'
// eslint-disable-next-line antfu/no-import-dist -- These integration tests exercise the published build.
import config from '../dist/index.js'

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
