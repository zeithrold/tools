/* eslint-disable antfu/no-top-level-await -- Node's test runner awaits the real typed config before tests. */
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import test from 'node:test'
import { fixtureRoot, lintText, makeESLint, messagesFor } from './helpers.mjs'

const root = resolve(fixtureRoot, 'markdown-consumer')
const project = resolve(root, 'tsconfig.json')
const options = { typescript: { tsconfigPath: project, tsconfigRootDir: root }, react: true }
const eslint = await makeESLint(options)

test('a strict application project lints README TypeScript and TSX fences', async () => {
  const [result] = await eslint.lintFiles(resolve(root, 'README.md'))
  assert.deepEqual(result.messages, [])
})

test('virtual examples have syntax checks while actual application files retain typed checks', async () => {
  for (const filename of [
    'README.md/0_0.ts',
    'README.md/0_1.tsx',
    'Card.astro/0.ts',
  ]) {
    const effective = await eslint.calculateConfigForFile(resolve(root, filename))
    assert.equal(effective.languageOptions.parserOptions.project, undefined, filename)
    assert.equal(effective.languageOptions.parserOptions.projectService, undefined, filename)
    assert.equal(effective.rules['ts/no-floating-promises'], undefined, filename)
    assert.equal(effective.rules['ts/no-explicit-any'][0], 2, filename)
    assert.equal(effective.rules['ztd/array-layout'][0], 2, filename)
  }
  const app = await eslint.calculateConfigForFile(resolve(root, 'src/Card.tsx'))
  assert.equal(app.languageOptions.parserOptions.project, project)
  assert.equal(app.languageOptions.parserOptions.projectService, false)
  assert.equal(app.rules['ts/no-floating-promises'][0], 2)
  assert.equal(app.rules['ts/strict-boolean-expressions'][0], 2)
  assert.equal(app.rules['react/no-leaked-conditional-rendering'][0], 2)
})

test('Markdown markup, React semantics, TypeScript syntax and array rules remain enforced', async () => {
  const source = [
    '# Example',
    '',
    '[Empty destination]()',
    '',
    '```tsx',
    'export default function Example() {',
    '  return <button>Example</button>',
    '}',
    '```',
    '',
    '```ts',
    'export const ids = [1, 2, 3]',
    'export const unsafe: any = 1',
    '```',
    '',
  ].join('\n')
  const result = await lintText(eslint, source, resolve(root, 'README.md'))
  assert.equal(result.fatalErrorCount, 0)
  for (const rule of [
    'markdown/no-empty-links',
    'react/dom-no-missing-button-type',
    'ztd/array-layout',
    'ts/no-explicit-any',
  ]) {
    assert.ok(messagesFor(result, rule).length > 0, rule)
  }
})

test('real application promises still fail and out-of-project TS is not given syntax-only fallback', async () => {
  const card = await readFile(resolve(root, 'src/Card.tsx'), 'utf8')
  const result = await lintText(eslint, `${card}\nvoid Promise.resolve(1)\n`, resolve(root, 'src/Card.tsx'))
  assert.equal(result.fatalErrorCount, 0)
  assert.equal(messagesFor(result, 'ts/no-floating-promises').length, 1)
  const outside = await lintText(eslint, 'export const value = 1\n', resolve(root, 'outside.ts'))
  assert.equal(outside.fatalErrorCount, 1)
  assert.match(outside.messages[0].message, /TSConfig does not include this file/u)
})

test('typed Vitest checks exclude Markdown virtual files without excluding real test files', async () => {
  const tests = await makeESLint({ ...options, test: true })
  const virtual = await tests.calculateConfigForFile(resolve(root, '__tests__/README.md/0_0.tsx'))
  assert.equal(virtual.languageOptions.parserOptions.project, undefined)
  assert.equal(virtual.rules['test/unbound-method'], undefined)
  assert.equal(virtual.rules['test/no-only-tests'][0], 2)
  const actual = await tests.calculateConfigForFile(resolve(root, 'src/card.test.ts'))
  assert.equal(actual.languageOptions.parserOptions.project, project)
  assert.equal(actual.rules['test/unbound-method'][0], 2)
  const readme = await readFile(resolve(root, 'README.md'), 'utf8')
  const result = await lintText(tests, readme, resolve(root, '__tests__/README.md'))
  assert.deepEqual(result.messages, [])
})
