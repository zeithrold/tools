import assert from 'node:assert/strict'
import { mkdtemp, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { checkCss } from '../src/css.mjs'

async function inspect(code) {
  const cwd = await mkdtemp(path.join(os.tmpdir(), 'ztd-class-tokens-'))
  await writeFile(path.join(cwd, 'tokens.css'), ':root { --known: #123456; }')
  await writeFile(path.join(cwd, 'controls.tsx'), code)
  return checkCss({ cwd, files: ['tokens.css'], classFiles: ['controls.tsx'] })
}
test('static semantic classes and known arbitrary variables pass', async () => {
  const report = await inspect('const classes = "bg-surface text-control text-foreground bg-[var(--known)]";')
  assert.equal(report.status, 'passed', JSON.stringify(report))
})
test('palette and arbitrary colors fail inside TSX literal and template fragments', async () => {
  const code = 'const tone = "bg-red-500"; const node = <div className={`hover:text-[#abc] '
    + '$' + '{tone} border-[rgb(1_2_3)]`} />;'
  const report = await inspect(code)
  assert.equal(report.status, 'failed')
  assert.equal(report.warnings.filter(w => w.rule === 'ztd/semantic-class-color').length, 3)
})
test('arbitrary missing custom properties fail in bracket and shorthand utilities', async () => {
  const report = await inspect('const classes = "bg-[var(--missing)] text-(--other)";')
  assert.equal(report.warnings.filter(w => w.rule === 'ztd/defined-class-custom-property').length, 2)
})
test('comments and non-color utility literals are not color findings', async () => {
  const report = await inspect('// bg-red-500\n'
    + 'const classes = "text-sm font-serif border-0 ring-2 grid-cols-[1fr_auto]";')
  assert.equal(report.status, 'passed', JSON.stringify(report))
})
test('explicit class file globs must match rather than silently dropping coverage', async () => {
  const cwd = await mkdtemp(path.join(os.tmpdir(), 'ztd-missing-class-'))
  await writeFile(path.join(cwd, 'tokens.css'), ':root { --known: #123456; }')
  await assert.rejects(checkCss({ cwd, files: ['tokens.css'], classFiles: ['missing.tsx'] }), /matched no files/u)
})
