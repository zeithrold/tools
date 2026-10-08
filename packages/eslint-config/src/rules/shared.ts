import type { Linter } from 'eslint'
import { unusedVariables } from '../unused-variables.js'

export const sharedRules: Linter.RulesRecord = {
  'complexity': [
    'error',
    {
      max: 10,
      variant: 'classic',
    },
  ],
  'sonarjs/cognitive-complexity': ['error', 15],
  'max-depth': [
    'error',
    {
      max: 4,
    },
  ],
  'max-lines-per-function': [
    'error',
    {
      max: 60,
      skipBlankLines: true,
      skipComments: true,
      IIFEs: true,
    },
  ],
  'max-lines': [
    'error',
    {
      max: 300,
      skipBlankLines: true,
      skipComments: true,
    },
  ],
  'max-params': [
    'error',
    {
      max: 4,
    },
  ],
  'style/max-len': [
    'error',
    {
      code: 120,
      comments: 120,
      tabWidth: 2,
      ignoreUrls: true,
      ignoreStrings: false,
      ignoreTemplateLiterals: false,
      ignoreRegExpLiterals: false,
      ignoreComments: false,
      ignoreTrailingComments: false,
    },
  ],
  'eqeqeq': ['error', 'always'],
  'curly': ['error', 'all'],
  'array-callback-return': [
    'error',
    {
      allowImplicit: false,
      checkForEach: true,
      allowVoid: false,
    },
  ],
  'no-empty': [
    'error',
    {
      allowEmptyCatch: false,
    },
  ],
  'no-shadow': [
    'error',
    {
      builtinGlobals: false,
      hoist: 'all',
      ignoreOnInitialization: false,
    },
  ],
  'no-nested-ternary': 'error',
  'no-unused-expressions': [
    'error',
    {
      allowShortCircuit: false,
      allowTernary: false,
      allowTaggedTemplates: true,
      enforceForJSX: true,
    },
  ],
  'eslint-comments/require-description': [
    'error',
    {
      ignore: ['eslint-enable'],
    },
  ],
  'unused-imports/no-unused-vars': unusedVariables,
  'no-constant-binary-expression': 'error',
  'no-unsafe-optional-chaining': [
    'error',
    {
      disallowArithmeticOperators: true,
    },
  ],
  'no-promise-executor-return': [
    'error',
    {
      allowVoid: false,
    },
  ],
  'no-setter-return': 'error',
}
