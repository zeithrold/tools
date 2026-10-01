import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'
import antfu from '@antfu/eslint-config'
import migrate from '@oxlint/migrate'
import { materialize, residualConfig } from '../prototype/materialize.mjs'
import { contextProbe } from '../prototype/plugin.mjs'
import { baseOptions, eslintFor, generated, json, results, root, runOx } from './common.mjs'

const dir = path.join(generated, 'experiments')
await mkdir(dir, { recursive: true })
const pluginUrl = pathToFileURL(path.join(root, 'prototype/plugin.mjs')).href
const report = { versions: { node: process.version, antfu: '9.5.1', oxlint: '1.86.0', eslint: '10.11.0', tsgolint: '7.0.2003' }, specimens: [], migration: {}, matching: {}, hybrid: {} }
async function wrapper(namespace, options = baseOptions) {
  const file = path.join(dir, `plugin-${encodeURIComponent(namespace)}-${options.react ? 'react' : 'base'}.mjs`)
  await writeFile(file, `import { resolvedPlugin } from ${JSON.stringify(pluginUrl)};\nexport default await resolvedPlugin(${JSON.stringify(namespace)}, ${JSON.stringify(options)});\n`)
  return file
}
function slim(config, rules, extra = {}) {
  return [{ files: ['**/*.{js,jsx,ts,tsx,mjs,cjs,mts,cts}'], languageOptions: config.languageOptions, plugins: config.plugins, settings: config.settings ?? {}, linterOptions: { reportUnusedDisableDirectives: 'off' }, rules, ...extra }]
}
function oxMessages(run) {
  return run.output?.diagnostics ?? []
}
function counts(messages, ox = false) {
  return messages.map(m => ox ? m.severity === 'warning' ? 1 : 2 : m.severity).sort()
}
async function specimen({ name, id, source, file = 'case.ts', entry, target, namespace, specifier, options = baseOptions, fix = true, nativePlugin, flags = [], configExtra = {}, eslintExtra = {} }) {
  const resolved = await eslintFor(await antfu(structuredClone(options))).calculateConfigForFile(path.join(root, file))
  const ruleEntry = entry ?? resolved.rules[id]
  assert.ok(ruleEntry, `Missing ${id}`)
  const rules = { [id]: ruleEntry }
  const baseline = eslintFor(slim(resolved, rules, eslintExtra))
  const eslintResult = (await baseline.lintText(source, { filePath: file }))[0]
  let jsPlugins = []
  if (namespace) {
    const specifier = await wrapper(namespace, options)
    jsPlugins = [{ name: target.slice(0, target.lastIndexOf('/')), specifier }]
  }
  if (specifier)
    jsPlugins = [{ name: target.slice(0, target.lastIndexOf('/')), specifier }]
  const oxConfig = { plugins: nativePlugin ? [nativePlugin] : [], categories: { correctness: 'off' }, env: { builtin: true }, globals: resolved.languageOptions.globals, jsPlugins, rules: { [target]: ruleEntry }, ...configExtra }
  const configPath = path.join(dir, `${name}.json`)
  await json(configPath, oxConfig)
  const sourcePath = path.join(dir, file)
  await writeFile(sourcePath, source)
  const oxResult = runOx(configPath, [sourcePath], flags)
  let edits
  if (fix) {
    const fixed = (await eslintFor(slim(resolved, rules, eslintExtra), root, { fix: true }).lintText(source, { filePath: file }))[0]
    const oxFixed = runOx(configPath, [sourcePath], [...flags, '--fix'])
    const output = await readFile(sourcePath, 'utf8')
    edits = { eslint: fixed.output ?? source, oxlint: output, equal: (fixed.output ?? source) === output, run: oxFixed }
  }
  assert.ok(eslintResult.messages.every(m => m.ruleId === id), `${name}: baseline contains an unrelated diagnostic`)
  const runtimeError = !oxResult.output || oxMessages(oxResult).some(m => m.message.startsWith('Error running JS plugin'))
  const record = { name, id, target, file, source, entry: ruleEntry, eslintEntry: eslintExtra.rules?.[id] ?? ruleEntry, eslint: eslintResult.messages, oxlint: oxResult, runtimeError, matchingSeverityCounts: !runtimeError && JSON.stringify(counts(eslintResult.messages)) === JSON.stringify(counts(oxMessages(oxResult), true)), edits }
  report.specimens.push(record)
  console.log(name, 'eslint', eslintResult.messages.length, 'oxlint', oxMessages(oxResult).length, 'status', oxResult.status, 'fixEqual', edits?.equal)
  return record
}

