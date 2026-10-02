/* eslint-disable antfu/no-top-level-await -- Node's test runner awaits ESM setup before registering the tests. */
import assert from 'node:assert/strict'
import test from 'node:test'
import { lintText, makeESLint, messagesFor } from './helpers.mjs'

const eslint = await makeESLint()

async function checkBoundary(rule, atLimit, overLimit) {
  assert.equal(messagesFor(await lintText(eslint, atLimit), rule).length, 0)
  assert.equal(messagesFor(await lintText(eslint, overLimit), rule).length, 1)
}

function branches(count) {
  const lines = Array.from({ length: count }, (_, index) => `if (v === ${index}) { value++ }`)
  return `export function f(v) { let value = 0; ${lines.join('\n')}; return value }`
}

test('classic cyclomatic complexity accepts 10 and rejects 11', async () => {
  await checkBoundary('complexity', branches(9), branches(10))
})

test('cognitive complexity accepts 15 and rejects 16', async () => {
  const nested = `${'if (v) {\n'.repeat(5)}value++\n${'}\n'.repeat(5)}`
  const wrap = body => `export function f(v) { let value = 0; ${body}; return value }`
  await checkBoundary('sonarjs/cognitive-complexity', wrap(nested), wrap(`${nested}\nif (v) { value++ }`))
})

function functionLines(count) {
  return [
    'export function f() {',
    'let value = 0',
    ...Array.from({ length: count - 4 }).fill('value += 1'),
    'return value',
    '}',
  ].join('\n')
}

test('function length accepts 60 effective lines and rejects 61, including IIFEs', async () => {
  await checkBoundary('max-lines-per-function', functionLines(60), functionLines(61))
  const commented = functionLines(60).replace('let value = 0', 'let value = 0\n\n// comment')
  const result = await lintText(eslint, commented)
  assert.equal(messagesFor(result, 'max-lines-per-function').length, 0)
  const iife = functionLines(61).replace('export function f()', '(function()')
  assert.equal(messagesFor(await lintText(eslint, `${iife})()`), 'max-lines-per-function').length, 1)
})

function fileLines(count) {
  return Array.from({ length: count }, (_, index) => `export const v${index} = ${index}`).join('\n')
}

test('file length accepts 300 effective lines and rejects 301', async () => {
  await checkBoundary('max-lines', fileLines(300), fileLines(301))
  const result = await lintText(eslint, `${fileLines(300)}\n\n// excluded comment\n`)
  assert.equal(messagesFor(result, 'max-lines').length, 0)
})

function lineWidth(width) {
  return `export const label = '${'a'.repeat(width - 23)}'`
}

test('line width accepts 120 and rejects 121 for strings and comments', async () => {
  assert.equal(lineWidth(120).length, 120)
  await checkBoundary('style/max-len', lineWidth(120), lineWidth(121))
  await checkBoundary('style/max-len', `// ${'a'.repeat(117)}`, `// ${'a'.repeat(118)}`)
})

test('depth 4 and parameter count 4 are inclusive', async () => {
  const nesting = depth => `export function f(v) { `
    + `${'if(v){'.repeat(depth)} f(false) ${'}'.repeat(depth)} }`
  await checkBoundary('max-depth', nesting(4), nesting(5))
  const four = 'export function f(a,b,c,d) { return a+b+c+d }'
  const five = 'export function f(a,b,c,d,e) { return a+b+c+d+e }'
  await checkBoundary('max-params', four, five)
})
