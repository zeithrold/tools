import antfu from '@antfu/eslint-config'
import { builtinRules } from 'eslint/use-at-your-own-risk'

// Reuse the actual resolved antfu plugin objects, including its synthetic test plugin.
// No guessing npm names and no replacement of @eslint-react with eslint-plugin-react.
export async function resolvedPlugin(namespace, options) {
  if (namespace === 'core-js')
    return { meta: { name: 'antfu-core-js' }, rules: Object.fromEntries(builtinRules) }
  const configs = await antfu(structuredClone(options))
  const plugins = Object.assign({}, ...configs.map(c => c.plugins ?? {}))
  if (!plugins[namespace])
    throw new Error(`Missing resolved antfu plugin ${namespace}`)
  return plugins[namespace]
}

export const contextProbe = {
  meta: { name: 'bridge-probe' },
  rules: {
    context: {
      meta: { type: 'problem', schema: [], messages: { probe: '{{data}}' } },
      create(context) {
        let codePaths = 0
        return {
          onCodePathStart() { codePaths++ },
          'Program:exit': function (node) {
            const source = context.sourceCode
            context.report({
              node,
              messageId: 'probe',
              data: { data: JSON.stringify({
                sourceType: node.sourceType,
                tokens: source.getTokens(node).length,
                comments: source.getAllComments().length,
                variables: source.getScope(node).variables.map(v => v.name).filter(n => n === 'value'),
                codePaths,
                parserServices: Object.keys(source.parserServices),
                setting: context.settings.bridgeProbe,
                filename: context.filename.split('/').at(-1),
              }) },
            })
          },
        }
      },
    },
  },
}
