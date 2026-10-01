import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { ESLint } from 'eslint'

const reserved = new Set(['eslint', 'typescript', '@typescript-eslint', 'unicorn', 'import', 'import-x', 'react', 'react-hooks', 'node', 'jsdoc', 'vue', 'jest', 'vitest', 'jsx-a11y', 'nextjs', 'oxc', 'promise', 'react-perf'])

function isJson(value, seen = new Set()) {
  if (value === null || ['string', 'boolean'].includes(typeof value))
    return true
  if (typeof value === 'number')
    return Number.isFinite(value)
  if (!value || typeof value !== 'object' || seen.has(value))
    return false
  if (!Array.isArray(value) && ![Object.prototype, null].includes(Object.getPrototypeOf(value)))
    return false
  const keys = Reflect.ownKeys(value)
  if (keys.some(key => typeof key === 'symbol'))
    return false
  if (Array.isArray(value) && (keys.length !== value.length + 1 || Array.from({ length: value.length }, (_, i) => i).some(i => !Object.hasOwn(value, i))))
    return false
  if (keys.some(key => !(Array.isArray(value) && key === 'length') && (!Object.getOwnPropertyDescriptor(value, key).enumerable || !('value' in Object.getOwnPropertyDescriptor(value, key)))))
    return false
  seen.add(value)
  const valid = Object.values(value).every(v => isJson(v, seen))
  seen.delete(value)
  return valid
}

// Files must be an explicit, finite manifest. ESLint remains the matching/merge oracle.
// Routes are an externally verified allowlist, never inferred from matching rule names.
export async function materialize({ configModule, files, routes, outputDirectory, cwd }) {
  const imported = await import(pathToFileURL(configModule).href)
  const config = await imported.default
  const eslint = new ESLint({ cwd, overrideConfigFile: true, overrideConfig: config })
  const plan = { groups: [], fallback: [], ignored: [], mappings: {} }
  await mkdir(outputDirectory, { recursive: true })
  for (const file of files) {
    const absolute = path.resolve(cwd, file)
    if (!absolute.startsWith(`${path.resolve(cwd)}${path.sep}`))
      throw new Error(`File escapes cwd: ${file}`)
    const resolved = await eslint.calculateConfigForFile(absolute)
    if (!resolved) {
      plan.ignored.push(file)
      continue
    }
    const language = resolved.languageOptions
    const source = await readFile(absolute, 'utf8')
    const impliedType = /\.cjs$/.test(file) ? 'commonjs' : 'module'
    const parserName = language?.parser?.meta?.name ?? language?.parser?.name ?? ''
    if (!/\.[cm]?[jt]sx?$/.test(file) || resolved.processor
      || !['espree', 'typescript-eslint/parser'].includes(parserName)
      || language.sourceType !== impliedType || ![2026, 'latest'].includes(language.ecmaVersion)
      || resolved.linterOptions?.noInlineConfig
      || !isJson(resolved.settings ?? {})
      || /(?:\/\*|\/\/)\s*(?:eslint[\s\-]|oxlint-)/.test(source)) {
      plan.fallback.push({ file, wholeFile: true, reason: 'Unsupported file, processor, parser, sourceType, ecmaVersion or inline directive semantics' })
      continue
    }
    const moved = {}
    const fallback = []
    const plugins = []
    const jsPlugins = []
    for (const [id, entry] of Object.entries(resolved.rules ?? {})) {
      if ([0, 'off'].includes(entry[0] ?? entry))
        continue
      const route = routes[id]
      const pluginNamespace = Object.keys(resolved.plugins ?? {}).filter(key => id.startsWith(`${key}/`)).sort((a, b) => b.length - a.length)[0]
      const implementation = pluginNamespace ? resolved.plugins[pluginNamespace].rules?.[id.slice(pluginNamespace.length + 1)] : null
      if (!route || !isJson(entry) || route.typeAware || implementation?.meta?.docs?.requiresTypeChecking
        || id === 'react/no-leaked-conditional-rendering' || !route.accept(entry, resolved)) {
        fallback.push(id)
        continue
      }
      if (route.kind === 'native') {
        moved[route.id] = entry
        if (route.plugin && !plugins.includes(route.plugin))
          plugins.push(route.plugin)
        plan.mappings[id] = route.id
      }
      else {
        const namespace = pluginNamespace ?? 'core-js'
        const rule = pluginNamespace ? id.slice(pluginNamespace.length + 1) : id
        const alias = reserved.has(namespace) ? `bridge-${namespace}` : namespace
        const wrapper = path.join(outputDirectory, `${encodeURIComponent(namespace)}-${plan.groups.length}.mjs`)
        // Preserve the original plugin object and rule implementation from the config module.
        // Generated wrappers contain references, never serialized functions.
        const code = `import config from ${JSON.stringify(pathToFileURL(configModule).href)};\nimport { ESLint } from 'eslint';\nimport { builtinRules } from 'eslint/use-at-your-own-risk';\nconst eslint = new ESLint({ cwd: ${JSON.stringify(cwd)}, overrideConfigFile: true, overrideConfig: await config });\nconst resolved = await eslint.calculateConfigForFile(${JSON.stringify(absolute)});\nexport default ${!pluginNamespace ? '{ rules: Object.fromEntries(builtinRules) }' : `resolved.plugins[${JSON.stringify(namespace)}]`};\n`
        await writeFile(wrapper, code)
        if (!jsPlugins.some(p => p.name === alias))
          jsPlugins.push({ name: alias, specifier: wrapper })
        moved[`${alias}/${rule}`] = entry
        plan.mappings[id] = `${alias}/${rule}`
      }
    }
    // A group per file also retains per-file settings; Oxlint overrides cannot do this.
    const target = {
      plugins,
      jsPlugins,
      categories: { correctness: 'off' },
      env: { builtin: true },
      globals: Object.fromEntries(Object.entries(resolved.languageOptions.globals ?? {}).map(([key, value]) => [key, ({ true: 'writable', false: 'readonly', readable: 'readonly', writeable: 'writable' })[value] ?? value])),
      settings: resolved.settings,
      rules: moved,
      options: { reportUnusedDisableDirectives: 'off' },
    }
    const configPath = path.join(outputDirectory, `file-${plan.groups.length}.json`)
    await writeFile(configPath, `${JSON.stringify(target, null, 2)}\n`)
    plan.groups.push({ file, configPath, moved: Object.keys(moved).length })
    plan.fallback.push({ file, wholeFile: false, rules: fallback })
  }
  return plan
}

// Use this with the same file manifest. Keep processors and all rules on non-JS files.
// noInlineConfig, inline rule options, renamed disables and unused directive accounting
// require additional auditing before using a partition in production.
export function residualConfig(config, plan) {
  return [...config, ...plan.fallback.filter(f => !f.wholeFile).map(f => ({
    files: [f.file],
    rules: Object.fromEntries(Object.keys(plan.mappings).filter(id => !f.rules.includes(id)).map(id => [id, 'off'])),
  }))]
}
