import { copyFile, mkdir, readFile, writeFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'

const require = createRequire(import.meta.url)
const fontRoot = path.dirname(require.resolve('@fontsource-variable/inter/package.json'))
await mkdir('dist/assets', { recursive: true })
for (const subset of ['latin', 'latin-ext']) {
  const file = `inter-${subset}-wght-normal.woff2`
  await copyFile(path.join(fontRoot, 'files', file), path.join('dist/assets', file))
}
await copyFile(path.join(fontRoot, 'LICENSE'), 'dist/assets/INTER-OFL.txt')
await copyFile('third-party/SHADCN-MIT.txt', 'dist/assets/SHADCN-MIT.txt')
const fontSource = await readFile(path.join(fontRoot, 'wght.css'), 'utf8')
const fonts = fontSource.split('/*').filter(part => /^ inter-latin(?:-ext)?-wght-normal/u.test(part))
const fontCss = fonts.map(part => `/*${part}`).join('\n').replaceAll('./files/', './assets/')
const tokens = await readFile('src/styles/tokens.css', 'utf8')
const shell = await readFile('src/styles/shell.css', 'utf8')
await writeFile('dist/styles.css', `${fontCss}
${tokens}
${shell}`)

await writeFile('dist/styles.css.d.ts', 'export {}\n')
