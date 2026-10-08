import { execFileSync } from 'node:child_process'
import { cp, mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import process from 'node:process'

export const packageRoot = process.cwd()

const runtimeSmoke = `import assert from 'node:assert/strict'
import config, { createConfig } from '@ztd-me/eslint'
import { ESLint } from 'eslint'
assert.equal(config, createConfig)
const js = new ESLint({overrideConfigFile:true,overrideConfig:await config({typescript:false,gitignore:false})})
const [bad] = await js.lintText('export const ids = [1, 2, 3]\\n',{filePath:'sample.js'})
assert.ok(bad.messages.some(message => message.ruleId === 'ztd/array-layout'))
const [usedVariable] = await js.lintText('const number = 1\\nconsole.warn(number)\\n',{filePath:'used.mjs'})
assert.equal(usedVariable.errorCount,0,JSON.stringify(usedVariable.messages))
const [usedImport] = await js.lintText(
  'import { basename } from \\'node:path\\'\\n\\nexport const name = basename(\\'example\\')\\n',
  {filePath:'used.mjs'},
)
assert.equal(usedImport.errorCount,0,JSON.stringify(usedImport.messages))
const [unusedVariable] = await js.lintText('const unused = 1\\n',{filePath:'unused.mjs'})
assert.ok(unusedVariable.messages.some(message=>message.ruleId==='no-unused-vars'))
const [unusedImport] = await js.lintText('import { basename as _unused } from \\'node:path\\'\\n',{
  filePath:'unused.mjs',
})
assert.ok(unusedImport.messages.some(message=>message.ruleId==='unused-imports/no-unused-imports'))
const typed = new ESLint({overrideConfigFile:true,overrideConfig:await config({react:true,vue:true,gitignore:false})})
const [safe] = await typed.lintFiles('sample.ts')
assert.equal(safe.errorCount,0,JSON.stringify(safe.messages))
const effective = await typed.calculateConfigForFile('sample.ts')
assert.deepEqual(effective.rules['ts/consistent-type-definitions'],[2,'type'])
const [interfaceResult] = await typed.lintText('export interface Entry { title: string }\\n',{filePath:'sample.ts'})
assert.ok(interfaceResult.messages.some(message=>message.ruleId==='ts/consistent-type-definitions'))
const [aliasResult] = await typed.lintText('export type Entry = { title: string }\\n',{filePath:'sample.ts'})
assert.equal(aliasResult.errorCount,0,JSON.stringify(aliasResult.messages))
const react = await typed.calculateConfigForFile('sample.tsx')
const vue = await typed.calculateConfigForFile('Sample.vue')
assert.equal(react.rules['react/rules-of-hooks'][0],2)
assert.equal(vue.rules['ts/no-floating-promises'][0],2)
assert.equal(vue.rules['vue-a11y/alt-text'][0],2)
assert.equal(Object.keys(react.rules).some(rule=>rule.startsWith('jsx-a11y/')),false)
const routes = new ESLint({overrideConfigFile:true,overrideConfig:await config({
  typescript:false,react:{framework:'vinext'},gitignore:false,
})})
const source = 'export const metadata = { title: \\'Example\\' }\\n'
  + 'export default function Page() {\\n  return <main>Example</main>\\n}\\n'
const [route] = await routes.lintText(source,{filePath:'app/layout.jsx'})
assert.equal(route.errorCount,0,JSON.stringify(route.messages))
const [ordinary] = await routes.lintText(source,{filePath:'components/Card.jsx'})
assert.ok(ordinary.messages.some(message=>message.ruleId==='react-refresh/only-export-components'))
const [client] = await routes.lintText('\\'use client\\'\\n'+source,{filePath:'app/page.jsx'})
assert.ok(client.messages.some(message=>message.ruleId==='ztd/app-router-exports'))
console.log('ESM and framework smoke passed')
`

export async function consumerSmoke(spec, label) {
  const directory = await mkdtemp(join(tmpdir(), 'ztd-eslint-consumer-'))
  const packageInfo = JSON.parse(await readFile(join(packageRoot, 'package.json'), 'utf8'))
  const env = {
    ...process.env,
    pnpm_config_store_dir: join(tmpdir(), 'ztd-eslint-smoke-store'),
    pnpm_config_cache_dir: join(tmpdir(), 'ztd-eslint-smoke-cache'),
  }
  await writeFile(join(directory, 'package.json'), JSON.stringify({
    private: true,
    type: 'module',
    packageManager: 'pnpm@11.22.0',
    dependencies: {
      '@ztd-me/eslint': spec,
      'eslint': '10.11.0',
      'typescript': '6.0.3',
      'react': packageInfo.devDependencies.react,
      '@types/react': packageInfo.devDependencies['@types/react'],
    },
  }))
  await writeFile(join(directory, 'pnpm-workspace.yaml'), `packages:
  - .
minimumReleaseAge: 1440
minimumReleaseAgeStrict: true
minimumReleaseAgeExclude:
  - '@ztd-me/*'
minimumReleaseAgeExcludePrune: true
shellEmulator: true
trustPolicy: no-downgrade
trustPolicyExclude:
  - semver@6.3.1
`)
  execFileSync('pnpm', ['install'], { cwd: directory, stdio: 'inherit', env })
  await writeConsumerFiles(directory)
  await checkMarkdownConsumer(directory, env)
  execFileSync('node', ['smoke.mjs'], { cwd: directory, stdio: 'inherit' })
  execFileSync('pnpm', [
    'exec',
    'tsc',
    '-p',
    'tsconfig.json',
  ], { cwd: directory, stdio: 'inherit', env })
  console.log(`${label}: installation, ESM exports, types, JS/TS/React/Vue configs and lint all passed: ${directory}`)
  return directory
}

async function checkMarkdownConsumer(directory, env) {
  const consumer = join(directory, 'markdown-consumer')
  await mkdir(join(consumer, 'src'), { recursive: true })
  for (const file of [
    'README.md',
    'eslint.config.js',
    'tsconfig.json',
    'src/Card.tsx',
  ]) {
    await cp(join(packageRoot, 'test/fixtures/markdown-consumer', file), join(consumer, file))
  }
  const options = { cwd: consumer, stdio: 'inherit', env: { ...env, CI: 'true', TSESTREE_SINGLE_RUN: 'true' } }
  execFileSync('pnpm', [
    'exec',
    'eslint',
    '.',
    '--max-warnings',
    '0',
  ], options)
  execFileSync('pnpm', [
    'exec',
    'tsc',
    '--noEmit',
  ], options)
  console.log('Fresh typed app and README TSX/TS Markdown consumer passed')
}

async function writeConsumerFiles(directory) {
  await writeFile(join(directory, 'tsconfig.json'), JSON.stringify({
    compilerOptions: {
      strict: true,
      noUncheckedIndexedAccess: true,
      exactOptionalPropertyTypes: true,
      target: 'ES2023',
      module: 'NodeNext',
      moduleResolution: 'NodeNext',
      skipLibCheck: true,
      noEmit: true,
    },
    include: ['*.ts'],
  }))
  await writeFile(join(directory, 'consumer.ts'), [
    'import config, { createConfig, type ConfigOptions } from \'@ztd-me/eslint\'',
    'import type { Linter } from \'eslint\'',
    'const options: ConfigOptions = {',
    '  typescript: false, react: { compiler: true, framework: \'vinext\' }, vue: true,',
    '}',
    'export const defaults: Promise<Linter.Config[]> = config(options)',
    'export const named: Promise<Linter.Config[]> = createConfig(options)',
  ].join('\n'))
  await writeFile(join(directory, 'sample.ts'), `${[
    'export function describe(value: unknown): string {',
    '  return typeof value === \'string\' ? value : \'unknown\'',
    '}',
  ].join('\n')}\n`)
  await writeFile(join(directory, 'smoke.mjs'), runtimeSmoke)
}
