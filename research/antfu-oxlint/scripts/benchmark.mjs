import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdir, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { performance } from 'node:perf_hooks'
import process from 'node:process'
import antfu from '@antfu/eslint-config'
import { baseOptions, eslintFor, generated, json, oxlint, results, root } from './common.mjs'

const dir = path.join(generated, 'benchmark')
const corpus = path.join(dir, 'corpus')
await mkdir(corpus, { recursive: true })
const native = ['no-debugger', 'no-console', 'eqeqeq', 'prefer-const', 'ts/ban-ts-comment', 'ts/consistent-type-definitions']
const mixed = [...native, 'style/semi', 'style/quotes', 'ts/consistent-type-imports']
const moved = ['no-debugger', 'style/semi', 'style/quotes']
const files = []
const source = `import { Imported } from './types';\nexport type Shape = { value: Imported };\nexport function check(value: number) {\n  debugger;\n  let total = 1;\n  console.log("hello");\n  if (value == 1)\n    return total;\n  return value;\n}\n// @ts-expect-error\nexport const wrong: string = 1;\n${Array.from({ length: 60 }, (_, i) => `export const filler${i} = ${i}\n`).join('')}`
const fileCount = Number(process.env.BENCH_FILES ?? 200)
const repetitions = Number(process.env.BENCH_REPETITIONS ?? 5)
for (let i = 0; i < fileCount; i++) {
  const file = path.join(corpus, `file${String(i).padStart(4, '0')}.ts`)
  await writeFile(file, source)
  files.push(file)
}
const manifest = { files, native, mixed, moved, options: baseOptions }
const manifestPath = path.join(dir, 'manifest.json')
const configs = await antfu(structuredClone(baseOptions))
const effective = await eslintFor(configs).calculateConfigForFile(files[0])
manifest.entries = Object.fromEntries(mixed.map(id => [id, effective.rules[id]]))
manifest.globals = effective.languageOptions.globals
await json(manifestPath, manifest)
const canonical = id => id.startsWith('ts/') ? `typescript/${id.slice(3)}` : id
const makeConfig = ids => ({ plugins: ['typescript'], categories: { correctness: 'off' }, globals: effective.languageOptions.globals, rules: Object.fromEntries(ids.map(id => [canonical(id), effective.rules[id]])) })
const nativePath = path.join(dir, 'native.json')
await json(nativePath, makeConfig(native))
const mixedConfig = makeConfig(native)
mixedConfig.jsPlugins = [{ name: 'style', specifier: '@stylistic/eslint-plugin' }, { name: 'syntax-adapter', specifier: path.join(root, 'prototype/syntax-type-imports.mjs') }]
mixedConfig.rules['style/semi'] = effective.rules['style/semi']
mixedConfig.rules['style/quotes'] = effective.rules['style/quotes']
mixedConfig.rules['syntax-adapter/consistent-type-imports'] = effective.rules['ts/consistent-type-imports']
const mixedPath = path.join(dir, 'mixed.json')
await json(mixedPath, mixedConfig)
const hybridConfig = { plugins: [], categories: { correctness: 'off' }, jsPlugins: [{ name: 'style', specifier: '@stylistic/eslint-plugin' }], rules: Object.fromEntries(moved.map(id => [id, effective.rules[id]])) }
const hybridPath = path.join(dir, 'hybrid.json')
await json(hybridPath, hybridConfig)

