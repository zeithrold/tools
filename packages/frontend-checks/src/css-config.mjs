import { createRequire } from 'node:module'
import standard from 'stylelint-config-standard'

const require = createRequire(import.meta.url)

export const cssConfig = {
  ...standard,
  extends: require.resolve('stylelint-config-standard'),
  rules: {
    ...standard.rules,
    'import-notation': 'string',
    'at-rule-no-unknown': [
      true,
      {
        ignoreAtRules: [
          'theme',
          'utility',
          'variant',
          'custom-variant',
          'apply',
          'reference',
          'source',
          'config',
          'plugin',
        ],
      },
    ],
    'nesting-selector-no-missing-scoping-root': [
      true,
      { ignoreAtRules: ['custom-variant', 'utility'] },
    ],
    'custom-property-pattern': [
      '^(?:[a-z][a-z0-9]*(?:-[a-z0-9]+)*|text-[a-z][a-z0-9-]*--(?:line-height|letter-spacing|font-weight))$',
    ],
    'function-no-unknown': [
      true,
      { ignoreFunctions: ['--alpha', '--spacing'] },
    ],
  },
}
