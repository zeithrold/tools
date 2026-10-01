import { spawnSync } from 'node:child_process'
import { writeFile } from 'node:fs/promises'
import path from 'node:path'
import antfu from '@antfu/eslint-config'
import migrate from '@oxlint/migrate'
import { eslintFor, json, nativeCandidate, oxlint, profiles, results, root, ruleImplementation, samplePaths, severity } from './common.mjs'

const native = JSON.parse(spawnSync(oxlint, ['--rules', '--format', 'json'], { encoding: 'utf8' }).stdout)
await json(path.join(results, 'native-rules.json'), native)
const inventory = { denominator: 'Unique enabled rule IDs in ESLint effective configs for the 21 recorded sample paths, per profile; not all npm plugin rules, not config declarations or proven equivalent rules.', nativeRuleCount: native.length, profiles: {} }
for (const [name, options] of Object.entries(profiles)) {
  const configs = await antfu(structuredClone(options))
  const eslint = eslintFor(configs)
  const rules = new Map()
  const files = {}
  for (const file of samplePaths) {
    const resolved = await eslint.calculateConfigForFile(path.join(root, file))
    if (!resolved) {
      files[file] = { ignoredOrUnmatched: true }
      continue
    }
    const active = Object.entries(resolved.rules ?? {}).filter(([, entry]) => severity(entry) > 0)
    files[file] = { count: active.length, parser: resolved.languageOptions?.parser?.meta?.name, processor: !!resolved.processor }
    for (const [id, entry] of active) {
      const impl = ruleImplementation(resolved, id)
      const candidate = nativeCandidate(id, impl, native)
      const requiresTypeChecking = !!impl?.meta?.docs?.requiresTypeChecking || id === 'react/no-leaked-conditional-rendering'
      const nonJs = !/\.[cm]?[jt]sx?$/.test(file)
      const category = nonJs || (requiresTypeChecking && !candidate)
        ? 'eslint-fallback'
        : candidate ? 'native-candidate' : impl ? 'js-plugin-candidate' : 'eslint-fallback'
      const record = rules.get(id) ?? { id, docs: impl?.meta?.docs?.url, requiresTypeChecking, native: candidate ? { id: `${candidate.scope}/${candidate.value}`, typeAware: candidate.type_aware, fix: candidate.fix } : undefined, occurrences: [] }
      record.occurrences.push({ file, entry, category })
      rules.set(id, record)
    }
  }
  const counts = { 'native-candidate': 0, 'js-plugin-candidate': 0, 'eslint-fallback': 0 }
  for (const record of rules.values()) {
    record.category = record.occurrences.some(x => x.category === 'native-candidate') ? 'native-candidate' : record.occurrences.some(x => x.category === 'js-plugin-candidate') ? 'js-plugin-candidate' : 'eslint-fallback'
    counts[record.category]++
    const variants = new Map()
    for (const occurrence of record.occurrences) {
      const key = JSON.stringify([occurrence.entry, occurrence.category])
      const variant = variants.get(key) ?? { entry: occurrence.entry, category: occurrence.category, files: [] }
      variant.files.push(occurrence.file)
      variants.set(key, variant)
    }
    record.occurrences = [...variants.values()]
  }
  const warnings = []
  const skipped = {}
  const reporter = {
    addWarning: message => warnings.push(message),
    getWarnings: () => warnings,
    markSkipped: (rule, category) => { (skipped[category] ??= new Set()).add(rule) },
    removeSkipped: (rule, category) => skipped[category]?.delete(rule),
    getSkippedRulesByCategory: () => Object.fromEntries(Object.entries(skipped).map(([k, v]) => [k, [...v]])),
  }
  // Pass the live composer: migrate has antfu-specific renamePlugins handling.
  const migrated = await migrate(antfu(structuredClone(options)), undefined, { typeAware: true, jsPlugins: true, withNursery: true, reporter })
  await json(path.join(results, `migrated-${name}.json`), migrated)
  inventory.profiles[name] = {
    options,
    blocks: configs.map(c => ({ name: c.name, files: c.files, ignores: c.ignores, parser: c.languageOptions?.parser?.meta?.name, parserOptions: c.languageOptions?.parserOptions, language: c.language, processor: !!c.processor, plugins: Object.fromEntries(Object.entries(c.plugins ?? {}).map(([k, p]) => [k, p.meta?.name ?? '(no meta.name)'])) })),
    denominator: rules.size,
    counts,
    files,
    warnings: [...new Set(warnings)],
    skipped: reporter.getSkippedRulesByCategory(),
    rules: [...rules.values()].sort((a, b) => a.id.localeCompare(b.id)),
  }
  console.log(name, rules.size, counts)
}
// Keep each rule and its full occurrence matrix on one line; avoid 185k lines of
// repeated JSON indentation while retaining every measured option/file variant.
const profileText = Object.entries(inventory.profiles).map(([name, { rules, ...metadata }]) => {
  const prefix = JSON.stringify(metadata, null, 2).slice(0, -2)
  const value = `${prefix},\n  "rules": [\n${rules.map(rule => `    ${JSON.stringify(rule)}`).join(',\n')}\n  ]\n}`
  return `    ${JSON.stringify(name)}: ${value.replaceAll('\n', '\n    ')}`
}).join(',\n')
await writeFile(path.join(results, 'inventory.json'), `{\n  "denominator": ${JSON.stringify(inventory.denominator)},\n  "nativeRuleCount": ${inventory.nativeRuleCount},\n  "profiles": {\n${profileText}\n  }\n}\n`)
const csv = [['profile', 'rule', 'candidate_class', 'native_candidate', 'requires_type_checking', 'option_variants', 'file_count']]
for (const [profile, data] of Object.entries(inventory.profiles)) {
  for (const rule of data.rules)
    csv.push([profile, rule.id, rule.category, rule.native?.id ?? '', rule.requiresTypeChecking, rule.occurrences.length, new Set(rule.occurrences.flatMap(o => o.files)).size])
}
await writeFile(path.join(results, 'rule-matrix.csv'), `${csv.map(row => row.map(value => `"${String(value).replaceAll('"', '""')}"`).join(',')).join('\n')}\n`)
