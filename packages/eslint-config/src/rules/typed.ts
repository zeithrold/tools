import type { Linter } from 'eslint'

export const typedRules: Linter.RulesRecord = {
  'ts/strict-boolean-expressions': [
    'error',
    {
      allowAny: false,
      allowNumber: false,
      allowString: false,
      allowNullableBoolean: false,
      allowNullableEnum: false,
      allowNullableNumber: false,
      allowNullableObject: false,
      allowNullableString: false,
    },
  ],
  'ts/no-floating-promises': [
    'error',
    {
      checkThenables: true,
      ignoreVoid: false,
      ignoreIIFE: false,
      allowForKnownSafeCalls: [],
      allowForKnownSafePromises: [],
    },
  ],
  'ts/no-misused-promises': [
    'error',
    {
      checksConditionals: {
        flagUnions: 'all',
      },
      checksSpreads: true,
      checksVoidReturn: true,
    },
  ],
  'ts/restrict-plus-operands': [
    'error',
    {
      allowAny: false,
      allowBoolean: false,
      allowNullish: false,
      allowNumberAndString: false,
      allowRegExp: false,
      skipCompoundAssignments: false,
    },
  ],
  'ts/restrict-template-expressions': [
    'error',
    {
      allow: [],
      allowAny: false,
      allowArray: false,
      allowBoolean: false,
      allowNullish: false,
      allowNumber: true,
      allowRegExp: false,
      allowNever: false,
    },
  ],
  'ts/return-await': ['error', 'always'],
  'ts/switch-exhaustiveness-check': [
    'error',
    {
      allowDefaultCaseForExhaustiveSwitch: true,
      considerDefaultExhaustiveForUnions: false,
      requireDefaultForNonUnion: true,
    },
  ],
  'ts/only-throw-error': [
    'error',
    {
      allow: [],
      allowRethrowing: true,
      allowThrowingAny: false,
      allowThrowingUnknown: false,
    },
  ],
  'ts/no-unsafe-type-assertion': 'error',
  'ts/use-unknown-in-catch-callback-variable': 'error',
  'ts/no-unnecessary-condition': [
    'error',
    {
      allowConstantLoopConditions: 'only-allowed-literals',
      checkTypePredicates: true,
    },
  ],
  'ts/no-base-to-string': [
    'error',
    {
      checkUnknown: true,
      ignoredTypeNames: [
        'Error',
        'RegExp',
        'URL',
        'URLSearchParams',
      ],
    },
  ],
  'ts/no-misused-spread': [
    'error',
    {
      allow: [],
    },
  ],
  'ts/no-unsafe-enum-comparison': 'error',
  'ts/no-deprecated': [
    'error',
    {
      allow: [],
    },
  ],
  'ts/consistent-type-exports': [
    'error',
    {
      fixMixedExportsWithInlineTypeSpecifier: false,
    },
  ],
  'ts/prefer-readonly': [
    'error',
    {
      onlyInlineLambdas: false,
    },
  ],
  'ts/require-array-sort-compare': [
    'error',
    {
      ignoreStringArrays: true,
    },
  ],
  'ts/require-await': 'error',
  'ts/prefer-nullish-coalescing': [
    'error',
    {
      ignoreConditionalTests: true,
      ignoreMixedLogicalExpressions: false,
      ignorePrimitives: {
        bigint: false,
        boolean: false,
        number: false,
        string: false,
      },
    },
  ],
  'ts/prefer-optional-chain': [
    'error',
    {
      requireNullish: true,
      allowPotentiallyUnsafeFixesThatModifyTheReturnTypeIKnowWhatImDoing: false,
    },
  ],
  'ts/no-array-delete': 'error',
  'ts/no-unsafe-unary-minus': 'error',
  'ts/prefer-promise-reject-errors': [
    'error',
    {
      allowEmptyReject: false,
      allowThrowingAny: false,
      allowThrowingUnknown: false,
    },
  ],
  'ts/await-thenable': 'error',
  'ts/no-for-in-array': 'error',
  'ts/no-implied-eval': 'error',
  'ts/no-unnecessary-type-assertion': 'error',
  'ts/no-unsafe-argument': 'error',
  'ts/no-unsafe-assignment': 'error',
  'ts/no-unsafe-call': 'error',
  'ts/no-unsafe-member-access': 'error',
  'ts/no-unsafe-return': 'error',
  'ts/unbound-method': 'error',
}
