import type { Linter } from 'eslint'

export const testsRules: Linter.RulesRecord = {
  'test/valid-expect': [
    'error',
    {
      alwaysAwait: true,
      minArgs: 1,
      maxArgs: 1,
    },
  ],
  'test/no-conditional-expect': [
    'error',
    {
      expectAssertions: false,
    },
  ],
  'test/no-conditional-tests': 'error',
  'test/no-disabled-tests': 'error',
  'test/expect-expect': [
    'error',
    {
      assertFunctionNames: ['expect', 'assert'],
      additionalTestBlockFunctions: [],
    },
  ],
  'test/unbound-method': [
    'error',
    {
      ignoreStatic: false,
    },
  ],
}
