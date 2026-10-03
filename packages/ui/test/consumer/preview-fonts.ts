import type { ServerResponse } from 'node:http'
import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

// Test server only. Neither this adapter nor its cache is in the source registry.
export function localFontPreview(directory: string | undefined) {
  const pending = new Map<string, Promise<Uint8Array>>()
  async function font(filename: string): Promise<Uint8Array> {
    if (!directory) throw new Error('Local-font preview was not explicitly selected')
    const file = path.join(directory, filename)
    try {
      return await readFile(file)
    }
    catch (error) {
      if (!(error instanceof Error && 'code' in error && error.code === 'ENOENT')) throw error
    }
    const files: Record<string, string> = JSON.parse(await readFile(path.join(directory, 'files.json'), 'utf8'))
    const url = files[filename]
    if (!url || new URL(url).origin !== 'https://fonts.gstatic.com') throw new Error('Unknown preview font')
    const response = await fetch(url, { signal: AbortSignal.timeout(20_000) })
    if (!response.ok) throw new Error(`Preview font download returned ${response.status}`)
    const bytes = new Uint8Array(await response.arrayBuffer())
    await writeFile(file, bytes)
    return bytes
  }
  return async (url: URL, response: ServerResponse): Promise<boolean> => {
    if (!directory || !url.pathname.startsWith('/__local-noto-preview/')) return false
    const filename = path.basename(url.pathname)
    if (filename === 'fonts.css') {
      response.setHeader('Content-Type', 'text/css')
      response.end(await readFile(path.join(directory, filename)))
    }
    else if (/^[a-f0-9]{64}\.woff2$/u.test(filename)) {
      response.setHeader('Content-Type', 'font/woff2')
      if (!pending.has(filename)) pending.set(filename, font(filename))
      response.end(await pending.get(filename))
    }
    else {
      response.statusCode = 404
      response.end()
    }
    return true
  }
}
