import ts from '@typescript-eslint/eslint-plugin'

// Pinned to @typescript-eslint/eslint-plugin 8.71.0. This rule only reads the two
// decorator flags from getParserServices(context, true); it never reads TS maps
// or a Program. Keep a hard guard so a future upstream change cannot silently
// pretend to provide TypeScript semantic information.
export function syntaxTypeImportsRule({ experimentalDecorators = false, emitDecoratorMetadata = false } = {}) {
  if (ts.meta.version !== '8.71.0')
    throw new Error('Syntax adapter requires reviewed typescript-eslint 8.71.0; use ESLint')
  if (experimentalDecorators || emitDecoratorMetadata)
    throw new Error('Decorator metadata requires the original TS parser; use ESLint')
  const original = ts.rules['consistent-type-imports']
  const unavailableMap = Object.freeze({
    get() { throw new Error('TS node mapping is unavailable; use ESLint') },
    has() { throw new Error('TS node mapping is unavailable; use ESLint') },
  })
  const services = Object.freeze({
    esTreeNodeToTSNodeMap: unavailableMap,
    tsNodeToESTreeNodeMap: unavailableMap,
    experimentalDecorators: false,
    emitDecoratorMetadata: false,
  })
  return {
    meta: original.meta,
    defaultOptions: original.defaultOptions,
    create(context) {
      const originalSource = context.sourceCode
      const sourceFacade = new Proxy({}, {
        get(_target, key) {
          if (key === 'parserServices')
            return services
          const value = Reflect.get(originalSource, key, originalSource)
          return typeof value === 'function' ? value.bind(originalSource) : value
        },
      })
      const facade = Object.create(context)
      Object.defineProperty(facade, 'sourceCode', { value: sourceFacade })
      return original.create(facade)
    },
  }
}

export default {
  meta: { name: 'antfu-syntax-adapter' },
  rules: { 'consistent-type-imports': syntaxTypeImportsRule() },
}
