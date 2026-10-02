import assert from 'node:assert/strict'
import { resolve } from 'node:path'
import test from 'node:test'
import { fixtureRoot, lintText, makeESLint, messagesFor } from './helpers.mjs'

const metadata = [
  'export const metadata = { title: \'Example\' }',
  'export default function Page() {',
  '  return <main>Example</main>',
  '}',
  '',
].join('\n')

test('published 0.1.0 behavior is retained for framework-neutral React', async () => {
  const eslint = await makeESLint({ react: true })
  const result = await lintText(eslint, metadata, 'app/layout.jsx')
  assert.equal(messagesFor(result, 'react-refresh/only-export-components').length, 1)
  const ordinary = await eslint.calculateConfigForFile('components/Card.jsx')
  assert.deepEqual(ordinary.rules['react-refresh/only-export-components'], [
    2,
    { allowConstantExport: false, allowExportNames: [] },
  ])
})

test('Next and vinext typed server page/layout fixtures pass', async () => {
  for (const framework of ['next', 'vinext']) {
    const eslint = await makeESLint({
      typescript: { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') },
      react: { framework, appDir: 'test/fixtures/app' },
    })
    const [layout] = await eslint.lintFiles(resolve(fixtureRoot, 'app/layout.tsx'))
    assert.deepEqual(layout.messages, [])
    const [plain] = await eslint.lintFiles(resolve(fixtureRoot, 'app/plain/page.ts'))
    assert.deepEqual(plain.messages, [])
    const nested = await makeESLint({
      typescript: { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') },
      react: { framework, appDir: 'test/fixtures/src/app' },
    })
    const [page] = await nested.lintFiles(resolve(fixtureRoot, 'src/app/(marketing)/page.tsx'))
    assert.deepEqual(page.messages, [])
  }
})

test('route scopes handle root, src, groups, parallel slots and dynamic segments', async () => {
  const eslint = await makeESLint({ react: { framework: 'vinext' } })
  for (const path of [
    'app/layout.jsx',
    'app/(site)/[slug]/page.jsx',
    'src/app/@modal/(.)photo/[id]/page.jsx',
    'src/app/(site)/layout.js',
  ]) {
    const result = await lintText(eslint, metadata, path)
    assert.equal(result.errorCount, 0, JSON.stringify(result.messages))
  }
})

test('ordinary components and private/non-route files keep mixed-export errors', async () => {
  const eslint = await makeESLint({ react: { framework: 'next' } })
  for (const path of [
    'components/Card.jsx',
    'components/app/layout.jsx',
    'app/components/Card.jsx',
    'app/loading.jsx',
    'app/_components/page.jsx',
    'pages/page.jsx',
  ]) {
    const result = await lintText(eslint, metadata, path)
    assert.equal(messagesFor(result, 'react-refresh/only-export-components').length, 1, path)
  }
})

test('client route directives retain mixed-export errors including constant exports', async () => {
  const eslint = await makeESLint({ react: { framework: 'vinext' } })
  for (const declaration of [
    'export const metadata = { title: \'Example\' }',
    'export const runtime = \'edge\'',
    'export const revalidate = 60',
  ]) {
    const source = [
      '// Leading comments do not change directives.',
      '\'use client\'',
      declaration,
      metadata.slice(metadata.indexOf('export default')),
    ].join('\n')
    const result = await lintText(eslint, source, 'app/page.jsx')
    assert.equal(messagesFor(result, 'ztd/app-router-exports').length, 1)
  }
})

test('typed client directives and non-JSX route filenames enforce the export boundary', async () => {
  const typed = await makeESLint({
    typescript: { tsconfigPath: resolve(fixtureRoot, 'tsconfig.json') },
    react: { framework: 'vinext', appDir: 'test/fixtures/app' },
  })
  const path = 'test/fixtures/app/layout.tsx'
  const result = await lintText(typed, `'use client'\n${metadata}`, path)
  assert.equal(messagesFor(result, 'ztd/app-router-exports').length, 1)
  const js = await makeESLint({ react: { framework: 'vinext' } })
  const source = [
    'export const metadata = { title: \'Example\' }',
    'export function helper() { return 1 }',
    'export default function Page() { return null }',
    '',
  ].join('\n')
  const plain = await lintText(js, source, 'app/page.js')
  assert.equal(messagesFor(plain, 'ztd/app-router-exports').length, 1)
})

test('server routes allow recognized framework exports and reject arbitrary exports', async () => {
  const eslint = await makeESLint({ react: { framework: 'next' } })
  const component = metadata.slice(metadata.indexOf('export default'))
  for (const name of [
    'dynamic',
    'dynamicParams',
    'revalidate',
    'fetchCache',
    'runtime',
    'preferredRegion',
    'maxDuration',
    'instant',
    'prefetch',
  ]) {
    const result = await lintText(eslint, `export const ${name} = 'example'\n${component}`, 'app/page.jsx')
    assert.equal(messagesFor(result, 'ztd/app-router-exports').length, 0, name)
  }
  for (const name of [
    'utility',
    'generateImageMetadata',
    'generateSitemaps',
    'arbitraryConstant',
  ]) {
    const result = await lintText(eslint, `export const ${name} = 'example'\n${component}`, 'app/page.jsx')
    assert.equal(messagesFor(result, 'ztd/app-router-exports').length, 1, name)
  }
  const wildcard = await lintText(eslint, `export * from './shared.js'\n${component}`, 'app/page.jsx')
  assert.equal(messagesFor(wildcard, 'ztd/app-router-exports').length, 1)
  const reexport = await lintText(eslint, `export { helper } from './shared.js'\n${component}`, 'app/page.jsx')
  assert.equal(messagesFor(reexport, 'ztd/app-router-exports').length, 1)
})

test('vinext does not inherit Next-only exports and React file scopes remain respected', async () => {
  const eslint = await makeESLint({ react: { framework: 'vinext', files: ['app/selected/**'] } })
  const outside = await eslint.calculateConfigForFile('app/other/page.jsx')
  assert.equal(outside.rules['ztd/app-router-exports'], undefined)
  const source = `export const prefetch = 'auto'\n${metadata.slice(metadata.indexOf('export default'))}`
  const result = await lintText(eslint, source, 'app/selected/page.jsx')
  assert.equal(messagesFor(result, 'ztd/app-router-exports').length, 1)
})

test('explicit appDir scopes a monorepo integration and unsafe scopes fail clearly', async () => {
  const eslint = await makeESLint({ react: { framework: 'vinext', appDir: 'apps/web/app' } })
  assert.equal((await lintText(eslint, metadata, 'apps/web/app/layout.jsx')).errorCount, 0)
  const outside = await lintText(eslint, metadata, 'app/layout.jsx')
  assert.equal(messagesFor(outside, 'react-refresh/only-export-components').length, 1)
  for (const react of [
    { appDir: 'app' },
    { framework: 'vinext', appDir: '**/app' },
    { framework: 'vinext', appDir: '../app' },
    { framework: 'vinext', appDir: '/app' },
    { framework: 'vinext', appDir: 12 },
    { framework: 'unknown' },
  ]) {
    await assert.rejects(makeESLint({ react }), /react\.(?:appDir|framework)/u)
  }
})
