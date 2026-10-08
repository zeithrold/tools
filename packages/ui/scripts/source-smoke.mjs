import assert from 'node:assert/strict'
import { spawn, spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { cp, mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import path from 'node:path'
import process from 'node:process'
import { browserArtifactRoot } from './browser-artifacts.mjs'

const cli = 'shadcn@4.21.1'
const repository = path.resolve('../..')
const sourceSha = process.argv[2]
if (sourceSha !== undefined) {
  assert.equal(process.env.ZTD_LOCAL_FONT_PREVIEW, undefined, 'Public verification requires actual Google Fonts')
  assert.match(sourceSha, /^[a-f0-9]{40}$/u, 'Use the full approved source commit SHA')
  const head = spawnSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' })
  assert.equal(head.status, 0)
  assert.equal(head.stdout.trim(), sourceSha, 'Check out the pinned source commit first')
}
const registryUrl = sourceSha === undefined
  ? 'http://127.0.0.1:4329/{name}.json'
  : `https://raw.githubusercontent.com/zeithrold/tools/${sourceSha}/registry/{name}.json`
await mkdir('.artifacts', { recursive: true })
const consumer = await mkdtemp(path.resolve('.artifacts/source-consumer-'))
const destination = 'components/ui/ztd-me'
const payloadPath = path.resolve('.artifacts/registry/ui.json')

async function run(command, args, cwd = consumer, env = process.env) {
  await new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd, env, stdio: 'inherit' })
    child.once('error', reject)
    child.once('exit', (code) => {
      if (code === 0) {
        resolve()
      }
      else {
        reject(new Error(`${command} ${args.join(' ')} exited ${code}`))
      }
    })
  })
}

async function prepareProject() {
  await cp('test/consumer', consumer, { recursive: true })
  const packageInfo = JSON.parse(await readFile('package.json', 'utf8'))
  await writeFile(path.join(consumer, 'package.json'), JSON.stringify({
    private: true,
    type: 'module',
    packageManager: 'pnpm@11.22.0',
    devDependencies: packageInfo.devDependencies,
  }))
  const workspace = await readFile('pnpm-workspace.yaml', 'utf8')
  await writeFile(path.join(consumer, 'pnpm-workspace.yaml'), workspace.split('patchedDependencies:')[0])
  const config = JSON.parse(await readFile('tsconfig.build.json', 'utf8'))
  delete config.compilerOptions.rootDir
  config.compilerOptions.types.push('vite/client')
  config.compilerOptions.paths = { '@/*': ['./*'] }
  config.include = [
    '*.tsx',
    `${destination}/**/*.ts`,
    `${destination}/**/*.tsx`,
  ]
  await writeFile(path.join(consumer, 'tsconfig.json'), JSON.stringify(config))
  await writeFile(path.join(consumer, 'components.json'), JSON.stringify({
    $schema: 'https://ui.shadcn.com/schema.json',
    style: 'new-york',
    rsc: false,
    tsx: true,
    tailwind: { config: '', css: 'fixture.css', baseColor: 'neutral', cssVariables: true },
    aliases: {
      components: '@/components',
      ui: '@/components/ui',
      utils: '@/lib/utils',
      lib: '@/lib',
      hooks: '@/hooks',
    },
    registries: { '@ztd-me': registryUrl },
  }))
  for (const file of [
    'Fixture.tsx',
    'PrimitiveFixture.tsx',
    'api.tsx',
    'client.tsx',
    'server.tsx',
  ]) {
    const filename = path.join(consumer, file)
    const source = await readFile(filename, 'utf8')
    const local = source.replaceAll('@ztd-me/ui/client', `./${destination}/client.js`)
      .replaceAll('@ztd-me/ui/styles.css', `./${destination}/styles.css`)
      .replaceAll('@ztd-me/ui', `./${destination}/index.js`)
    await writeFile(filename, local)
  }
}

async function inventorySource(item) {
  const packageInfo = JSON.parse(await readFile(path.join(consumer, 'package.json'), 'utf8'))
  for (const name of ['@ztd-me/ui', '@ztd-me/frontend']) {
    assert.equal(name in (packageInfo.dependencies ?? {}), false)
    assert.equal(name in packageInfo.devDependencies, false)
  }
  return Promise.all(item.files.map(async (file) => {
    const installed = file.target.replace('@ui/', 'components/ui/')
    const bytes = await readFile(path.join(consumer, installed))
    assert.equal(bytes.toString(), file.content, `Installed source changed: ${installed}`)
    return { installed, sha256: createHash('sha256').update(bytes).digest('hex') }
  }))
}

async function configureVerification() {
  const eslint = (await readFile('eslint.config.js', 'utf8')).replace('tsconfig.build.json', 'tsconfig.json')
  await writeFile(path.join(consumer, 'eslint.config.js'), eslint)
  const css = (await readFile('css-check.config.mjs', 'utf8')).replaceAll('src/', `${destination}/`)
  await writeFile(path.join(consumer, 'css-check.config.mjs'), css)
  const workspacePath = path.join(consumer, 'pnpm-workspace.yaml')
  const workspace = await readFile(workspacePath, 'utf8')
  await writeFile(workspacePath, `${workspace}patchedDependencies:\n`
  + `  '@radix-ui/react-select@2.3.7': ${destination}/patches/@radix-ui__react-select@2.3.7.patch\n`
  + `  'vaul@1.1.2': ${destination}/patches/vaul@1.1.2.patch\n`)
  await writeFile(path.join(consumer, 'vite.config.ts'), 'import { defineConfig } from \'vite\';\n'
  + 'import tailwind from \'@tailwindcss/vite\';\n'
  + 'export default defineConfig({ plugins: [tailwind()] });\n')
  const fixture = path.join(consumer, 'fixture.css')
  const cssEntry = `@import "tailwindcss";\n@import "./${destination}/tailwind.css";\n`
    + `@source "./${destination}";\n`
  await writeFile(fixture, cssEntry + await readFile(fixture, 'utf8'))
  // This disposable lock must first record the newly installed patch; verifySource then freezes it.
  await run('pnpm', ['install', '--no-frozen-lockfile'])
}

