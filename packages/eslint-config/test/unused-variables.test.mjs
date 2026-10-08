/* eslint-disable antfu/no-top-level-await -- Native regression tests await both real parser profiles. */
import assert from 'node:assert/strict'
import { resolve } from 'node:path'
import test from 'node:test'
import { fixtureRoot, lintText, makeESLint, messagesFor } from './helpers.mjs'

const js = await makeESLint({ react: true, vue: true })
const typed = await makeESLint({
  typescript: { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') },
  react: true,
  vue: true,
})
const unusedRules = [
  'no-unused-vars',
  'unused-imports/no-unused-vars',
  'unused-imports/no-unused-imports',
]

function unusedMessages(result) {
  assert.equal(result.fatalErrorCount, 0, JSON.stringify(result.messages))
  return result.messages.filter(message => unusedRules.includes(message.ruleId))
}

test('Espree JS and JSX used values, exports and imports do not become type-only references', async () => {
  const cases = [
    ['used.mjs', 'const number = 1\nconsole.log(number)\n'],
    ['used.cjs', 'const number = 1\nmodule.exports = number\n'],
    ['used.js', 'export const number = 1\n'],
    ['used.js', 'import { basename } from \'node:path\'\nexport const name = basename(\'example\')\n'],
    ['used.jsx', 'const text = \'hello\'\nexport default function Card() { return <p>{text}</p> }\n'],
    ['used.jsx', 'import { Fragment } from \'react\'\nexport default function Card() { return <Fragment /> }\n'],
    ['used.mjsx', 'const text = \'hello\'\nexport default function Card() { return <p>{text}</p> }\n'],
    ['used.cjsx', 'const text = \'hello\'\nexport default function Card() { return <p>{text}</p> }\n'],
  ]
  for (const eslint of [js, typed]) {
    for (const [file, source] of cases) {
      assert.deepEqual(unusedMessages(await lintText(eslint, source, file)), [], file)
    }
  }
})

test('native JS still rejects unused variables, parameters, catches and used ignored names', async () => {
  for (const source of [
    'const unused = 1\n',
    'export function value(unused) { return 1 }\n',
    'try { throw new Error(\'example\') } catch (error) {}\n',
    'const _used = 1\nconsole.log(_used)\n',
  ]) {
    const findings = unusedMessages(await lintText(js, source))
    assert.equal(findings.length, 1, JSON.stringify(findings))
    assert.equal(findings[0].ruleId, 'no-unused-vars')
  }
})

test('JS variables keep strict native options while TS keeps typed ownership', async () => {
  const javascript = await typed.calculateConfigForFile('used.js')
  const typescript = await typed.calculateConfigForFile(resolve(fixtureRoot, 'safe.ts'))
  assert.partialDeepStrictEqual(javascript.rules['no-unused-vars'], typescript.rules['unused-imports/no-unused-vars'])
  assert.equal(javascript.rules['unused-imports/no-unused-vars'][0], 0)
  assert.equal(javascript.rules['unused-imports/no-unused-imports'][0], 2)
  assert.equal(typescript.rules['no-unused-vars'][0], 0)
  assert.equal(typescript.rules['unused-imports/no-unused-imports'][0], 2)
})

test('Vue without TypeScript retains native template references and detects unused script values', async () => {
  const file = resolve(fixtureRoot, 'Safe.vue')
  const source = '<script setup>\nconst text = \'hello\'\n</script>\n\n<template><p>{{ text }}</p></template>\n'
  assert.deepEqual(unusedMessages(await lintText(js, source, file)), [])
  const invalid = source.replace('const text = \'hello\'', 'const unused = \'hello\'\nconst text = \'hello\'')
  const findings = unusedMessages(await lintText(js, invalid, file))
  assert.equal(findings.length, 1)
  assert.equal(findings[0].ruleId, 'no-unused-vars')
  const ordinary = '<script>\nconst text = \'hello\'\nconsole.warn(text)\n'
    + 'export default {}\n</script>\n\n<template><p>hello</p></template>\n'
  assert.deepEqual(unusedMessages(await lintText(js, ordinary, file)), [])
})

test('typed Vue and TypeScript preserve used values and reject values used only as types', async () => {
  const [vue] = await typed.lintFiles(resolve(fixtureRoot, 'Safe.vue'))
  assert.deepEqual(unusedMessages(vue), [])
  const source = 'const value = \'example\'\nexport type Label = typeof value\n'
  const invalid = await lintText(typed, source, resolve(fixtureRoot, 'safe.ts'))
  assert.equal(messagesFor(invalid, 'unused-imports/no-unused-vars').length, 1)
})

test('unused JS imports keep the native import rejection, including ignored-prefix names', async () => {
  for (const source of [
    'import { basename } from \'node:path\'\nexport const name = \'example\'\n',
    'import path from \'node:path\'\nexport const name = \'example\'\n',
    'import * as path from \'node:path\'\nexport const name = \'example\'\n',
  ]) {
    const findings = unusedMessages(await lintText(js, source))
    assert.equal(findings.length, 2)
    assert.ok(findings.some(message => message.ruleId === 'no-unused-vars'))
    assert.ok(findings.some(message => message.ruleId === 'unused-imports/no-unused-imports'))
  }
  const ignored = 'import { basename as _unused } from \'node:path\'\nexport const name = \'example\'\n'
  const ignoredFindings = unusedMessages(await lintText(js, ignored))
  assert.equal(ignoredFindings.length, 1)
  assert.equal(ignoredFindings[0].ruleId, 'unused-imports/no-unused-imports')
})

test('unused JS import autofix remains active alongside native variable ownership', async () => {
  const fixer = await makeESLint({}, { fix: true })
  const source = 'import { basename } from \'node:path\'\n\nexport const name = \'example\'\n'
  const fixed = await lintText(fixer, source)
  assert.equal(fixed.errorCount, 0, JSON.stringify(fixed.messages))
  assert.ok(!fixed.output.includes('basename'))
})

test('Markdown JS and JSX examples retain the native partial-fragment policy and other checks', async () => {
  for (const language of ['js', 'jsx']) {
    const source = [
      '# Example',
      '',
      `\`\`\`${language}`,
      'import { basename } from \'node:path\'',
      'const unused = 1',
      'const ids = [1, 2, 3]',
      '\`\`\`',
      '',
    ].join('\n')
    const result = await lintText(typed, source, 'README.md')
    assert.deepEqual(unusedMessages(result), [])
    assert.equal(messagesFor(result, 'ztd/array-layout').length, 1)
  }
  const real = await lintText(typed, 'const unused = 1\n', 'scripts/actual.mjs')
  assert.equal(messagesFor(real, 'no-unused-vars').length, 1)
})