await specimen({ name: 'console-options-shadowing', id: 'no-console', target: 'no-console', file: 'case.js', source: 'console.log(\'bad\'); console.warn(\'ok\'); console.error(\'ok\');\nfunction local(console) { console.log(\'local\') }\nconsole[\'log\'](\'bad\'); const x = \'log\'; console[x](\'dynamic\');\n' })
await specimen({ name: 'console-original-core-js', id: 'no-console', target: 'core-js/no-console', namespace: 'core-js', file: 'case.js', source: 'console.log(\'bad\'); console.warn(\'ok\'); console.error(\'ok\');\nfunction local(console) { console.log(\'local\') }\nconsole[\'log\'](\'bad\'); const x = \'log\'; console[x](\'dynamic\');\n' })
await specimen({ name: 'console-warn', id: 'no-console', target: 'no-console', entry: [1, { allow: ['warn', 'error'] }], source: 'console.log(\'bad\')\nconsole.warn(\'ok\')\n' })
await specimen({ name: 'eqeqeq-smart', id: 'eqeqeq', target: 'eqeqeq', source: 'export function check(a) { return a == 1 || a == null || typeof a == \'string\' || 1 == 2 }\n' })
await specimen({ name: 'prefer-const-options', id: 'prefer-const', target: 'prefer-const', source: 'let value = 1; export { value };\nlet {a, b} = source; b = 2; use(a, b);\n' })
await specimen({ name: 'ban-ts-comment', id: 'ts/ban-ts-comment', target: 'typescript/ban-ts-comment', nativePlugin: 'typescript', source: '// @ts-expect-error\nexport const value: string = 1\n// @ts-expect-error: a documented reason\nexport const other: string = 2\n' })
await specimen({ name: 'type-definitions', id: 'ts/consistent-type-definitions', target: 'typescript/consistent-type-definitions', nativePlugin: 'typescript', source: 'export type Thing = { value: string };\n' })
await specimen({ name: 'type-imports-native', id: 'ts/consistent-type-imports', target: 'typescript/consistent-type-imports', nativePlugin: 'typescript', source: 'import { Thing } from \'./types\';\nexport const value: Thing = {};\nexport type Other = import(\'./types\').Other;\n' })
await specimen({ name: 'type-imports-js', id: 'ts/consistent-type-imports', target: 'ts/consistent-type-imports', namespace: 'ts', source: 'import { Thing } from \'./types\';\nexport const value: Thing = {};\n' })
const adapter = path.join(root, 'prototype/syntax-type-imports.mjs')
await specimen({ name: 'type-imports-adapted', id: 'ts/consistent-type-imports', target: 'syntax-adapter/consistent-type-imports', specifier: adapter, source: 'import { Thing, makeThing } from \'./types\';\nexport const value: Thing = makeThing();\nexport type Other = import(\'./types\').Other;\n' })
await specimen({ name: 'type-imports-adapted-comments', id: 'ts/consistent-type-imports', target: 'syntax-adapter/consistent-type-imports', specifier: adapter, source: 'import { /* keep */ Thing } from \'./types\';\nexport const value: Thing = {};\n' })
await specimen({ name: 'import-lite-options-native-raw', id: 'import/consistent-type-specifier-style', target: 'import/consistent-type-specifier-style', nativePlugin: 'import', source: 'import { type Thing, value } from \'./types\';\nexport { value };\nexport type Other = Thing;\n', fix: false })
await specimen({ name: 'import-lite-options-native-translated', id: 'import/consistent-type-specifier-style', target: 'import/consistent-type-specifier-style', nativePlugin: 'import', entry: [2, 'prefer-top-level'], eslintExtra: { rules: { 'import/consistent-type-specifier-style': [2, 'top-level'] } }, source: 'import { type Thing, value } from \'./types\';\nexport { value };\nexport type Other = Thing;\n' })
await specimen({ name: 'import-lite-original-js', id: 'import/consistent-type-specifier-style', target: 'bridge-import/consistent-type-specifier-style', namespace: 'import', source: 'import { type Thing, value } from \'./types\';\nexport { value };\nexport type Other = Thing;\n' })
await specimen({ name: 'style-semi', id: 'style/semi', target: 'style/semi', namespace: 'style', source: 'export const value = 1;\n' })
await specimen({ name: 'style-quotes', id: 'style/quotes', target: 'style/quotes', namespace: 'style', source: 'export const value = "hello";\nexport const other = "can\'t";\n' })
await specimen({ name: 'style-indent', id: 'style/indent', target: 'style/indent', namespace: 'style', source: 'export function f() {\n    return {\n        value: 1,\n    };\n}\n' })
await specimen({ name: 'style-comma-dangle', id: 'style/comma-dangle', target: 'style/comma-dangle', namespace: 'style', source: 'export const value = {\n  a: 1\n};\n' })
await specimen({ name: 'antfu-curly', id: 'antfu/curly', target: 'antfu/curly', namespace: 'antfu', source: 'export function f(x) { if (x) { run(); } else run(); }\n' })
await specimen({ name: 'antfu-top-level-await', id: 'antfu/no-top-level-await', target: 'antfu/no-top-level-await', namespace: 'antfu', source: 'await work();\nexport async function f() { await work(); }\n' })
await specimen({ name: 'core-selector-js', id: 'no-restricted-syntax', target: 'core-js/no-restricted-syntax', namespace: 'core-js', source: 'const enum State { Ready }\nexport = State;\n' })
await specimen({ name: 'perfectionist-sort-imports', id: 'perfectionist/sort-imports', target: 'perfectionist/sort-imports', namespace: 'perfectionist', source: 'import z from \'./z\';\nimport a from \'a\';\nexport { z, a };\n' })
await specimen({ name: 'unused-imports-autofix', id: 'unused-imports/no-unused-imports', target: 'unused-imports/no-unused-imports', namespace: 'unused-imports', source: 'import { unused, used } from \'./dependency\';\nexport { used };\n' })
await writeFile(path.join(dir, 'module-with-side-effects.mjs'), 'globalThis.bridgeEffect = 1;\nexport default 1;\n')
const risk = await specimen({ name: 'unused-imports-side-effect-risk', id: 'unused-imports/no-unused-imports', target: 'unused-imports/no-unused-imports', namespace: 'unused-imports', source: 'import unused from \'./module-with-side-effects.mjs\';\nexport const value = globalThis.bridgeEffect ?? 0;\n' })
const values = {}
for (const [engine, source] of Object.entries({ original: risk.source, eslintFix: risk.edits.eslint, oxlintFix: risk.edits.oxlint })) {
  const file = path.join(dir, `risk-${engine}.mjs`)
  await writeFile(file, source)
  const run = spawnSync(process.execPath, ['--input-type=module', '--eval', `import { value } from ${JSON.stringify(pathToFileURL(file).href)}; console.log(value);`], { encoding: 'utf8' })
  assert.equal(run.status, 0, run.stderr)
  values[engine] = Number(run.stdout.trim())
}
report.fixSafety = values
await specimen({ name: 'regexp-token-scope', id: 'regexp/no-dupe-characters-character-class', target: 'regexp/no-dupe-characters-character-class', namespace: 'regexp', source: 'export const regex = /[aab]/;\n' })
await specimen({ name: 'e18e-date-now', id: 'e18e/prefer-date-now', target: 'e18e/prefer-date-now', namespace: 'e18e', source: 'export const time = new Date().getTime();\n' })
await specimen({ name: 'test-synthetic-plugin', id: 'test/no-only-tests', target: 'test/no-only-tests', namespace: 'test', file: 'case.test.ts', source: 'describe.only(\'x\', () => { it(\'works\', () => {}) });\n' })
await specimen({ name: 'react-actual-plugin', id: 'react/no-array-index-key', target: 'bridge-react/no-array-index-key', namespace: 'react', file: 'case.tsx', options: { ...baseOptions, react: true }, source: 'export function List({items}) { return items.map((item, index) => <span key={index}>{item}</span>); }\n' })

