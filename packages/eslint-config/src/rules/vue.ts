import type { Linter } from 'eslint'

export const vueRules: Linter.RulesRecord = {
  'vue/no-dupe-keys': 'error',
  'vue/no-v-html': 'error',
  'vue/no-setup-props-reactivity-loss': 'error',
  'vue/no-ref-object-reactivity-loss': 'error',
  'vue/no-mutating-props': [
    'error',
    {
      shallowOnly: false,
    },
  ],
  'vue/require-explicit-emits': [
    'error',
    {
      allowProps: false,
    },
  ],
  'vue/require-explicit-slots': 'error',
  'vue/require-prop-types': 'error',
  'vue/no-undef-components': [
    'error',
    {
      ignorePatterns: [],
    },
  ],
  'vue/no-undef-directives': [
    'error',
    {
      ignore: [],
    },
  ],
  'vue/no-unused-emit-declarations': 'error',
  'vue/html-button-has-type': [
    'error',
    {
      button: true,
      submit: true,
      reset: true,
    },
  ],
  'vue/no-template-target-blank': [
    'error',
    {
      allowReferrer: false,
      enforceDynamicLinks: 'always',
    },
  ],
  'vue/eqeqeq': ['error', 'always'],
  'vue/max-lines-per-block': [
    'error',
    {
      script: 200,
      template: 200,
      style: 150,
      skipBlankLines: true,
    },
  ],
  'vue/max-template-depth': [
    'error',
    {
      maxDepth: 5,
    },
  ],
  'vue/max-len': [
    'error',
    {
      code: 120,
      template: 120,
      comments: 120,
      tabWidth: 2,
      ignoreUrls: true,
      ignoreStrings: false,
      ignoreTemplateLiterals: false,
      ignoreRegExpLiterals: false,
      ignoreHTMLAttributeValues: false,
      ignoreHTMLTextContents: false,
    },
  ],
  'vue/no-template-shadow': 'error',
  'vue/one-component-per-file': 'error',
  'vue/no-required-prop-with-default': 'error',
  'vue/html-end-tags': 'error',
  'vue/no-async-in-computed-properties': 'error',
  'vue/no-side-effects-in-computed-properties': 'error',
  'vue/no-use-v-if-with-v-for': 'error',
  'vue/require-v-for-key': 'error',
  'vue/no-watch-after-await': 'error',
  'vue/no-lifecycle-after-await': 'error',
  'vue/no-expose-after-await': 'error',
}
