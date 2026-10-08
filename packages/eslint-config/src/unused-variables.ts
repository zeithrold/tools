import type { Linter } from 'eslint'
import type { LocalConfig } from './options.js'

export const unusedVariables: Linter.RuleEntry = [
  'error',
  {
    args: 'all',
    argsIgnorePattern: '^_',
    caughtErrors: 'all',
    caughtErrorsIgnorePattern: '^_',
    vars: 'all',
    varsIgnorePattern: '^_',
    ignoreRestSiblings: true,
    reportUsedIgnorePattern: true,
  },
]

/** Native JS scopes do not expose TypeScript value/type reference flags. */
export function javascriptUnusedVariables(files: string[]): LocalConfig[] {
  return [
    {
      name: 'ztd/javascript-unused-variables',
      files,
      rules: {
        'no-unused-vars': unusedVariables,
        'unused-imports/no-unused-vars': 'off',
      },
    },
    {
      name: 'ztd/javascript-examples',
      files: files.map(file => ['**/*.md/**', file]),
      // Preserve antfu/markdown/disables/code for incomplete extracted examples.
      rules: {
        'no-unused-vars': 'off',
        'unused-imports/no-unused-vars': 'off',
        'unused-imports/no-unused-imports': 'off',
      },
    },
  ]
}