async function checkUnits() {
  await mkdir(path.join(consumer, 'test'), { recursive: true })
  for (const file of [
    'preferences.test.mjs',
    'i18n.test.mjs',
    'footer.test.mjs',
    'radix-types.test.mjs',
  ]) {
    const original = await readFile(`test/${file}`, 'utf8')
    const base = `../dist/type-smoke/${destination}`
    const localClient = original.replaceAll('@ztd-me/ui/client', `${base}/client.js`)
    const localServer = localClient.replaceAll('@ztd-me/ui', `${base}/index.js`)
    const test = localServer.replace('src/radix-probe.mts', `${destination}/radix-probe.mts`)
    await writeFile(path.join(consumer, 'test', file), test)
  }
  await run('node', [
    '--test',
    'test/preferences.test.mjs',
    'test/i18n.test.mjs',
    'test/footer.test.mjs',
    'test/radix-types.test.mjs',
  ])
}

async function verifySource() {
  await run('pnpm', ['install', '--frozen-lockfile'])
  await run('pnpm', [
    'exec',
    'eslint',
    `${destination}/**/*.{ts,tsx}`,
    '--max-warnings',
    '0',
  ])
  if (sourceSha === undefined) {
    await run('node', [
      path.resolve('../frontend-checks/src/css-cli.mjs'),
      'css-check.config.mjs',
    ])
  }
  else {
    await run('pnpm', [
      'exec',
      'ztd-css',
      'css-check.config.mjs',
    ])
  }
  await run('pnpm', [
    'exec',
    'tsc',
    '--noEmit',
  ])
  await run('pnpm', [
    'exec',
    'tsc',
    '--outDir',
    'dist/type-smoke',
    '--declaration',
    'false',
  ])
  await checkUnits()
  await run('pnpm', [
    'exec',
    'vite',
    'build',
    '--outDir',
    'dist/client',
  ])
  await run('pnpm', [
    'exec',
    'vite',
    'build',
    '--ssr',
    'server.tsx',
    '--outDir',
    'dist/server',
  ])
}

await run('pnpm', [
  'dlx',
  cli,
  'build',
  'registry.json',
  '--output',
  'packages/ui/.artifacts/registry',
], repository)
await prepareProject()
await run('pnpm', ['install'])
const payload = JSON.parse(await readFile(payloadPath, 'utf8'))
const committedPayload = JSON.parse(await readFile(path.join(repository, 'registry/ui.json'), 'utf8'))
assert.deepEqual(committedPayload, payload, 'Regenerate registry/ui.json before publishing this revision')
if (sourceSha !== undefined) {
  const response = await fetch(registryUrl.replace('{name}', 'ui'))
  assert.equal(response.status, 200)
  assert.deepEqual(await response.json(), payload, 'Public item differs from this source revision')
}
const server = createServer(async (_request, response) => {
  response.setHeader('Content-Type', 'application/json')
  response.end(await readFile(payloadPath))
})
if (sourceSha === undefined) {
  await new Promise((resolve) => {
    server.listen(4329, '127.0.0.1', resolve)
  })
}
try {
  await run('pnpm', [
    'dlx',
    cli,
    'add',
    '@ztd-me/ui',
    '--dry-run',
  ])
  await run('pnpm', [
    'dlx',
    cli,
    'add',
    '@ztd-me/ui',
    '--yes',
  ])
}
finally {
  if (server.listening) {
    server.close()
  }
}
const files = await inventorySource(payload)
await configureVerification()
await verifySource()
const deliveryMode = sourceSha ? 'public-source' : 'source'
const receipt = {
  consumer,
  cli,
  item: '@ztd-me/ui',
  payloadSha256: createHash('sha256').update(await readFile(payloadPath)).digest('hex'),
  dependencies: payload.dependencies,
  files,
  requiredUiPackage: false,
  sourceSha,
  registryUrl,
  browserArtifacts: browserArtifactRoot(deliveryMode, Boolean(process.env.ZTD_LOCAL_FONT_PREVIEW)),
  fontVerification: process.env.ZTD_LOCAL_FONT_PREVIEW ? 'local-preview-only' : 'google-fonts-api',
  cssVerification: sourceSha === undefined ? 'local-candidate-checker' : 'published-checker',
  publicInstallationVerified: false,
}
await writeFile('.artifacts/source-consumer.json', JSON.stringify(receipt, null, 2))
await run('pnpm', [
  'exec',
  'playwright',
  'test',
], process.cwd(), { ...process.env, ZTD_UI_CONSUMER: consumer, ZTD_UI_VERIFICATION_MODE: deliveryMode })
receipt.publicInstallationVerified = sourceSha !== undefined
await writeFile('.artifacts/source-consumer.json', JSON.stringify(receipt, null, 2))
