import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { consumerSmoke } from './consumer-smoke.mjs'

const patch = '@ztd-me__frontend-checks@0.1.1.patch'
const receipt = JSON.parse(await readFile('patches/css-first-candidate.json', 'utf8'))
assert.equal(createHash('sha256').update(await readFile(`patches/${patch}`)).digest('hex'), receipt.patchSha256)

async function beforeInstall(consumer) {
  await mkdir(path.join(consumer, 'patches'))
  await cp(`patches/${patch}`, path.join(consumer, 'patches', patch))
  const workspace = path.join(consumer, 'pnpm-workspace.yaml')
  await writeFile(workspace, `${await readFile(workspace, 'utf8')}
verifyDepsBeforeRun: error
patchedDependencies:
  '@ztd-me/frontend-checks@0.1.1': patches/${patch}
packageExtensions:
  '@ztd-me/frontend-checks@0.1.1':
    dependencies:
      typescript: 6.0.3
`)
  const manifest = path.join(consumer, 'package.json')
  const info = JSON.parse(await readFile(manifest, 'utf8'))
  info.devDependencies.tailwindcss = '4.3.3'
  info.scripts = { 'check:css': 'ztd-css css-check.config.mjs' }
  await writeFile(manifest, JSON.stringify(info))
}

async function checkCSSCandidate(consumer) {
  const installed = path.join(consumer, 'node_modules/@ztd-me/frontend-checks')
  for (const [name, expected] of Object.entries(receipt.files)) {
    const actual = createHash('sha256').update(await readFile(path.join(installed, name))).digest('hex')
    assert.equal(actual, expected, name)
  }
  await mkdir(path.join(consumer, 'test'))
  const tests = [
    'css.test.mjs',
    'css-classes.test.mjs',
    'css-tailwind.test.mjs',
  ]
  for (const name of tests) {
    const code = await readFile(`test/${name}`, 'utf8')
    const imported = code.replaceAll('\'../src/', '\'../node_modules/@ztd-me/frontend-checks/src/')
    await writeFile(path.join(consumer, 'test', name), imported)
  }
  execFileSync('node', [
    '--test',
    ...tests.map(name => `test/${name}`),
  ], { cwd: consumer, stdio: 'inherit' })
  await writeFile(path.join(consumer, 'candidate.css'), `@import "./app.css";

@theme inline {
  --color-foreground: var(--ink);
  --text-body: 1rem;
  --text-body--line-height: 1.75;
}

@utility candidate-utility { color: var(--ink); }

:root { --ink: #181818; }
`)
  await writeFile(path.join(consumer, 'candidate.tsx'), 'export const utility = "text-foreground h-8"\n')
  const config = 'export default { files: ["candidate.css"], classFiles: ["candidate.tsx"] }\n'
  await writeFile(path.join(consumer, 'css-check.config.mjs'), config)
  execFileSync('pnpm', ['run', 'check:css'], { cwd: consumer, stdio: 'inherit' })
}

const patchedConsumer = await consumerSmoke(
  '0.1.1',
  'Public package with reviewed CSS-first patch',
  { beforeInstall },
)
await checkCSSCandidate(patchedConsumer)
await mkdir('.artifacts', { recursive: true })
await writeFile('.artifacts/css-patch-verification.json', JSON.stringify({
  consumer: patchedConsumer,
  patchSha256: receipt.patchSha256,
  filesVerified: Object.keys(receipt.files),
  cssTests: 17,
  nativeCssCommand: 'pnpm run check:css',
  publishedVersion: '0.1.1',
  patched: true,
  publicPromotion: false,
}, null, 2))
process.stdout.write(`${JSON.stringify({ consumer: patchedConsumer, patchSha256: receipt.patchSha256 })}\n`)
