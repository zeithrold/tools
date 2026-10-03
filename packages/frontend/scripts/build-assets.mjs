import { copyFile, mkdir, readFile, writeFile } from 'node:fs/promises'

await mkdir('dist/assets', { recursive: true })
const fonts = await readFile('src/styles/fonts.css', 'utf8')
await copyFile('third-party/SHADCN-MIT.txt', 'dist/assets/SHADCN-MIT.txt')
for (const family of [
  'SANS',
  'SANS-SC',
  'SANS-JP',
  'SANS-KR',
  'EMOJI',
]) {
  await copyFile(`third-party/NOTO-${family}-OFL.txt`, `dist/assets/NOTO-${family}-OFL.txt`)
}
const tokens = await readFile('src/styles/tokens.css', 'utf8')
const shell = await readFile('src/styles/shell.css', 'utf8')
const motion = await readFile('src/styles/motion.css', 'utf8')
await writeFile('dist/styles.css', `${fonts}
${tokens}
${shell}
${motion}`)

await writeFile('dist/styles.css.d.ts', 'export {}\n')
