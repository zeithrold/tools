import postcss from 'postcss'
import stylelint from 'stylelint'

// Stylelint treats unknown block at-rules as descriptor scopes, skipping property validation.
// Recheck these native Tailwind declaration scopes as ordinary rules with the same source locations.
export async function tailwindPropertyWarnings(roots) {
  const reports = await Promise.all(roots.map(async (root) => {
    const declarations = []
    root.walkAtRules(/^(?:utility|custom-variant|variant)$/u, (rule) => {
      rule.walkDecls(declaration => declarations.push(declaration))
    })
    if (!declarations.length) {
      return []
    }
    const probe = postcss.root()
    const originals = new Map()
    declarations.forEach((declaration, index) => {
      const selector = `.ztd-tailwind-probe-${index}`
      probe.append(postcss.rule({ selector, nodes: [
        declaration.clone(),
      ] }))
      originals.set(selector, declaration)
    })
    const code = probe.toString()
    const locations = new Map()
    postcss.parse(code).walkDecls(declaration =>
      locations.set(declaration.source.start.line, originals.get(declaration.parent.selector)))
    const lint = await stylelint.lint({
      code,
      codeFilename: root.source.input.file,
      config: { rules: { 'property-no-unknown': true } },
    })
    return lint.results.flatMap(result => result.warnings.map((warning) => {
      const declaration = locations.get(warning.line)
      return {
        file: root.source.input.file,
        line: declaration?.source.start.line ?? warning.line,
        column: declaration?.source.start.column ?? warning.column,
        rule: warning.rule,
        text: warning.text,
      }
    }))
  }))
  return reports.flat()
}
