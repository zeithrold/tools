import assert from 'node:assert/strict'
import { readdir, readFile } from 'node:fs/promises'
import { test } from 'node:test'

test('Go and frontend CI are separate workflows without a custom selector', async () => {
  const base = new URL('../.github/workflows/', import.meta.url)
  const go = await readFile(new URL('ci.yml', base), 'utf8')
  const frontend = await readFile(new URL('frontend-ci.yml', base), 'utf8')
  assert.ok(!go.includes('playwright'), 'Go CI does not install or invoke browsers')
  assert.ok(go.includes('go test ./...'))
  assert.ok(frontend.includes('    paths:'), 'GitHub selects frontend changes natively')
  assert.ok(frontend.includes('if: matrix.node == 24'))
  for (const workflow of [go, frontend]) {
    assert.ok(!workflow.includes('workflow-selection'))
    assert.ok(!workflow.includes('VERIFY_FRONTEND'))
    assert.ok(!workflow.includes('  scope:'))
  }
})

test('external Actions use consistent full commit SHA pins and version comments', async () => {
  const directory = new URL('../.github/workflows/', import.meta.url)
  const pins = new Map()
  let checked = 0
  for (const filename of (await readdir(directory)).filter(name => /\.ya?ml$/.test(name))) {
    const lines = (await readFile(new URL(filename, directory), 'utf8')).split('\n')
    for (const [index, line] of lines.entries()) {
      const match = /^\s*(?:-\s*)?uses:\s*['"]?([^\s'"#]+)['"]?(?:\s+#\s*(.*))?$/.exec(line)
      if (!match || match[1].startsWith('./')) {
        continue
      }
      const location = `${filename}:${index + 1}`
      assert.match(match[1], /^[^@\s]+@[a-f0-9]{40}$/, `${location}: pin external Actions to a reviewed full commit SHA`)
      assert.match(match[2] ?? '', /^v\d+(?:\.\d+)*(?:\s|$)/, `${location}: document the release version`)
      const [action, sha] = match[1].split('@')
      assert.ok(!pins.has(action) || pins.get(action) === sha, `${location}: ${action} must use the same reviewed SHA across workflows`)
      pins.set(action, sha)
      checked += 1
    }
  }
  assert.ok(checked > 0, 'No external Action references were checked')
})
