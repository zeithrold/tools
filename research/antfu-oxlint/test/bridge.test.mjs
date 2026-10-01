import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { before, test } from 'node:test'
import { pathToFileURL } from 'node:url'
import { materialize, residualConfig } from '../prototype/materialize.mjs'
import { syntaxTypeImportsRule } from '../prototype/syntax-type-imports.mjs'
import { eslintFor, generated, json, root, runOx } from '../scripts/common.mjs'

before(() => mkdir(generated, { recursive: true }))

async function project(code, sources) {
  const cwd = await mkdtemp(path.join(generated, 'test-'))
  const configModule = path.join(cwd, 'eslint.config.mjs')
  await writeFile(configModule, code)
  for (const [file, source] of Object.entries(sources))
    await writeFile(path.join(cwd, file), source)
  const config = await (await import(pathToFileURL(configModule).href)).default
  return { cwd, configModule, config, files: Object.keys(sources), outputDirectory: path.join(cwd, 'bridge') }
}

test('ESLint oracle retains AND patterns, local ignores, severity-only options and global ignores', async () => {
  const p = await project(`export default [
    { ignores: ['ignored.js'] },
    { files: ['**/*.js'], rules: { 'no-console': [2, {allow:['warn']}] } },
    { files: [['**/*.js', '*test.js']], rules: { 'no-console': 'off' } },
    { files: ['**/*.js'], ignores: ['skip.js', '*test.js'], rules: { 'no-console': 'warn' } }
  ];`, { 'keep.js': 'console.warn(1); console.log(2);', 'skip.js': 'console.log(1);', 'a.test.js': 'console.log(1);', 'ignored.js': 'console.log(1);' })
  const routes = { 'no-console': { kind: 'native', id: 'no-console', accept: () => true } }
  const plan = await materialize({ ...p, routes })
  assert.deepEqual(plan.ignored, ['ignored.js'])
  const keep = await readFile(plan.groups[0].configPath, 'utf8')
  assert.deepEqual(JSON.parse(keep).rules['no-console'], [1, { allow: ['warn'] }])
  const skip = JSON.parse(await readFile(plan.groups[1].configPath, 'utf8'))
  assert.equal(skip.rules['no-console'][0], 2)
  assert.deepEqual(JSON.parse(await readFile(plan.groups[2].configPath, 'utf8')).rules, {})
  const original = await eslintFor(p.config, p.cwd).lintFiles(p.files)
  const residual = await eslintFor(residualConfig(p.config, plan), p.cwd).lintFiles(p.files)
  const movedCount = plan.groups.reduce((n, g) => n + (runOx(g.configPath, [path.join(p.cwd, g.file)], [], p.cwd).output?.diagnostics.length ?? 0), 0)
  assert.equal(original.reduce((n, f) => n + f.messages.length, 0), residual.reduce((n, f) => n + f.messages.length, 0) + movedCount)
})

test('unverified rules and unsupported file/parser/directive semantics stay on ESLint', async () => {
  const p = await project(`export default [{files:['**/*.{js,ts}'],rules:{'no-debugger':'error','no-console':'error'}}];`, { 'normal.js': 'debugger; console.log(1);', 'directive.js': '/* eslint-disable no-debugger */\ndebugger;', 'document.md': '# heading' })
  const plan = await materialize({ ...p, routes: { 'no-debugger': { kind: 'native', id: 'no-debugger', accept: () => true } } })
  assert.equal(plan.groups.length, 1)
  assert.deepEqual(plan.fallback[0].rules, ['no-console'])
  assert.equal(plan.fallback[1].wholeFile, true)
  assert.deepEqual(plan.ignored, ['document.md'])
  await assert.rejects(materialize({ ...p, files: ['../escape.js'], routes: {} }), /escapes cwd/)
})

