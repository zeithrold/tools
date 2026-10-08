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
for (const license of [
  'SHADCN-MIT.txt',
  'RADIX-MIT.txt',
  'REACT-REMOVE-SCROLL-BAR-MIT.txt',
  'VAUL-MIT.txt',
  'NOTO-SANS-OFL.txt',
  'NOTO-SANS-SC-OFL.txt',
  'NOTO-SANS-JP-OFL.txt',
  'NOTO-SANS-KR-OFL.txt',
  'NOTO-SERIF-OFL.txt',
  'NOTO-SERIF-SC-OFL.txt',
  'NOTO-SERIF-JP-OFL.txt',
  'NOTO-SERIF-KR-OFL.txt',
  'NOTO-EMOJI-OFL.txt',
  'NOTO-COLOR-EMOJI-OFL.txt',
]) { await copyFile(`third-party/${license}`, `dist/assets/${license}`) }
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