// Disable namespace mapping and inline option changes are independent from rule implementation parity.
await specimen({ name: 'disable-core', id: 'no-console', target: 'no-console', source: '// eslint-disable-next-line no-console -- reason\nconsole.log(1);\nconsole.log(2);\n', fix: false })
await specimen({ name: 'disable-style-original', id: 'style/semi', target: 'style/semi', namespace: 'style', source: '// eslint-disable-next-line style/semi -- reason\nexport const a = 1;\nexport const b = 2;\n', fix: false })
await specimen({ name: 'disable-style-renamed', id: 'style/semi', target: 'style-js/semi', namespace: 'style', source: '// eslint-disable-next-line style/semi -- reason\nexport const a = 1;\nexport const b = 2;\n', fix: false })
await specimen({ name: 'disable-ts-original-native', id: 'ts/ban-ts-comment', target: 'typescript/ban-ts-comment', nativePlugin: 'typescript', source: '// eslint-disable ts/ban-ts-comment\n// @ts-expect-error\nexport const a: string = 1;\n', fix: false })
await specimen({ name: 'disable-ts-original-js', id: 'ts/ban-ts-comment', target: 'ts/ban-ts-comment', namespace: 'ts', source: '// eslint-disable ts/ban-ts-comment\n// @ts-expect-error\nexport const a: string = 1;\n', fix: false })
await specimen({ name: 'disable-ts-block-native', id: 'ts/ban-ts-comment', target: 'typescript/ban-ts-comment', nativePlugin: 'typescript', source: '/* eslint-disable ts/ban-ts-comment */\n// @ts-expect-error\nexport const a: string = 1;\n', fix: false })
await specimen({ name: 'inline-rule-options', id: 'no-console', target: 'no-console', source: '/* eslint no-console: ["error", { "allow": ["log"] }] */\nconsole.log(1);\nconsole.warn(2);\n', fix: false })
await specimen({ name: 'invalid-native-options', id: 'no-console', target: 'no-console', source: 'console.log(1);\n', entry: [2, { allow: ['warn'], invented: true }], fix: false }).catch(error => report.specimens.push({ name: 'invalid-native-options', baselineError: error.message }))
const invalidPath = path.join(dir, 'invalid-options.json')
await json(invalidPath, { plugins: [], categories: { correctness: 'off' }, rules: { 'no-console': ['error', { allow: ['warn'], invented: true }] } })
report.invalidOptionsOxlint = runOx(invalidPath, [path.join(dir, 'case.ts')])

