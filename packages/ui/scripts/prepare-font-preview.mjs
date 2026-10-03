import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

// Explicit Cloud-only preview. This script and its ignored cache are not registry files.
const directory = path.resolve('.artifacts/local-noto-preview')
const source = await readFile('src/styles/fonts.css', 'utf8')
const apiUrl = source.match(/@import url\("([^"]+)"\)/u)?.[1]
assert.ok(apiUrl)
assert.equal(new URL(apiUrl).origin, 'https://fonts.googleapis.com')
const response = await fetch(apiUrl, {
  headers: {
    'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/151.0.0.0 Safari/537.36',
  },
  signal: AbortSignal.timeout(20_000),
})
assert.equal(response.status, 200)
const remoteCss = await response.text()
const files = {}
const localCss = remoteCss.replace(/url\((https:[^)]+)\)/gu, (_match, url) => {
  assert.equal(new URL(url).origin, 'https://fonts.gstatic.com')
  assert.ok(url.endsWith('.woff2'))
  const filename = `${createHash('sha256').update(url).digest('hex')}.woff2`
  files[filename] = url
  return `url(/__local-noto-preview/${filename})`
})
await mkdir(directory, { recursive: true })
await writeFile(path.join(directory, 'fonts.css'), localCss)
await writeFile(path.join(directory, 'files.json'), JSON.stringify(files))
await writeFile(path.join(directory, 'scope.json'), JSON.stringify({
  scope: 'Authorized local-font preview only; actual browser Google Fonts loading is unverified',
  apiUrl,
  cssSha256: createHash('sha256').update(remoteCss).digest('hex'),
  fontFiles: Object.keys(files).length,
  tlsVerification: 'Normal Node TLS with the executor-provided CA and proxy; no TLS bypass',
}, null, 2))
process.stdout.write(`LOCAL FONT PREVIEW ONLY: ${directory}\nProduction registry retains Google Fonts API.\n`)
