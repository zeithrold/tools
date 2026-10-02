import assert from 'node:assert/strict'
import { mkdtemp, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { test } from 'node:test'
import { compile } from 'tailwindcss'
import { checkCss } from '../src/css.mjs'

const systemDark = `@custom-variant dark {
  :root[data-frontend-mode="dark"] & { @slot; }

  @media (prefers-color-scheme: dark) {
    :root[data-frontend-mode="system"] & { @slot; }
  }
}
`
async function inspect(code) {
  const cwd = await mkdtemp(path.join(os.tmpdir(), 'ztd-tailwind-css-'))
  await writeFile(path.join(cwd, 'app.css'), code)
  return checkCss({ cwd, files: ['app.css'] })
}

test('Tailwind block custom variants provide the nesting root, including nested system media', async () => {
  const report = await inspect(systemDark)
  assert.equal(report.status, 'passed', JSON.stringify(report))
})

test('unscoped nesting outside custom variants still fails at root and in media', async () => {
  const report = await inspect(`${systemDark}
& { display: block; }

@media (prefers-color-scheme: dark) {
  & { display: block; }
}
`)
  assert.equal(report.status, 'failed')
  assert.equal(report.warnings.filter(w => w.rule === 'nesting-selector-no-missing-scoping-root').length, 2)
})

test('custom variant nesting exemption preserves property, token and color checks', async () => {
  const report = await inspect(`@custom-variant dark {
  & {
    color: var(--missing);
    background: red;
  }
}

a { unknown-ztd-property: 1; }
`)
  for (const rule of [
    'property-no-unknown',
    'ztd/defined-custom-property',
    'ztd/semantic-color',
  ]) {
    assert.ok(report.warnings.some(warning => warning.rule === rule), rule)
  }
  assert.equal(report.status, 'failed')
})

test('the exact system-dark variant compiles with native Tailwind 4.3.3', async () => {
  const compiler = await compile(`${systemDark}\n@tailwind utilities;`)
  const output = compiler.build(['dark:block'])
  assert.ok(output.includes(':root[data-frontend-mode="dark"] .dark\\:block'))
  assert.ok(output.includes('@media (prefers-color-scheme: dark)'))
  assert.ok(output.includes(':root[data-frontend-mode="system"] .dark\\:block'))
  assert.ok(!output.includes('&'), 'Native compilation resolves the variant placeholder')
})
