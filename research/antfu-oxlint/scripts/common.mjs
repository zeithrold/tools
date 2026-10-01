import { spawnSync } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import antfu from '@antfu/eslint-config'
import { ESLint } from 'eslint'
import { builtinRules } from 'eslint/use-at-your-own-risk'

export const root = fileURLToPath(new URL('../', import.meta.url))
export const generated = path.join(root, '.generated')
export const results = path.join(root, 'results')
export const oxlint = path.join(root, 'node_modules/.bin/oxlint')
export const baseOptions = { gitignore: false, isInEditor: false, typescript: true, vue: false }
export const profiles = {
  automatic: { isInEditor: false },
  base: baseOptions,
  editor: { ...baseOptions, isInEditor: true },
  library: { ...baseOptions, type: 'lib' },
  typed: { ...baseOptions, typescript: { tsconfigPath: 'tsconfig.json' } },
  react: { ...baseOptions, react: true, typescript: { tsconfigPath: 'tsconfig.json' } },
  vue: { ...baseOptions, vue: true },
  svelte: { ...baseOptions, svelte: true },
  astro: { ...baseOptions, astro: true },
  solid: { ...baseOptions, solid: true },
  formatters: { ...baseOptions, formatters: { css: true, html: true } },
}
export const samplePaths = [
  'src/app.js',
  'src/app.ts',
  'src/app.tsx',
  'src/app.test.ts',
  'src/app.d.ts',
  'scripts/build.ts',
  'bin/cli.js',
  'eslint.config.js',
  'src/app.cjs',
  'src/App.vue',
  'src/App.svelte',
  'src/App.astro',
  'package.json',
  'tsconfig.json',
  'src/data.json',
  'src/data.jsonc',
  'src/data.yaml',
  'src/data.toml',
  'docs/guide.md',
  'src/style.css',
  'src/page.html',
]
export function severity(entry) {
  const value = Array.isArray(entry) ? entry[0] : entry
  return ({ off: 0, warn: 1, error: 2 })[value] ?? value
}
export function eslintFor(config, cwd = root, extra = {}) {
  return new ESLint({ cwd, overrideConfigFile: true, overrideConfig: config, ...extra })
}
export function runOx(configPath, paths, flags = [], cwd = root) {
  const run = spawnSync(oxlint, ['-c', configPath, '--format', 'json', '--threads', '1', ...flags, ...paths], {
    cwd,
    encoding: 'utf8',
    maxBuffer: 32 * 1024 * 1024,
    timeout: 120_000,
  })
  let output
  try {
    output = JSON.parse(run.stdout)
  }
  catch {
    output = null
  }
  return { status: run.status, error: run.error?.message, stderr: run.stderr, stdout: output ? undefined : run.stdout, output }
}
export async function json(file, value) {
  await mkdir(path.dirname(file), { recursive: true })
  await writeFile(file, `${JSON.stringify(value, null, 2)}\n`)
}
export async function loadJson(file) {
  return JSON.parse(await readFile(file, 'utf8'))
}
export async function resolveProfile(name) {
  return await antfu(structuredClone(profiles[name]))
}
export function ruleImplementation(config, id) {
  const namespace = Object.keys(config.plugins ?? {}).filter(key => id.startsWith(`${key}/`)).sort((a, b) => b.length - a.length)[0]
  return namespace ? config.plugins[namespace].rules?.[id.slice(namespace.length + 1)] : builtinRules.get(id)
}
export function nativeCandidate(id, implementation, nativeRules) {
  const slash = id.indexOf('/')
  const prefix = slash < 0 ? 'eslint' : id.slice(0, slash)
  const name = slash < 0 ? id : id.slice(slash + 1)
  const aliases = { 'ts': 'typescript', 'test': 'vitest', 'node': 'node', 'next': 'nextjs', 'react-refresh': 'react' }
  // antfu react is @eslint-react, import is import-lite: matching names are insufficient.
  if (prefix === 'react' || prefix === 'import')
    return undefined
  if (prefix === 'test' && name === 'no-only-tests')
    return undefined
  const canonical = `${aliases[prefix] ?? prefix}/${name}`
  return nativeRules.find(x => `${x.scope}/${x.value}` === canonical)
}