function timed(command, args) {
  const start = performance.now()
  const run = spawnSync(command, args, { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, timeout: 120_000 })
  assert.ok(!run.error, run.error?.message)
  const ms = performance.now() - start
  assert.ok([0, 1].includes(run.status), run.stderr)
  return { ms, output: JSON.parse(run.stdout) }
}
const worker = mode => timed(process.execPath, [path.join(root, 'scripts/benchmark-worker.mjs'), manifestPath, mode])
function ox(configPath) {
  const run = timed(oxlint, ['-c', configPath, '--threads', '1', '--format', 'json', ...files])
  assert.ok(run.output.diagnostics.every(m => m.code && !m.message.startsWith('Error running JS plugin')), 'Runtime/config error in benchmark')
  const entries = run.output.diagnostics.map((m) => {
    const match = /^([^()]+)\(([^()]*)\)$/.exec(m.code)
    assert.ok(match, m.code)
    const namespace = match[1]
    const rule = match[2]
    const id = namespace === 'eslint' ? rule : namespace === 'typescript' ? `ts/${rule}` : namespace === 'syntax-adapter' ? `ts/${rule}` : `${namespace}/${rule}`
    return [path.basename(m.filename), id, m.severity === 'warning' ? 1 : 2]
  })
  const rules = entries.reduce((acc, [, id]) => {
    acc[id] = (acc[id] ?? 0) + 1
    return acc
  }, {})
  return { ms: run.ms, output: { signature: createHash('sha256').update(JSON.stringify(entries.sort())).digest('hex'), diagnostics: entries.length, files: run.output.number_of_files, rules } }
}
const rows = []
for (const mode of ['native', 'native-lean', 'mixed-lean', 'full-hybrid']) {
  console.log(`Benchmark ${mode}: ${fileCount} files; ${repetitions} warm process runs`)
  const baselineMode = mode === 'full-hybrid' ? 'full' : mode
  const target = mode.startsWith('native') ? nativePath : mode.startsWith('mixed') ? mixedPath : hybridPath
  const baseline = worker(baselineMode)
  const first = ox(target)
  const residual = mode === 'full-hybrid' ? worker('residual') : null
  if (residual) {
    assert.equal(baseline.output.diagnostics, first.output.diagnostics + residual.output.diagnostics)
    const expected = { ...first.output.rules }
    for (const [id, count] of Object.entries(residual.output.rules))
      expected[id] = (expected[id] ?? 0) + count
    assert.deepEqual(expected, baseline.output.rules)
  }
  else {
    assert.equal(first.output.files, fileCount)
    assert.equal(first.output.signature, baseline.output.signature, `${mode}: workload behavior mismatch`)
  }
  const baselineTimes = []
  const targetTimes = []
  for (let i = 0; i < repetitions; i++) {
    // Alternate ordering to reduce systematic thermal/cache order bias.
    const runBaseline = () => {
      const r = worker(baselineMode)
      assert.equal(r.output.signature, baseline.output.signature)
      baselineTimes.push(r.ms)
    }
    const runTarget = () => {
      const r = ox(target)
      assert.equal(r.output.signature, first.output.signature)
      targetTimes.push(r.ms + (residual ? worker('residual').ms : 0))
    }
    if (i % 2) {
      runTarget()
      runBaseline()
    }
    else {
      runBaseline()
      runTarget()
    }
  }
  const median = values => [...values].sort((a, b) => a - b)[Math.floor(values.length / 2)]
  rows.push({ mode, configuredRuleCount: mode.startsWith('native') ? native.length : mode.startsWith('mixed') ? mixed.length : Object.values(effective.rules).filter(v => v[0] > 0).length, baselineDiagnostics: baseline.output, targetDiagnostics: first.output, residualDiagnostics: residual?.output, firstProcessMs: { eslint: baseline.ms, target: first.ms + (residual?.ms ?? 0) }, warmProcessMs: { eslint: baselineTimes, target: targetTimes }, medianMs: { eslint: median(baselineTimes), target: median(targetTimes) }, speedup: median(baselineTimes) / median(targetTimes) })
}
await json(path.join(results, 'benchmark.json'), { environment: { platform: process.platform, arch: process.arch, node: process.version, cpu: os.cpus()[0].model, logicalCpus: os.cpus().length, memoryBytes: os.totalmem() }, corpus: { fileCount, source, linesPerFile: source.split('\n').length - 1, repetitions, oxlintThreads: 1, eslintConcurrency: 'off' }, protocol: 'First new process after generating corpus (not OS-cache cold); warm = repeated new processes with warm filesystem/module byte cache, alternating order. No lint-result caches. Full hybrid is sequential sum of both complete passes. No persistent Oxlint process API or privileged cache eviction used. Type-aware workload not benchmarked.', rows })
console.log(rows.map(r => ({ mode: r.mode, medianMs: r.medianMs, speedup: r.speedup })))