test('exact per-file plugin objects and settings survive; scoped plugins are not guessed', async () => {
  const p = await project(`
  function plugin(label) { return { rules: { check: { meta: {schema:[]}, create(c) { return { Program(node) { c.report({node,message:label+':'+c.settings.label}); } }; } } } }; }
  export default [
    {files:['a.js'],plugins:{'@local/pkg':plugin('A')},settings:{label:'one'},rules:{'@local/pkg/check':'error'}},
    {files:['b.js'],plugins:{'@local/pkg':plugin('B')},settings:{label:'two'},rules:{'@local/pkg/check':'warn'}}
  ];`, { 'a.js': 'export const x = 1;', 'b.js': 'export const y = 1;' })
  const plan = await materialize({ ...p, routes: { '@local/pkg/check': { kind: 'js', accept: () => true } } })
  const first = runOx(plan.groups[0].configPath, [path.join(p.cwd, 'a.js')], [], p.cwd)
  const second = runOx(plan.groups[1].configPath, [path.join(p.cwd, 'b.js')], [], p.cwd)
  assert.equal(first.output.diagnostics[0].message, 'A:one')
  assert.equal(second.output.diagnostics[0].message, 'B:two')
  assert.equal(second.output.diagnostics[0].severity, 'warning')
})

test('syntax adapter rejects decorator metadata and retains antfu fixStyle', async () => {
  assert.throws(() => syntaxTypeImportsRule({ emitDecoratorMetadata: true }), /use ESLint/)
  assert.throws(() => syntaxTypeImportsRule({ experimentalDecorators: true }), /use ESLint/)
  const cwd = await mkdtemp(path.join(generated, 'adapter-test-'))
  const source = 'import { Thing, make } from \'./thing\';\nexport const value: Thing = make();\n'
  const file = path.join(cwd, 'case.ts')
  await writeFile(file, source)
  const configPath = path.join(cwd, 'ox.json')
  await json(configPath, { plugins: [], categories: { correctness: 'off' }, jsPlugins: [{ name: 'syntax-adapter', specifier: path.join(root, 'prototype/syntax-type-imports.mjs') }], rules: { 'syntax-adapter/consistent-type-imports': ['error', { prefer: 'type-imports', fixStyle: 'separate-type-imports', disallowTypeAnnotations: false }] } })
  const run = runOx(configPath, [file], ['--fix'])
  assert.equal(run.status, 0)
  assert.equal(await readFile(file, 'utf8'), 'import type { Thing} from \'./thing\';\nimport { make } from \'./thing\';\nexport const value: Thing = make();\n')
})

test('nested rule names retain the exact plugin namespace; type services and lossy settings fall back', async () => {
  const p = await project(`
    const rule = {meta:{schema:[]},create(c){return {Program(node){c.report({node,message:'nested-rule'})}}}};
    export default [
      {files:['nested.js'],plugins:{node:{rules:{'prefer-global/process':rule}}},rules:{'node/prefer-global/process':'error'}},
      {files:['typed.js'],plugins:{local:{rules:{check:{...rule,meta:{schema:[],docs:{requiresTypeChecking:true}}}}}},rules:{'local/check':'error'}},
      {files:['settings.js'],settings:{label:Symbol('not-json')},rules:{'no-debugger':'error'}}
    ];`, { 'nested.js': 'export const x = 1;', 'typed.js': 'export const y = 1;', 'settings.js': 'debugger;' })
  const plan = await materialize({ ...p, routes: {
    'node/prefer-global/process': { kind: 'js', accept: () => true },
    'local/check': { kind: 'js', accept: () => true },
    'no-debugger': { kind: 'native', id: 'no-debugger', accept: () => true },
  } })
  const run = runOx(plan.groups[0].configPath, [path.join(p.cwd, 'nested.js')], [], p.cwd)
  assert.equal(run.output.diagnostics[0].message, 'nested-rule')
  assert.equal(run.output.diagnostics[0].code, 'bridge-node(prefer-global/process)')
  assert.deepEqual(plan.fallback[1].rules, ['local/check'])
  assert.equal(plan.fallback[2].wholeFile, true)
})
