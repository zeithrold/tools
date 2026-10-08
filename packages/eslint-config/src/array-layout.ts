import type { AST, Rule, SourceCode } from 'eslint'
import type { Node } from 'estree'

import { isRecord, isSimple } from './array-simple.js'

type Tokens = Pick<SourceCode, 'getFirstToken' | 'getLastToken' | 'getTokenBefore' | 'getCommentsInside'>

function fitsLine(source: SourceCode, node: Node, text: string, tokens: Tokens): boolean {
  const start = tokens.getFirstToken(node)
  const end = tokens.getLastToken(node)
  if (start === null || end === null) {
    return false
  }
  const prefix = source.lines[start.loc.start.line - 1]?.slice(0, start.loc.start.column) ?? ''
  const suffix = source.lines[end.loc.end.line - 1]?.slice(end.loc.end.column) ?? ''
  return Array.from(`${prefix}${text}${suffix}`.replaceAll('\t', '  ')).length <= 120
}

type TemplateServices = {
  getTemplateBodyTokenStore: () => Tokens
  defineTemplateBodyVisitor: (template: Rule.RuleListener, script: Rule.RuleListener) => Rule.RuleListener
}

function hasTemplateVisitor(value: unknown): value is TemplateServices {
  return isRecord(value) && typeof value.defineTemplateBodyVisitor === 'function'
    && typeof value.getTemplateBodyTokenStore === 'function'
}

function gapBefore(source: Tokens, token: AST.Token | null): [number, number][] {
  if (token === null) {
    return []
  }
  const previous = source.getTokenBefore(token)
  if (previous === null || previous.loc.end.line !== token.loc.start.line) {
    return []
  }
  return [
    [
      previous.range[1],
      token.range[0],
    ],
  ]
}

function expand(context: Rule.RuleContext, node: Node, elements: Node[], tokens: Tokens): void {
  const source = context.sourceCode
  const open = tokens.getFirstToken(node)
  const close = tokens.getLastToken(node)
  if (open === null || close === null) {
    return
  }
  const ranges = elements.flatMap(element => gapBefore(tokens, tokens.getFirstToken(element)))
  ranges.push(...gapBefore(tokens, close))
  if (ranges.length === 0) {
    return
  }
  const indent = source.lines[open.loc.start.line - 1]?.match(/^\s*/u)?.[0] ?? ''
  const newline = source.text.includes('\r\n') ? '\r\n' : '\n'
  context.report({
    node,
    messageId: 'expand',
    fix(fixer) {
      // Comments remain in place; a non-whitespace gap requires a manual layout edit.
      if (ranges.some(([start, end]) => /\S/u.test(source.text.slice(start, end)))) {
        return null
      }
      return ranges.map(range => fixer.replaceTextRange(
        range,
        `${newline}${indent}${range[1] === close.range[0] ? '' : '  '}`,
      ))
    },
  })
}

function createVisitor(context: Rule.RuleContext, tokens: Tokens): Rule.RuleListener {
  return {
    ArrayExpression(node) {
      const elements = node.elements.filter(element => element !== null)
      if (elements.length !== node.elements.length) {
        return
      }
      const source = context.sourceCode
      const complex = elements.some(element => !isSimple(element) || /[\r\n]/u.test(source.getText(element)))
      if (elements.length >= 3 || complex) {
        expand(context, node, elements, tokens)
        return
      }
      if (tokens.getCommentsInside(node).length > 0) {
        return
      }
      const compact = `[${elements.map(element => source.getText(element)).join(', ')}]`
      if (elements.length > 0 && !fitsLine(source, node, compact, tokens)) {
        expand(context, node, elements, tokens)
        return
      }
      if (/[\r\n]/u.test(source.getText(node)) && fitsLine(source, node, compact, tokens)) {
        context.report({ node, messageId: 'collapse', fix: fixer => fixer.replaceText(node, compact) })
      }
    },
  }
}

export const arrayLayout: Rule.RuleModule = {
  meta: {
    type: 'layout',
    docs: { description: 'Expand arrays of at least three items or complex items; collapse safe short arrays' },
    fixable: 'whitespace',
    schema: [],
    messages: {
      expand: 'Use separate bracket and element lines for arrays with at least 3 items or complex/multiline items.',
      collapse: 'Keep a simple array with fewer than 3 items on one line when the complete line fits 120 characters.',
    },
  },
  create(context) {
    const script = createVisitor(context, context.sourceCode)
    const services: unknown = context.sourceCode.parserServices
    if (hasTemplateVisitor(services)) {
      return services.defineTemplateBodyVisitor(createVisitor(context, services.getTemplateBodyTokenStore()), script)
    }
    return script
  },
}
