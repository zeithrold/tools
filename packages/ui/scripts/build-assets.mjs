import { copyFile, mkdir, readdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { compile } from '@tailwindcss/node'

async function candidates(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const groups = await Promise.all(entries.map(async (entry) => {
    const file = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      return candidates(file)
    }
    if (!/\.tsx?$/u.test(entry.name)) {
      return []
    }
    return (await readFile(file, 'utf8')).match(/[^\s"'`]+/gu) ?? []
  }))
  return groups.flat()
}
await mkdir('dist/assets', { recursive: true })
// third-party/ is the single source of truth for shipped notices: it holds one original license
// per copied or patched dependency, and holds nothing for work served by its upstream host.
// scripts/pack-smoke.mjs asserts each of these files reaches the packed artifact unchanged.
for (const notice of await readdir('third-party')) {
  await copyFile(`third-party/${notice}`, `dist/assets/${notice}`)
}
const compiler = await compile('@import "tailwindcss";\n@import "./src/tailwind.css";', {
  base: process.cwd(),
  onDependency: () => {},
})
const compiled = compiler.build(await candidates('src'))
const imports = [
  ...compiled.matchAll(/@import (?:url\([^;]+|"https?:[^"]+");/gu),
].map(match => match[0])
const body = compiled.replace(/@import (?:url\([^;]+|"https?:[^"]+");/gu, '')
await writeFile('dist/styles.css', `${imports.join('\n')}\n${body}`)
await copyFile('src/styles/fonts-serif.css', 'dist/fonts-serif.css')
await writeFile('dist/styles.css.d.ts', 'export {}\n')
await writeFile('dist/fonts-serif.css.d.ts', 'export {}\n')