// A direct probe exercises modern SourceCode token/scope/CFG/settings APIs.
const probeFile = path.join(dir, 'probe.js')
const probeSource = '// comment\nexport const value = 1;\nfunction f() { return value }\n'
await writeFile(probeFile, probeSource)
const probeEslint = (await eslintFor([{ plugins: { probe: contextProbe }, settings: { bridgeProbe: 'per-file' }, rules: { 'probe/context': 'error' } }]).lintText(probeSource, { filePath: 'probe.js' }))[0]
const probePlugin = path.join(dir, 'probe-plugin.mjs')
await writeFile(probePlugin, `export { contextProbe as default } from ${JSON.stringify(pluginUrl)};\n`)
const probeConfig = path.join(dir, 'probe.json')
await json(probeConfig, { categories: { correctness: 'off' }, plugins: [], jsPlugins: [{ name: 'probe', specifier: probePlugin }], settings: { bridgeProbe: 'per-file' }, rules: { 'probe/context': 'error' } })
report.context = { eslint: probeEslint.messages, oxlint: runOx(probeConfig, [probeFile]) }

// Normal fixes, suggestions and dangerous suggestions are distinct execution modes.
report.fixModes = []
const consoleResolved = await eslintFor(await antfu(structuredClone(baseOptions))).calculateConfigForFile(path.join(root, 'mode.js'))
for (const [name, source] of Object.entries({ statement: 'console.log(\'hi\');\n', sequence: 'export const value = (console.log(\'hi\'), 1);\n' })) {
  const baseline = (await eslintFor(slim(consoleResolved, { 'no-console': consoleResolved.rules['no-console'] }), root, { fix: true }).lintText(source, { filePath: 'mode.js' }))[0]
  for (const engine of ['native', 'core-js']) {
    const configPath = path.join(dir, `fix-mode-${engine}.json`)
    await json(configPath, { categories: { correctness: 'off' }, plugins: [], jsPlugins: engine === 'native' ? [] : [{ name: 'core-js', specifier: await wrapper('core-js') }], rules: { [engine === 'native' ? 'no-console' : 'core-js/no-console']: consoleResolved.rules['no-console'] } })
    for (const flags of [['--fix'], ['--fix-suggestions'], ['--fix-suggestions', '--fix-dangerously']]) {
      const file = path.join(dir, `fix-mode-${name}.js`)
      await writeFile(file, source)
      const run = runOx(configPath, [file], flags)
      report.fixModes.push({ name, engine, source, flags, eslintFix: baseline.output ?? source, eslintSuggestions: baseline.messages.flatMap(m => m.suggestions ?? []), output: await readFile(file, 'utf8'), run })
    }
  }
}

