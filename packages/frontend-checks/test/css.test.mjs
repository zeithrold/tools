import assert from 'node:assert/strict'
import { mkdtemp, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { checkCss } from '../src/css.mjs'

async function fixture(files) {
  const cwd = await mkdtemp(path.join(os.tmpdir(), 'ztd-css-'))
  await Promise.all(Object.entries(files).map(([file, code]) => writeFile(path.join(cwd, file), code)))
  return cwd
}

test('checks real CSS and cross-file tokens including Tailwind declarations', async () => {
  const cwd = await fixture({
    'theme.css': '@theme { --color-action: #234567; }\n',
    'app.css': '.action { color: var(--color-action); }\n',
  })
  const report = await checkCss({ cwd, files: ['app.css'], tokenFiles: ['theme.css'] })
  assert.equal(report.status, 'passed', JSON.stringify(report))
  assert.equal(report.files.length, 1)
})

test('undefined references fail even with fallbacks; nested fallback references are checked', async () => {
  const cwd = await fixture({
    'app.css': ':root { --known: #234567; }\na { color: var(--ink, var(--also-missing)); }\n',
  })
  const report = await checkCss({ cwd, files: ['app.css'] })
  const warnings = report.warnings.filter(warning => warning.rule === 'ztd/defined-custom-property')
  assert.equal(report.status, 'failed')
  assert.equal(warnings.length, 2)
  assert.ok(warnings.some(warning => warning.text.includes('--ink')))
})

test('literal paint colors fail and token definitions preserve project branding', async () => {
  const cwd = await fixture({
    'app.css': `:root { --brand: oklch(60% 0.2 20); }
a { color: red; background: #abcdef; border: 1px solid rgb(0 0 0); }
`,
  })
  const report = await checkCss({ cwd, files: ['app.css'] })
  assert.equal(report.warnings.filter(warning => warning.rule === 'ztd/semantic-color').length, 3)
})

test('CSS syntax and invalid properties are reported by the native parser/linter', async () => {
  const cwd = await fixture({ 'invalid.css': 'a { unknown-ztd-property: 1; }\n', 'syntax.css': 'a { color: ' })
  const report = await checkCss({ cwd, files: ['invalid.css'] })
  assert.equal(report.status, 'failed')
  assert.ok(report.warnings.some(warning => warning.rule === 'property-no-unknown'))
  await assert.rejects(checkCss({ cwd, files: ['syntax.css'] }))
})

test('exact runtime properties are explicit; empty file matches block validation', async () => {
  const cwd = await fixture({ 'app.css': 'a { color: var(--runtime); }\n' })
  assert.equal((await checkCss({ cwd, files: ['app.css'], externalCustomProperties: ['--runtime'] })).status, 'passed')
  await assert.rejects(checkCss({ cwd, files: ['missing.css'] }), /matched no files/)
  await assert.rejects(checkCss({ cwd, files: ['app.css'], tokenFiles: ['missing.css'] }), /matched no files/)
})
