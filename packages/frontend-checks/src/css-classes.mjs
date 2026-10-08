import { readFile } from 'node:fs/promises'
import glob from 'fast-glob'
import ts from 'typescript'
import { hasLiteralColor, references } from './css-tokens.mjs'

const colors = new RegExp([
  '^(?:bg|text|border(?:-[xysetbrl])?|outline|ring|fill|stroke|',
  'decoration|shadow|accent|caret|from|via|to)-(.+)$',
].join(''), 'u')
const palette = new RegExp([
  '^(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|',
  'sky|blue|indigo|violet|purple|fuchsia|pink|rose)-(?:50|[1-9]00|950)(?:/\\d+)?$',
].join(''), 'u')

function utility(token) {
  let start = 0
  let depth = 0
  for (let index = 0; index < token.length; index++) {
    if (token[index] === '[' || token[index] === '(') {
      depth++
    }
    if (token[index] === ']' || token[index] === ')') {
      depth--
    }
    if (token[index] === ':' && depth === 0) {
      start = index + 1
    }
  }
  return token.slice(start).replace(/^!|!$/gu, '')
}
function customProperties(roots, external) {
  const known = new Set(external ?? [])
  for (const root of roots) {
    root.walkDecls(declaration => known.add(declaration.prop))
    root.walkAtRules('property', rule => known.add(rule.params.trim()))
  }
  return known
}
function arbitraryValue(value) {
  if (value.startsWith('(')) {
    return `var(${value.slice(1, value.lastIndexOf(')'))})`
  }
  if (value.startsWith('[')) {
    return value.slice(1, value.lastIndexOf(']')).replaceAll('_', ' ')
  }
  return value
}
function literalColor(value) {
  return palette.test(value) || ['white', 'black'].includes(value)
    || (value.startsWith('[') && hasLiteralColor(arbitraryValue(value)))
}
function staticLiteral(node) {
  return ts.isStringLiteralLike(node) || ts.isTemplateHead(node)
    || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)
}
function tokenIssues(token, known) {
  const match = colors.exec(utility(token))
  if (!match) {
    return []
  }
  const value = match[1]
  const issues = []
  if (literalColor(value)) {
    issues.push([
      'ztd/semantic-class-color',
      `Use a semantic Tailwind color token instead of ${token}`,
    ])
  }
  for (const name of references(arbitraryValue(value))) {
    if (!known.has(name)) {
      issues.push([
        'ztd/defined-class-custom-property',
        `Undefined custom property ${name} in ${token}`,
      ])
    }
  }
  return issues
}
function inspect(file, code, known) {
  const source = ts.createSourceFile(file, code, ts.ScriptTarget.Latest, true)
  const warnings = []
  function visit(node) {
    if (staticLiteral(node)) {
      const location = source.getLineAndCharacterOfPosition(node.getStart(source))
      for (const token of node.text.split(/\s+/u)) {
        warnings.push(...tokenIssues(token, known).map(([rule, text]) => ({
          file,
          line: location.line + 1,
          column: location.character + 1,
          rule,
          text,
        })))
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(source)
  return warnings
}
export async function classWarnings(options, roots, cwd) {
  const groups = await Promise.all((options.classFiles ?? []).map(async (pattern) => {
    const selected = await glob(pattern, { cwd, absolute: true, onlyFiles: true, followSymbolicLinks: false })
    if (!selected.length) {
      throw new Error(`Class pattern matched no files: ${pattern}`)
    }
    return selected
  }))
  const known = customProperties(roots, options.externalCustomProperties)
  const results = await Promise.all([
    ...new Set(groups.flat()),
  ].sort().map(async file =>
    inspect(file, await readFile(file, 'utf8'), known)))
  return results.flat()
}
