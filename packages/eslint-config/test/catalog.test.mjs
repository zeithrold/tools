/* eslint-disable antfu/no-top-level-await -- Node's test runner awaits ESM setup before registering the tests. */
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import test from 'node:test'
// eslint-disable-next-line antfu/no-import-dist -- These integration tests exercise the published build.
import config from '../dist/index.js'
import { fixtureRoot, makeESLint } from './helpers.mjs'

const catalog = JSON.parse(await readFile(new URL('../docs/rule-catalog.json', import.meta.url), 'utf8'))
const tsOptions = { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') }
const eslint = await makeESLint({ typescript: tsOptions, react: true, vue: true, test: true })
const configs = {}
const paths = {
  'shared': 'sample.js',
  'ts-syntax': 'safe.ts',
  'typed': 'safe.ts',
  'react': 'Button.tsx',
  'vue': 'Safe.vue',
  'vue-a11y': 'Safe.vue',
  'tests': 'example.test.ts',
}
for (const [group, file] of Object.entries(paths)) {
  configs[group] = await eslint.calculateConfigForFile(resolve(fixtureRoot, file))
}

function normalized(value) {
  const array = Array.isArray(value)
    ? [
        ...value,
      ]
    : [value]
  if (array[0] === 'error') {
    array[0] = 2
  }
  return array
}

test('all 198 research entries have an explicit disposition and enabled options match effective configs', () => {
  assert.equal(catalog.length, 198)
  for (const entry of catalog.filter(row => row.status.startsWith('enabled'))) {
    const owner = entry.rule === 'unused-imports/no-unused-vars' && entry.group === 'shared'
      ? 'no-unused-vars'
      : entry.rule
    const current = configs[entry.group].rules[owner]
    assert.partialDeepStrictEqual(current, normalized(entry.value), entry.rule)
  }
})

test('framework files, typed rules and unique rule owners remain scoped', async () => {
  const scoped = await makeESLint({ typescript: tsOptions, react: { files: ['react/**'] }, vue: true })
  const vue = await scoped.calculateConfigForFile(resolve(fixtureRoot, 'Safe.vue'))
  assert.equal(vue.rules['react/rules-of-hooks'], undefined)
  assert.equal(vue.rules['react/no-leaked-conditional-rendering'], undefined)
  assert.equal(vue.rules['style/max-len'][0], 0)
  assert.equal(vue.rules['vue/max-len'][0], 2)
  const ts = configs.typed
  for (const rule of [
    'max-params',
    'no-shadow',
    'no-unused-expressions',
    'no-unused-vars',
    'ts/no-unused-vars',
  ]) {
    assert.equal(ts.rules[rule][0], 0, rule)
  }
})

test('compiler and experimental checks are deliberate opt-ins', async () => {
  const opted = await makeESLint({ react: { compiler: true, experimental: true } })
  const enabled = await opted.calculateConfigForFile('sample.jsx')
  for (const rule of [
    'react-hooks/config',
    'react-hooks/gating',
    'react/web-api-no-leaked-fetch',
  ]) {
    assert.equal(enabled.rules[rule][0], 2)
    assert.equal(configs.react.rules[rule][0], 0)
  }
})

test('handwritten configuration, scripts and declarations retain strict limits and safety', async () => {
  for (const file of [
    'eslint.config.ts',
    'scripts/build.ts',
    'api.d.ts',
  ]) {
    const effective = await eslint.calculateConfigForFile(resolve(fixtureRoot, file))
    assert.equal(effective.rules.complexity[1].max, 10)
    assert.equal(effective.rules['max-lines'][1].max, 300)
    assert.equal(effective.rules['ts/no-explicit-any'][0], 2)
    assert.equal(effective.rules['unused-imports/no-unused-vars'][0], 2)
    assert.equal(effective.rules['eslint-comments/no-unlimited-disable'][0], 2)
  }
})

test('JSON, YAML and Markdown language configs do not receive JS AST rules', async () => {
  const base = await config({ typescript: false, gitignore: false })
  assert.ok(base.length > 0)
  for (const file of [
    'data.json',
    'config.yml',
    'README.md',
  ]) {
    const effective = await eslint.calculateConfigForFile(file)
    assert.equal(effective.rules.complexity, undefined)
    assert.equal(effective.rules['ztd/array-layout'], undefined)
  }
})

test('stable formatting cannot be disabled or replaced by conflicting experimental list rules', async () => {
  await assert.rejects(config({ typescript: false, stylistic: false }), /stable stylistic/u)
  await assert.rejects(config({ typescript: false, stylistic: { experimental: true } }), /stable stylistic/u)
})