// TS-native type awareness is distinct from the JS plugin API's empty parserServices.
const typedDir = path.join(dir, 'typed')
await mkdir(typedDir, { recursive: true })
await json(path.join(typedDir, 'tsconfig.json'), { compilerOptions: { strict: true, target: 'ES2022', module: 'ESNext', moduleResolution: 'bundler' }, include: ['*.ts'] })
const typedSource = 'export async function f(): Promise<void> {}\nf();\nvoid f();\ndeclare const unsafe: any;\nexport const value: string = unsafe;\n'
const typedFile = path.join(typedDir, 'case.ts')
await writeFile(typedFile, typedSource)
const typedOptions = { ...baseOptions, typescript: { tsconfigPath: path.join(typedDir, 'tsconfig.json') } }
const typedConfig = await antfu(typedOptions)
const typedResolved = await eslintFor(typedConfig).calculateConfigForFile(typedFile)
const typedRules = { 'ts/no-floating-promises': [2, { ignoreVoid: true }], 'ts/no-unsafe-assignment': [1] }
const typedBaseline = (await eslintFor(slim(typedResolved, typedRules)).lintFiles([typedFile]))[0]
const typedOxPath = path.join(typedDir, 'native.json')
await json(typedOxPath, { categories: { correctness: 'off' }, plugins: ['typescript'], rules: { 'typescript/no-floating-promises': typedRules['ts/no-floating-promises'], 'typescript/no-unsafe-assignment': typedRules['ts/no-unsafe-assignment'] } })
const typedJsPath = path.join(typedDir, 'js.json')
await json(typedJsPath, { categories: { correctness: 'off' }, plugins: [], jsPlugins: [{ name: 'ts', specifier: await wrapper('ts') }], rules: typedRules })
report.typed = { source: typedSource, eslint: typedBaseline.messages, nativeWithoutFlag: runOx(typedOxPath, [typedFile]), native: runOx(typedOxPath, [typedFile], ['--type-aware', '--tsconfig', path.join(typedDir, 'tsconfig.json')]), jsPlugin: runOx(typedJsPath, [typedFile]) }

