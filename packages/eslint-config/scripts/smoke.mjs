import { execFileSync } from 'node:child_process'
import { mkdtemp, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import process from 'node:process'

const runtimeSmoke = `import assert from 'node:assert/strict'
import config, { createConfig } from '@ztd-me/eslint'
import { ESLint } from 'eslint'
assert.equal(config, createConfig)
const js = new ESLint({overrideConfigFile:true,overrideConfig:await config({typescript:false,gitignore:false})})
const [bad] = await js.lintText('export const ids = [1, 2, 3]\\n',{filePath:'sample.js'})
assert.ok(bad.messages.some(message => message.ruleId === 'ztd/array-layout'))
const typed = new ESLint({overrideConfigFile:true,overrideConfig:await config({react:true,vue:true,gitignore:false})})
const [safe] = await typed.lintFiles('sample.ts')
assert.equal(safe.errorCount,0,JSON.stringify(safe.messages))
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
  const env = {
    ...process.env,
    pnpm_config_store_dir: join(tmpdir(), 'ztd-eslint-smoke-store'),
    pnpm_config_cache_dir: join(tmpdir(), 'ztd-eslint-smoke-cache'),
  }
  await writeFile(join(directory, 'package.json'), JSON.stringify({
    private: true,
    type: 'module',
    dependencies: { '@ztd-me/eslint': spec, 'eslint': '10.11.0', 'typescript': '6.0.3' },
  }))
  execFileSync('pnpm', ['install'], { cwd: directory, stdio: 'inherit', env })
  await writeConsumerFiles(directory)
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

export const packageRoot = process.cwd()
