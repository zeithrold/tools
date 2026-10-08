import type { Linter } from 'eslint'

export const tsSyntaxRules: Linter.RulesRecord = {
  'ts/consistent-type-definitions': ['error', 'type'],
  'ts/no-explicit-any': [
    'error',
    {
      fixToUnknown: false,
      ignoreRestArgs: false,
    },
  ],
  'ts/no-non-null-assertion': 'error',
  'ts/no-empty-object-type': [
    'error',
    {
      allowInterfaces: 'never',
      allowObjectTypes: 'never',
    },
  ],
  'ts/ban-ts-comment': [
    'error',
    {
      'ts-ignore': true,
      'ts-nocheck': true,
      'ts-check': false,
      'ts-expect-error': 'allow-with-description',
      'minimumDescriptionLength': 10,
    },
  ],
  'ts/consistent-type-assertions': [
    'error',
    {
      assertionStyle: 'as',
      objectLiteralTypeAssertions: 'never',
      arrayLiteralTypeAssertions: 'never',
    },
  ],
  'ts/explicit-module-boundary-types': [
    'error',
    {
      allowArgumentsExplicitlyTypedAsAny: false,
      allowDirectConstAssertionInArrowFunctions: false,
      allowHigherOrderFunctions: false,
      allowOverloadFunctions: false,
      allowTypedFunctionExpressions: true,
    },
  ],
  'ts/no-useless-constructor': 'error',
  'ts/no-invalid-void-type': [
    'error',
    {
      allowAsThisParameter: true,
      allowInGenericTypeArguments: true,
    },
  ],
  'ts/no-extraneous-class': [
    'error',
    {
      allowConstructorOnly: false,
      allowEmpty: false,
      allowStaticOnly: false,
      allowWithDecorator: true,
    },
  ],
  'ts/max-params': [
    'error',
    {
      max: 4,
      countVoidThis: false,
    },
  ],
  'ts/no-shadow': [
    'error',
    {
      builtinGlobals: false,
      hoist: 'functions-and-types',
      ignoreOnInitialization: false,
      ignoreTypeValueShadow: true,
      ignoreFunctionTypeParameterNameValueShadow: true,
    },
  ],
  'ts/no-unused-expressions': [
    'error',
    {
      allowShortCircuit: false,
      allowTernary: false,
      allowTaggedTemplates: true,
      enforceForJSX: true,
    },
  ],
}