// File-format and processor boundaries. These files deliberately contain real violations.
const formats = {
  'component.vue': '<script setup>debugger; const rows = [1];</script>\n<template><div v-for="row in rows">{{ row }}</div></template>\n',
  'component.svelte': '<script>debugger; let rows = [1];</script>\n{#each rows as row}<div>{row}</div>{/each}\n',
  'component.astro': '---\ndebugger;\n---\n<img src="/x.png" />\n',
  'data.jsonc': '{ "x": 1, "x": 2 }\n',
  'data.yaml': 'x: 1\nx: 2\n',
  'data.toml': 'x=1\nx=2\n',
  'guide.md': '# Repeated\n# Repeated\n\n```js\ndebugger;\n```\n',
}
const formatPaths = []
for (const [file, source] of Object.entries(formats)) {
  const filePath = path.join(dir, file)
  await writeFile(filePath, source)
  formatPaths.push(filePath)
}
const formatConfig = await antfu({ ...baseOptions, vue: true, svelte: true, astro: true })
const formatEslint = await eslintFor(formatConfig).lintFiles(formatPaths)
const formatOx = path.join(dir, 'formats.json')
await json(formatOx, { categories: { correctness: 'off' }, plugins: ['vue'], rules: { 'no-debugger': 'error' } })
report.formats = { sources: formats, eslint: formatEslint.map(f => ({ file: path.basename(f.filePath), messages: f.messages })), oxlint: runOx(formatOx, formatPaths) }
const vueJsPath = path.join(dir, 'vue-js.json')
await json(vueJsPath, { categories: { correctness: 'off' }, plugins: [], jsPlugins: [{ name: 'vue-js', specifier: await wrapper('vue', { ...baseOptions, vue: true }) }], rules: { 'vue-js/require-v-for-key': 'error' } })
report.formats.vueTemplateJs = runOx(vueJsPath, [formatPaths[0]])
const formatted = [path.join(dir, 'style.css'), path.join(dir, 'page.html')]
await writeFile(formatted[0], 'a {color:red;}\n')
await writeFile(formatted[1], '<html><body><div>x</div></body></html>\n')
const formattingConfig = await antfu({ ...baseOptions, formatters: { css: true, html: true } })
report.formats.formatters = { eslint: (await eslintFor(formattingConfig).lintFiles(formatted)).map(f => ({ file: path.basename(f.filePath), messages: f.messages })), oxlint: runOx(formatOx, formatted) }

// Actual migrator output, before and after correcting only the erroneous package specifier.
const migrated = await migrate(antfu(structuredClone(baseOptions)), undefined, { jsPlugins: true, withNursery: true })
const migratedPath = path.join(dir, 'migrated.json')
await json(migratedPath, migrated)
const migrationFile = path.join(dir, 'migration.ts')
await writeFile(migrationFile, 'export const value = "hello";\n')
report.migration.raw = runOx(migratedPath, [migrationFile])
const corrected = structuredClone(migrated)
function fixSpecifiers(config) {
  config.jsPlugins = config.jsPlugins?.map(p => p === 'eslint-plugin-eslint-comments' ? '@eslint-community/eslint-plugin-eslint-comments' : p === '@eslint/eslint-plugin-markdown' ? '@eslint/markdown' : p)
  config.overrides?.forEach(fixSpecifiers)
}
fixSpecifiers(corrected)
await json(migratedPath, corrected)
report.migration.correctedPackageOnly = runOx(migratedPath, [migrationFile])
function patchForExecution(config) {
  config.jsPlugins = config.jsPlugins?.map(p => p === '@eslint-community/eslint-plugin-eslint-comments' ? { name: 'eslint-comments', specifier: p } : p === '@eslint/markdown' ? { name: 'markdown', specifier: p } : p)
  if (config.rules?.['import/consistent-type-specifier-style']?.[1] === 'top-level')
    config.rules['import/consistent-type-specifier-style'][1] = 'prefer-top-level'
  config.overrides?.forEach(patchForExecution)
}
patchForExecution(corrected)
await json(migratedPath, corrected)
report.migration.patchedAliasesAndImportOption = runOx(migratedPath, [migrationFile])
report.migration.originalEslint = (await eslintFor(await antfu(structuredClone(baseOptions))).lintFiles([migrationFile]))[0].messages
const sortingFile = path.join(dir, 'migration-sorting.ts')
await writeFile(sortingFile, 'import z from \'./z\'\nimport a from \'a\'\n\nexport { z, a }\n')
report.migration.sortingAfterPatch = {
  eslint: (await eslintFor(await antfu(structuredClone(baseOptions))).lintFiles([sortingFile]))[0].messages,
  oxlint: runOx(migratedPath, [sortingFile]),
}

