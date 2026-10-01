import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { performance } from 'node:perf_hooks'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { ESLint } from 'eslint'

const [manifestPath, mode] = process.argv.slice(2)
const manifest = JSON.parse(await readFile(manifestPath, 'utf8'))
const root = fileURLToPath(new URL('../', import.meta.url))
const eslintFor = config => new ESLint({ cwd: root, overrideConfigFile: true, overrideConfig: config })
let config
if (mode.endsWith('-lean')) {
  const [{ default: ts }, { default: parser }] = await Promise.all([import('@typescript-eslint/eslint-plugin'), import('@typescript-eslint/parser')])
  const plugins = { ts }
  if (mode === 'mixed-lean')
    plugins.style = (await import('@stylistic/eslint-plugin')).default
  const selected = manifest[mode.replace('-lean', '')]
  config = [{ files: ['**/*.ts'], languageOptions: { parser, ecmaVersion: 2026, sourceType: 'module', globals: manifest.globals }, plugins, rules: Object.fromEntries(selected.map(id => [id, manifest.entries[id]])), linterOptions: { reportUnusedDisableDirectives: 'off' } }]
}
else {
  const { default: antfu } = await import('@antfu/eslint-config')
  const configs = await antfu(structuredClone(manifest.options))
  const effective = await eslintFor(configs).calculateConfigForFile(manifest.files[0])
  config = configs
  if (mode === 'native' || mode === 'mixed') {
    const rules = Object.fromEntries(manifest[mode].map(id => [id, effective.rules[id]]))
    config = [{ files: ['**/*.ts'], languageOptions: effective.languageOptions, plugins: effective.plugins, rules, linterOptions: { reportUnusedDisableDirectives: 'off' } }]
  }
  if (mode === 'residual')
    config = [...configs, { files: ['**/*.ts'], rules: Object.fromEntries(manifest.moved.map(id => [id, 'off'])) }]
}
const eslint = eslintFor(config)
const start = performance.now()
const output = await eslint.lintFiles(manifest.files)
const entries = output.flatMap(f => f.messages.map(m => [path.basename(f.filePath), m.ruleId, m.severity]))
if (entries.some(([, id]) => !id))
  throw new Error('Unexpected parser/unmatched-file diagnostic in benchmark')
const signature = createHash('sha256').update(JSON.stringify(entries.sort())).digest('hex')
const rules = {}
for (const [, id] of entries)
  rules[id] = (rules[id] ?? 0) + 1
console.log(JSON.stringify({ signature, diagnostics: entries.length, rules, files: output.length, lintMs: performance.now() - start }))
