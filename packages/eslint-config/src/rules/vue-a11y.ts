import type { Linter } from 'eslint'

export const vueA11yRules: Linter.RulesRecord = {
  'vue-a11y/alt-text': 'error',
  'vue-a11y/anchor-has-content': 'error',
  'vue-a11y/aria-props': 'error',
  'vue-a11y/aria-role': 'error',
  'vue-a11y/aria-unsupported-elements': 'error',
  'vue-a11y/click-events-have-key-events': 'error',
  'vue-a11y/form-control-has-label': 'error',
  'vue-a11y/heading-has-content': 'error',
  'vue-a11y/iframe-has-title': 'error',
  'vue-a11y/interactive-supports-focus': 'error',
  'vue-a11y/label-has-for': [
    'error',
    {
      required: {
        some: ['nesting', 'id'],
      },
      allowChildren: false,
    },
  ],
  'vue-a11y/media-has-caption': 'error',
  'vue-a11y/mouse-events-have-key-events': 'error',
  'vue-a11y/no-access-key': 'error',
  'vue-a11y/no-aria-hidden-on-focusable': 'error',
  'vue-a11y/no-autofocus': 'error',
  'vue-a11y/no-distracting-elements': 'error',
  'vue-a11y/no-redundant-roles': 'error',
  'vue-a11y/no-role-presentation-on-focusable': 'error',
  'vue-a11y/no-static-element-interactions': 'error',
  'vue-a11y/role-has-required-aria-props': 'error',
  'vue-a11y/tabindex-no-positive': 'error',
}