// Local ignores, AND patterns, basePath, severity-only merge and per-file settings.
const matchingDir = path.join(dir, 'matching')
await mkdir(path.join(matchingDir, 'src'), { recursive: true })
await mkdir(path.join(matchingDir, 'nested'), { recursive: true })
const matchFiles = ['src/keep.js', 'src/skip.js', 'src/keep.test.js', 'nested/keep.js']
for (const file of matchFiles)
  await writeFile(path.join(matchingDir, file), 'console.warn(1); console.log(2);\n')
const matchConfig = [
  { files: ['**/*.js'], rules: { 'no-console': [2, { allow: ['warn'] }] } },
  { files: [['src/**', '**/*.test.js']], rules: { 'no-console': 'off' } },
  { files: ['src/**'], ignores: ['src/skip.js'], rules: { 'no-console': 'warn' } },
  { basePath: 'nested', files: ['**/*.js'], rules: { 'no-console': 'off' } },
]
const matchingEslint = await eslintFor(matchConfig, matchingDir).lintFiles(matchFiles)
const matchWarnings = []
const matchingMigrated = await migrate(matchConfig, undefined, { jsPlugins: true, withNursery: true, reporter: { addWarning: m => matchWarnings.push(m) } })
const matchingPath = path.join(matchingDir, 'migrated.json')
await json(matchingPath, matchingMigrated)
report.matching = { config: matchConfig, eslint: matchingEslint.map(f => ({ file: path.relative(matchingDir, f.filePath), messages: f.messages })), migrated: matchingMigrated, warnings: matchWarnings, oxlint: runOx(matchingPath, matchFiles, [], matchingDir) }

// Minimal bridge + residual ESLint executes exactly the original rule workload.
const hybridDir = path.join(dir, 'hybrid')
await mkdir(hybridDir, { recursive: true })
const configModule = path.join(hybridDir, 'eslint.config.mjs')
await writeFile(configModule, `import antfu from '@antfu/eslint-config';\nexport default antfu(${JSON.stringify(baseOptions)}, { files: ['**/*.js'], rules: { 'no-console': [1, { allow: ['warn', 'error'] }] } });\n`)
const hybridFiles = ['normal.js', 'overridden.test.js', 'data.jsonc', 'directive.js']
for (const file of hybridFiles)
  await writeFile(path.join(hybridDir, file), file.endsWith('jsonc') ? '{"x":1,"x":2}\n' : `${file === 'directive.js' ? '// eslint-disable-next-line no-console\n' : ''}console.log("x");\ndebugger;\n`)
