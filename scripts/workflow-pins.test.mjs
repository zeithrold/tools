import assert from 'node:assert/strict'
import { readdir, readFile } from 'node:fs/promises'
import { test } from 'node:test'

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
