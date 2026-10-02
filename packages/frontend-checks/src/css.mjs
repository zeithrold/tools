import { readFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import glob from 'fast-glob'
import postcss from 'postcss'
import stylelint from 'stylelint'
import { cssConfig } from './css-config.mjs'
import { tokenWarnings } from './css-tokens.mjs'

async function sources(patterns, cwd) {
  const matches = await Promise.all(patterns.map(async (pattern) => {
    const selected = await glob(pattern, { cwd, absolute: true, onlyFiles: true, followSymbolicLinks: false })
    if (!selected.length) {
      throw new Error(`CSS pattern matched no files: ${pattern}`)
    }
    return selected
  }))
  const files = [
    ...new Set(matches.flat()),
  ]
  return Promise.all(files.sort().map(async (file) => {
    const code = await readFile(file, 'utf8')
    return postcss.parse(code, { from: file })
  }))
}

function validate(options) {
  if (!options || !Array.isArray(options.files) || !options.files.length) {
    throw new TypeError('CSS files must be a nonempty explicit array of paths/globs')
  }
}

export async function checkCss(options) {
  validate(options)
  const cwd = path.resolve(options.cwd ?? process.cwd())
  const roots = await sources(options.files, cwd)
  const definitions = await sources(options.tokenFiles ?? [], cwd)
  const lint = await stylelint.lint({ files: roots.map(root => root.source.input.file), config: cssConfig, cwd })
  const warnings = lint.results.flatMap(result => result.warnings.map(warning => ({
    file: result.source,
    line: warning.line,
    column: warning.column,
    rule: warning.rule,
    text: warning.text,
  })))
  warnings.push(...tokenWarnings(roots, definitions, options))
  return {
    schemaVersion: 1,
    status: lint.errored || warnings.length ? 'failed' : 'passed',
    files: roots.map(root => root.source.input.file),
    warnings,
  }
}