const routes = {
  'no-debugger': { kind: 'native', id: 'no-debugger', accept: entry => entry.length === 1 },
  'style/semi': { kind: 'js', accept: () => true },
  'style/quotes': { kind: 'js', accept: () => true },
}
const plan = await materialize({ configModule, files: hybridFiles, routes, outputDirectory: path.join(hybridDir, 'bridge'), cwd: hybridDir })
const originalConfig = await (await import(pathToFileURL(configModule).href)).default
const original = await eslintFor(originalConfig, hybridDir).lintFiles(hybridFiles)
const residual = await eslintFor(residualConfig(originalConfig, plan), hybridDir).lintFiles(hybridFiles)
const oxRuns = plan.groups.map(g => ({ file: g.file, ...runOx(g.configPath, [path.join(hybridDir, g.file)], [], hybridDir) }))
report.hybrid = { plan, original: original.map(f => ({ file: path.basename(f.filePath), messages: f.messages })), residual: residual.map(f => ({ file: path.basename(f.filePath), messages: f.messages })), oxlint: oxRuns, originalCount: original.reduce((n, f) => n + f.messages.length, 0), hybridCount: residual.reduce((n, f) => n + f.messages.length, 0) + oxRuns.reduce((n, r) => n + oxMessages(r).length, 0) }
const eslintSignatures = files => files.flatMap(f => f.messages.map(m => JSON.stringify([path.basename(f.filePath), m.ruleId, m.severity]))).sort()
const movedSignatures = oxRuns.flatMap((run) => {
  assert.ok(run.output, 'Hybrid config failed')
  return oxMessages(run).map((m) => {
    assert.ok(!m.message.startsWith('Error running JS plugin'), m.message)
    const [, namespace, rule] = /^([^()]+)\((.+)\)$/.exec(m.code)
    const id = namespace === 'eslint' ? rule : `${namespace}/${rule}`
    const originalId = Object.keys(plan.mappings).find(key => plan.mappings[key] === id)
    assert.ok(originalId, `Unmapped diagnostic: ${m.code}`)
    return JSON.stringify([run.file, originalId, m.severity === 'warning' ? 1 : 2])
  })
})
assert.deepEqual([...eslintSignatures(residual), ...movedSignatures].sort(), eslintSignatures(original))
report.hybrid.sameFileRuleSeverity = true
await json(path.join(results, 'experiments.json'), report)
assert.equal(report.hybrid.originalCount, report.hybrid.hybridCount)
assert.equal(plan.groups.length, 2)
const byName = name => report.specimens.find(s => s.name === name)
assert.equal(byName('console-options-shadowing').matchingSeverityCounts, false)
assert.equal(byName('console-original-core-js').matchingSeverityCounts, true)
assert.equal(byName('type-imports-js').runtimeError, true)
assert.equal(byName('type-imports-adapted').edits.equal, true)
assert.equal(byName('import-lite-options-native-translated').edits.equal, false)
assert.equal(byName('disable-style-renamed').matchingSeverityCounts, false)
assert.equal(byName('inline-rule-options').eslint[0].line, 3)
assert.equal(byName('inline-rule-options').oxlint.output.diagnostics[0].labels[0].span.line, 2)
assert.deepEqual(report.fixSafety, { original: 1, eslintFix: 0, oxlintFix: 0 })
assert.equal(report.typed.nativeWithoutFlag.output.number_of_rules, 0)
assert.equal(report.typed.native.output.diagnostics.length, report.typed.eslint.length)
assert.equal(report.formats.oxlint.output.number_of_files, 3)
assert.ok(report.fixModes.filter(m => m.flags.length === 1 && m.flags[0] === '--fix').every(m => m.output === m.source))
assert.ok(report.fixModes.find(m => m.name === 'sequence' && m.engine === 'native' && m.flags.length === 2).output.includes('undefined'))
assert.ok(report.migration.sortingAfterPatch.eslint.some(m => m.ruleId === 'perfectionist/sort-imports'))
assert.ok(!report.migration.sortingAfterPatch.oxlint.output.diagnostics.some(m => m.code === 'perfectionist(sort-imports)'))
console.log('Wrote results/experiments.json; hybrid counts:', report.hybrid.originalCount, report.hybrid.hybridCount)
