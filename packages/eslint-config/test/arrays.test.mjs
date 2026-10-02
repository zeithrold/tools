/* eslint-disable antfu/no-top-level-await -- Node's test runner awaits ESM setup before registering the tests. */
import assert from 'node:assert/strict'
import test from 'node:test'
import { lintText, makeESLint, messagesFor } from './helpers.mjs'

const eslint = await makeESLint({}, { fix: true })
const check = await makeESLint()

async function fix(source) {
  const first = await lintText(eslint, source)
  const output = first.output ?? source
  assert.equal(messagesFor(first, 'ztd/array-layout').length, 0)
  assert.equal((await lintText(eslint, output)).output, undefined)
  return output
}

test('array threshold includes exactly three elements', async () => {
  for (const source of [
    '[]',
    '[1]',
    '[1, 2]',
  ]) {
    const result = await lintText(check, `export const values = ${source}\n`)
    assert.equal(messagesFor(result, 'ztd/array-layout').length, 0)
  }
  for (const source of ['[1, 2, 3]', '[1, 2, 3, 4]']) {
    const result = await lintText(check, `export const values = ${source}\n`)
    assert.equal(messagesFor(result, 'ztd/array-layout').length, 1)
    const output = await fix(`export const values = ${source}\n`)
    assert.match(output, /\[\n {2}1,\n {2}2,\n {2}3,/u)
  }
})

test('safe short and empty arrays collapse', async () => {
  assert.equal(await fix('export const values = [\n  1,\n  2,\n]\n'), 'export const values = [1, 2]\n')
  assert.equal(await fix('export const values = [\n]\n'), 'export const values = []\n')
})

test('short complex, nested, call and spread arrays expand', async () => {
  for (const source of [
    '[{ id: 1 }]',
    '[[1, 2], [3, 4]]',
    '[getValue()]',
    '[...other]',
  ]) {
    const output = await fix(`export const values = ${source}\n`)
    assert.match(output, /values = \[\n/u)
  }
})

test('over-wide pairs stay expanded and single-line over-wide pairs expand', async () => {
  const a = `first${'x'.repeat(55)}`
  const b = `second${'x'.repeat(55)}`
  const expanded = `export const values = [\n  ${a},\n  ${b},\n]\n`
  assert.equal(await fix(expanded), expanded)
  assert.equal(await fix(`export const values = [${a}, ${b}]\n`), expanded)
})

test('comments retain their text and association across repeated fixes', async () => {
  const commented = 'export const values = [\n  1, // first\n  2,\n]\n'
  assert.equal(await fix(commented), commented)
  const source = 'export const values = [1 /* first */, 2, 3]\n'
  const output = await fix(source)
  assert.match(output, /1 \/\* first \*\/,/u)
  const unsafe = await lintText(check, 'export const values = [/* before */ 1, 2, 3]\n')
  assert.equal(messagesFor(unsafe, 'ztd/array-layout')[0].fix, undefined)
})

test('CRLF fixes preserve CRLF and values', async () => {
  const output = await fix('export const values = [1, 2, 3]\r\n')
  assert.match(output, /\[\r\n/u)
  assert.equal(output.replaceAll('\r\n', '').includes('\n'), false)
  const module = await import(`data:text/javascript,${encodeURIComponent(output)}`)
  assert.deepEqual(module.values, [
    1,
    2,
    3,
  ])
})

test('sparse arrays remain prohibited and destructuring is separate', async () => {
  assert.equal(messagesFor(await lintText(check, 'export const values = [1,,3]'), 'no-sparse-arrays').length, 1)
  const result = await lintText(check, 'const [a, b, c] = values\nexport { a, b, c }\n')
  assert.equal(messagesFor(result, 'ztd/array-layout').length, 0)
})

test('short-array collapse respects the complete-line boundary at 120 and 121', async () => {
  const a = 'a'.repeat(47)
  const b = 'b'.repeat(47)
  const compact = `export const values = [${a}, ${b}]\n`
  assert.equal(compact.trimEnd().length, 120)
  assert.equal(await fix(`export const values = [\n  ${a},\n  ${b},\n]\n`), compact)
  assert.match(await fix(compact.replace(b, `${b}b`)), /values = \[\n/u)
})
