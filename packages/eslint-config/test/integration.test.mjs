/* eslint-disable antfu/no-top-level-await -- Node's test runner awaits ESM setup before registering the tests. */
import assert from 'node:assert/strict'
import { resolve } from 'node:path'
import test from 'node:test'
// eslint-disable-next-line antfu/no-import-dist -- These integration tests exercise the published build.
import config from '../dist/index.js'
import { fixtureRoot, lintText, makeESLint, messagesFor } from './helpers.mjs'

const options = { typescript: { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') }, react: true, vue: true }
const eslint = await makeESLint(options)

test('real TypeScript, React TSX and typed Vue fixtures pass', async () => {
  for (const file of [
    'safe.ts',
    'Button.tsx',
    'Safe.vue',
  ]) {
    const [result] = await eslint.lintFiles(resolve(fixtureRoot, file))
    assert.deepEqual(result.messages, [], file)
  }
})

test('typed safety restores explicit any, unsafe access, promises, booleans and numeric sort', async () => {
  const [result] = await eslint.lintFiles(resolve(fixtureRoot, 'unsafe.ts'))
  for (const rule of [
    'ts/no-explicit-any',
    'ts/no-unsafe-member-access',
    'ts/no-floating-promises',
    'ts/strict-boolean-expressions',
    'ts/require-array-sort-compare',
  ]) {
    assert.ok(messagesFor(result, rule).length > 0, rule)
  }
  assert.equal(result.fatalErrorCount, 0)
})

test('React reports hooks, dependencies, button semantics and Promise event handlers', async () => {
  const [result] = await eslint.lintFiles(resolve(fixtureRoot, 'BadButton.tsx'))
  for (const rule of [
    'react/rules-of-hooks',
    'react/exhaustive-deps',
    'react/dom-no-missing-button-type',
    'ts/no-misused-promises',
  ]) {
    assert.ok(messagesFor(result, rule).length > 0, rule)
  }
  assert.equal(result.fatalErrorCount, 0)
  assert.equal(result.messages.some(message => message.ruleId?.startsWith('jsx-a11y/')), false)
})

test('typed Vue covers promises, raw HTML, accessibility and template arrays', async () => {
  const [result] = await eslint.lintFiles(resolve(fixtureRoot, 'Unsafe.vue'))
  for (const rule of [
    'ts/no-floating-promises',
    'vue/no-v-html',
    'vue/html-button-has-type',
    'vue-a11y/alt-text',
    'ztd/array-layout',
  ]) {
    assert.ok(messagesFor(result, rule).length > 0, rule)
  }
  assert.equal(result.fatalErrorCount, 0)
})

test('Vue template and script array fixes converge with antfu formatting', async () => {
  const fixer = await makeESLint(options, { fix: true })
  const [initial] = await eslint.lintFiles(resolve(fixtureRoot, 'Array.vue'))
  assert.equal(messagesFor(initial, 'ztd/array-layout').length, 2)
  const [fixed] = await fixer.lintFiles(resolve(fixtureRoot, 'Array.vue'))
  assert.equal(fixed.errorCount, 0)
  const repeat = await lintText(fixer, fixed.output, resolve(fixtureRoot, 'Array.vue'))
  assert.equal(repeat.output, undefined)
})

test('TypeScript array values expand but tuple types and patterns have separate policy', async () => {
  const source = 'export type RGB = [number, number, number]\nexport const color = [1, 2, 3] as const\n'
  const result = await lintText(eslint, source, resolve(fixtureRoot, 'safe.ts'))
  assert.equal(messagesFor(result, 'ztd/array-layout').length, 1)
})

test('bad and missing TS configuration fails explicitly', async () => {
  await assert.rejects(config({ typescript: { tsconfigPath: '/does-not-exist/tsconfig.json' } }), /cannot read/u)
})

test('Vue file length counts the full SFC, including its template', async () => {
  const source = count => [
    '<script setup lang="ts">',
    'const text = \'hello\'',
    '</script>',
    '<template>',
    ...Array.from({ length: count - 5 }).fill('  <p>{{ text }}</p>'),
    '</template>',
  ].join('\n')
  const file = resolve(fixtureRoot, 'Safe.vue')
  assert.equal(messagesFor(await lintText(eslint, source(300), file), 'max-lines').length, 0)
  assert.equal(messagesFor(await lintText(eslint, source(301), file), 'max-lines').length, 1)
})
