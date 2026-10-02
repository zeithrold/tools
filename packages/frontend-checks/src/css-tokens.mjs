import colors from 'color-name'
import valueParser from 'postcss-value-parser'

const colorFunctions = /^(?:rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch|color)$/i
const colorProperties = /^color$|color$|^background(?:$|-)|^border(?:$|-)|^outline(?:$|-)|shadow$|^fill$|^stroke$/i

function references(value) {
  const names = []
  valueParser(value).walk((node) => {
    if (node.type !== 'function' || node.value !== 'var') {
      return
    }
    const first = node.nodes.find(child => child.type !== 'space' && child.type !== 'comment')
    if (first?.type === 'word' && first.value.startsWith('--')) {
      names.push(first.value)
    }
  })
  return names
}

function hasLiteralColor(value) {
  let found = false
  valueParser(value).walk((node) => {
    if (node.type === 'function' && node.value === 'url') {
      return false
    }
    if (node.type === 'function' && colorFunctions.test(node.value)) {
      found = true
    }
    const literalWord = node.type === 'word'
      && (/^#[\da-f]{3,8}$/i.test(node.value) || Object.hasOwn(colors, node.value.toLowerCase()))
    if (literalWord) {
      found = true
    }
  })
  return found
}

// This is a closed declaration inventory, not a proof of the runtime cascade.
export function tokenWarnings(sources, definitions, options) {
  const known = new Set(options.externalCustomProperties ?? [])
  for (const root of [
    ...sources,
    ...definitions,
  ]) {
    root.walkDecls((declaration) => {
      if (declaration.prop.startsWith('--') && !declaration.prop.includes('*')) {
        known.add(declaration.prop)
      }
    })
    root.walkAtRules('property', rule => known.add(rule.params.trim()))
  }
  const warnings = []
  for (const root of sources) {
    root.walkDecls((declaration) => {
      const issue = (rule, text) => warnings.push({
        file: root.source.input.file,
        line: declaration.source.start.line,
        column: declaration.source.start.column,
        rule,
        text,
      })
      for (const name of references(declaration.value)) {
        if (!known.has(name)) {
          issue('ztd/defined-custom-property', `Undefined custom property ${name}; include its declaration source`)
        }
      }
      if (declaration.prop.startsWith('--') || !colorProperties.test(declaration.prop)) {
        return
      }
      if (hasLiteralColor(declaration.value)) {
        issue('ztd/semantic-color', 'Move literal colors into project-owned semantic custom properties')
      }
    })
  }
  return warnings
}
