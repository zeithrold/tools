# 规则清单（由 `npm run summarize` 生成）

分母是每个 profile 在 21 个记录路径上的有效配置中，severity > 0 的唯一 rule ID。候选分类不代表语义等价或生产覆盖率。同一 ID 在 JS 上有候选实现、在非 JS 文件上仍可能需要 ESLint。完整 options / severity / 文件分组见 [inventory.json](results/inventory.json)；全部候选 ID 见 [rule-matrix.csv](results/rule-matrix.csv)。

| Profile | 分母 | 原生候选 | JS 插件候选 | 文件/类型模型兜底 |
| --- | ---: | ---: | ---: | ---: |
| automatic | 426 | 151 | 184 | 91 |
| base | 426 | 151 | 184 | 91 |
| editor | 426 | 151 | 184 | 91 |
| library | 427 | 152 | 184 | 91 |
| typed | 445 | 170 | 184 | 91 |
| react | 512 | 171 | 249 | 92 |
| vue | 576 | 151 | 184 | 241 |
| svelte | 471 | 151 | 184 | 136 |
| astro | 434 | 151 | 184 | 99 |
| solid | 441 | 151 | 199 | 91 |
| formatters | 427 | 151 | 184 | 92 |

## 基础配置全部 91 个兜底 ID

这些 ID 只在所采样的非 JS 文件配置出现。其 parser、language、processor 或 AST 无法由当前 Oxlint JS 插件 API 直接提供；并不是说原规则自身没有 JS 实现。

### jsonc（38）

- `jsonc/array-bracket-spacing`
- `jsonc/comma-dangle`
- `jsonc/comma-style`
- `jsonc/indent`
- `jsonc/key-spacing`
- `jsonc/no-bigint-literals`
- `jsonc/no-binary-expression`
- `jsonc/no-binary-numeric-literals`
- `jsonc/no-dupe-keys`
- `jsonc/no-escape-sequence-in-identifier`
- `jsonc/no-floating-decimal`
- `jsonc/no-hexadecimal-numeric-literals`
- `jsonc/no-infinity`
- `jsonc/no-multi-str`
- `jsonc/no-nan`
- `jsonc/no-number-props`
- `jsonc/no-numeric-separators`
- `jsonc/no-octal`
- `jsonc/no-octal-escape`
- `jsonc/no-octal-numeric-literals`
- `jsonc/no-parenthesized`
- `jsonc/no-plus-sign`
- `jsonc/no-regexp-literals`
- `jsonc/no-sparse-arrays`
- `jsonc/no-template-literals`
- `jsonc/no-undefined-value`
- `jsonc/no-unicode-codepoint-escapes`
- `jsonc/no-useless-escape`
- `jsonc/object-curly-newline`
- `jsonc/object-curly-spacing`
- `jsonc/object-property-newline`
- `jsonc/quote-props`
- `jsonc/quotes`
- `jsonc/sort-array-values`
- `jsonc/sort-keys`
- `jsonc/space-unary-ops`
- `jsonc/valid-json-number`
- `jsonc/vue-custom-block/no-parsing-error`

### markdown（15）

- `markdown/heading-increment`
- `markdown/no-duplicate-definitions`
- `markdown/no-empty-definitions`
- `markdown/no-empty-images`
- `markdown/no-empty-links`
- `markdown/no-invalid-label-refs`
- `markdown/no-missing-atx-heading-space`
- `markdown/no-missing-link-fragments`
- `markdown/no-multiple-h1`
- `markdown/no-reference-like-urls`
- `markdown/no-reversed-media-syntax`
- `markdown/no-space-in-emphasis`
- `markdown/no-unused-definitions`
- `markdown/require-alt-text`
- `markdown/table-column-count`

### toml（19）

- `toml/array-bracket-newline`
- `toml/array-bracket-spacing`
- `toml/array-element-newline`
- `toml/comma-style`
- `toml/indent`
- `toml/inline-table-curly-spacing`
- `toml/key-spacing`
- `toml/keys-order`
- `toml/no-space-dots`
- `toml/no-unreadable-number-separator`
- `toml/padding-line-between-pairs`
- `toml/padding-line-between-tables`
- `toml/precision-of-fractional-seconds`
- `toml/precision-of-integer`
- `toml/quoted-keys`
- `toml/spaced-comment`
- `toml/table-bracket-spacing`
- `toml/tables-order`
- `toml/vue-custom-block/no-parsing-error`

### yaml（19）

- `yaml/block-mapping`
- `yaml/block-mapping-question-indicator-newline`
- `yaml/block-sequence`
- `yaml/block-sequence-hyphen-indicator-newline`
- `yaml/flow-mapping-curly-newline`
- `yaml/flow-mapping-curly-spacing`
- `yaml/flow-sequence-bracket-newline`
- `yaml/flow-sequence-bracket-spacing`
- `yaml/indent`
- `yaml/key-spacing`
- `yaml/no-empty-key`
- `yaml/no-empty-sequence-entry`
- `yaml/no-irregular-whitespace`
- `yaml/no-multiple-empty-lines`
- `yaml/no-tab-indent`
- `yaml/plain-scalar`
- `yaml/quotes`
- `yaml/spaced-comment`
- `yaml/vue-custom-block/no-parsing-error`

## 各 profile 在基础清单之外增加的兜底 ID

### react（1）

- `react/no-leaked-conditional-rendering`

### vue（150）

- `vue/array-bracket-spacing`
- `vue/arrow-spacing`
- `vue/attribute-hyphenation`
- `vue/attributes-order`
- `vue/block-order`
- `vue/block-spacing`
- `vue/block-tag-newline`
- `vue/brace-style`
- `vue/comma-dangle`
- `vue/comma-spacing`
- `vue/comma-style`
- `vue/comment-directive`
- `vue/component-definition-name-casing`
- `vue/component-name-in-template-casing`
- `vue/component-options-name-casing`
- `vue/custom-event-name-casing`
- `vue/define-macros-order`
- `vue/dot-location`
- `vue/dot-notation`
- `vue/eqeqeq`
- `vue/first-attribute-linebreak`
- `vue/html-closing-bracket-newline`
- `vue/html-closing-bracket-spacing`
- `vue/html-comment-content-spacing`
- `vue/html-end-tags`
- `vue/html-indent`
- `vue/html-quotes`
- `vue/html-self-closing`
- `vue/jsx-uses-vars`
- `vue/key-spacing`
- `vue/keyword-spacing`
- `vue/multiline-html-element-content-newline`
- `vue/mustache-interpolation-spacing`
- `vue/no-arrow-functions-in-watch`
- `vue/no-async-in-computed-properties`
- `vue/no-child-content`
- `vue/no-computed-properties-in-data`
- `vue/no-deprecated-data-object-declaration`
- `vue/no-deprecated-delete-set`
- `vue/no-deprecated-destroyed-lifecycle`
- `vue/no-deprecated-dollar-listeners-api`
- `vue/no-deprecated-dollar-scopedslots-api`
- `vue/no-deprecated-events-api`
- `vue/no-deprecated-filter`
- `vue/no-deprecated-functional-template`
- `vue/no-deprecated-html-element-is`
- `vue/no-deprecated-inline-template`
- `vue/no-deprecated-model-definition`
- `vue/no-deprecated-props-default-this`
- `vue/no-deprecated-router-link-tag-prop`
- `vue/no-deprecated-scope-attribute`
- `vue/no-deprecated-slot-attribute`
- `vue/no-deprecated-slot-scope-attribute`
- `vue/no-deprecated-v-bind-sync`
- `vue/no-deprecated-v-is`
- `vue/no-deprecated-v-on-native-modifier`
- `vue/no-deprecated-v-on-number-modifiers`
- `vue/no-deprecated-vue-config-keycodes`
- `vue/no-dupe-v-else-if`
- `vue/no-duplicate-attributes`
- `vue/no-empty-pattern`
- `vue/no-export-in-script-setup`
- `vue/no-expose-after-await`
- `vue/no-irregular-whitespace`
- `vue/no-lifecycle-after-await`
- `vue/no-lone-template`
- `vue/no-loss-of-precision`
- `vue/no-multi-spaces`
- `vue/no-multiple-slot-args`
- `vue/no-mutating-props`
- `vue/no-parsing-error`
- `vue/no-ref-as-operand`
- `vue/no-required-prop-with-default`
- `vue/no-reserved-component-names`
- `vue/no-reserved-keys`
- `vue/no-reserved-props`
- `vue/no-restricted-syntax`
- `vue/no-restricted-v-bind`
- `vue/no-shared-component-data`
- `vue/no-side-effects-in-computed-properties`
- `vue/no-spaces-around-equal-signs-in-attribute`
- `vue/no-sparse-arrays`
- `vue/no-template-key`
- `vue/no-template-shadow`
- `vue/no-textarea-mustache`
- `vue/no-unused-components`
- `vue/no-unused-refs`
- `vue/no-unused-vars`
- `vue/no-use-computed-property-like-method`
- `vue/no-use-v-if-with-v-for`
- `vue/no-useless-template-attributes`
- `vue/no-useless-v-bind`
- `vue/no-v-for-template-key-on-child`
- `vue/no-v-text-v-html-on-component`
- `vue/no-watch-after-await`
- `vue/object-curly-spacing`
- `vue/object-property-newline`
- `vue/object-shorthand`
- `vue/one-component-per-file`
- `vue/operator-linebreak`
- `vue/order-in-components`
- `vue/padding-line-between-blocks`
- `vue/prefer-import-from-vue`
- `vue/prefer-separate-static-class`
- `vue/prefer-template`
- `vue/prop-name-casing`
- `vue/quote-props`
- `vue/require-component-is`
- `vue/require-explicit-emits`
- `vue/require-prop-type-constructor`
- `vue/require-render-return`
- `vue/require-slots-as-functions`
- `vue/require-toggle-inside-transition`
- `vue/require-v-for-key`
- `vue/require-valid-default-prop`
- `vue/return-in-computed-property`
- `vue/return-in-emits-validator`
- `vue/singleline-html-element-content-newline`
- `vue/space-in-parens`
- `vue/space-infix-ops`
- `vue/space-unary-ops`
- `vue/template-curly-spacing`
- `vue/this-in-template`
- `vue/use-v-on-exact`
- `vue/v-bind-style`
- `vue/v-on-event-hyphenation`
- `vue/v-on-style`
- `vue/v-slot-style`
- `vue/valid-attribute-name`
- `vue/valid-define-emits`
- `vue/valid-define-options`
- `vue/valid-define-props`
- `vue/valid-next-tick`
- `vue/valid-template-root`
- `vue/valid-v-bind`
- `vue/valid-v-cloak`
- `vue/valid-v-else`
- `vue/valid-v-else-if`
- `vue/valid-v-for`
- `vue/valid-v-html`
- `vue/valid-v-if`
- `vue/valid-v-is`
- `vue/valid-v-memo`
- `vue/valid-v-model`
- `vue/valid-v-on`
- `vue/valid-v-once`
- `vue/valid-v-pre`
- `vue/valid-v-show`
- `vue/valid-v-slot`
- `vue/valid-v-text`

### svelte（45）

- `svelte/comment-directive`
- `svelte/derived-has-same-inputs-outputs`
- `svelte/html-closing-bracket-spacing`
- `svelte/html-quotes`
- `svelte/indent`
- `svelte/infinite-reactive-loop`
- `svelte/mustache-spacing`
- `svelte/no-at-debug-tags`
- `svelte/no-at-html-tags`
- `svelte/no-dom-manipulating`
- `svelte/no-dupe-else-if-blocks`
- `svelte/no-dupe-on-directives`
- `svelte/no-dupe-style-properties`
- `svelte/no-dupe-use-directives`
- `svelte/no-export-load-in-svelte-module-in-kit-pages`
- `svelte/no-immutable-reactive-statements`
- `svelte/no-inner-declarations`
- `svelte/no-inspect`
- `svelte/no-navigation-without-resolve`
- `svelte/no-not-function-handler`
- `svelte/no-object-in-text-mustaches`
- `svelte/no-raw-special-elements`
- `svelte/no-reactive-functions`
- `svelte/no-reactive-literals`
- `svelte/no-reactive-reassign`
- `svelte/no-shorthand-style-property-overrides`
- `svelte/no-spaces-around-equal-signs-in-attribute`
- `svelte/no-store-async`
- `svelte/no-svelte-internal`
- `svelte/no-trailing-spaces`
- `svelte/no-unknown-style-directive-property`
- `svelte/no-unnecessary-state-wrap`
- `svelte/no-unused-props`
- `svelte/no-unused-svelte-ignore`
- `svelte/no-useless-children-snippet`
- `svelte/no-useless-mustaches`
- `svelte/prefer-svelte-reactivity`
- `svelte/prefer-writable-derived`
- `svelte/require-each-key`
- `svelte/require-event-dispatcher-types`
- `svelte/require-store-reactive-access`
- `svelte/spaced-html-comment`
- `svelte/system`
- `svelte/valid-each-key`
- `svelte/valid-prop-names-in-kit-pages`

### astro（8）

- `astro/missing-client-only-directive-value`
- `astro/no-conflict-set-directives`
- `astro/no-deprecated-astro-canonicalurl`
- `astro/no-deprecated-astro-fetchcontent`
- `astro/no-deprecated-astro-resolve`
- `astro/no-deprecated-getentrybyslug`
- `astro/no-unused-define-vars-in-style`
- `astro/valid-compile`

### formatters（1）

- `format/prettier`

## 定向 specimen 的实际结果

计数是记录的诊断数；JS 运行异常单列，不能当作规则诊断。修复相同只表示该片段最终文本相同。inline options 的反例虽然计数相同，报错行不同，见 README。类型感知、文件格式、匹配、混合与副作用实验在 experiments.json 的独立字段。

| Specimen | 原 ID | ESLint / Oxlint 诊断 | 运行异常 | severity 计数一致 | 修复文本一致 |
| --- | --- | --- | --- | --- | --- |
| console-options-shadowing | no-console | 3 / 2 | false | false | true |
| console-original-core-js | no-console | 3 / 3 | false | true | true |
| console-warn | no-console | 1 / 1 | false | true | true |
| eqeqeq-smart | eqeqeq | 1 / 1 | false | true | true |
| prefer-const-options | prefer-const | 1 / 1 | false | true | true |
| ban-ts-comment | ts/ban-ts-comment | 1 / 1 | false | true | true |
| type-definitions | ts/consistent-type-definitions | 1 / 1 | false | true | true |
| type-imports-native | ts/consistent-type-imports | 1 / 1 | false | true | true |
| type-imports-js | ts/consistent-type-imports | 1 / 1 | true | false | false |
| type-imports-adapted | ts/consistent-type-imports | 1 / 1 | false | true | true |
| type-imports-adapted-comments | ts/consistent-type-imports | 1 / 1 | false | true | true |
| import-lite-options-native-raw | import/consistent-type-specifier-style | 1 / 配置失败 | true | false | 未执行 |
| import-lite-options-native-translated | import/consistent-type-specifier-style | 1 / 1 | false | true | false |
| import-lite-original-js | import/consistent-type-specifier-style | 1 / 1 | false | true | true |
| style-semi | style/semi | 1 / 1 | false | true | true |
| style-quotes | style/quotes | 2 / 2 | false | true | true |
| style-indent | style/indent | 3 / 3 | false | true | true |
| style-comma-dangle | style/comma-dangle | 1 / 1 | false | true | true |
| antfu-curly | antfu/curly | 1 / 1 | false | true | true |
| antfu-top-level-await | antfu/no-top-level-await | 1 / 1 | false | true | true |
| core-selector-js | no-restricted-syntax | 2 / 2 | false | true | true |
| perfectionist-sort-imports | perfectionist/sort-imports | 1 / 1 | false | true | true |
| unused-imports-autofix | unused-imports/no-unused-imports | 1 / 1 | false | true | true |
| unused-imports-side-effect-risk | unused-imports/no-unused-imports | 1 / 1 | false | true | true |
| regexp-token-scope | regexp/no-dupe-characters-character-class | 1 / 1 | false | true | true |
| e18e-date-now | e18e/prefer-date-now | 1 / 1 | false | true | true |
| test-synthetic-plugin | test/no-only-tests | 1 / 1 | false | true | true |
| react-actual-plugin | react/no-array-index-key | 1 / 1 | false | true | true |
| disable-core | no-console | 1 / 1 | false | true | 未执行 |
| disable-style-original | style/semi | 1 / 1 | false | true | 未执行 |
| disable-style-renamed | style/semi | 1 / 2 | false | false | 未执行 |
| disable-ts-original-native | ts/ban-ts-comment | 1 / 0 | false | false | 未执行 |
| disable-ts-original-js | ts/ban-ts-comment | 1 / 0 | false | false | 未执行 |
| disable-ts-block-native | ts/ban-ts-comment | 0 / 0 | false | true | 未执行 |
| inline-rule-options | no-console | 1 / 1 | false | true | 未执行 |
| invalid-native-options | no-console | ESLint 配置拒绝 | — | — | 未执行 |

## 官方 migrator 的 skipped 清单（基础 profile）

这是工具自己的分类，不代表原型无法补齐；例如 no-restricted-syntax 和合成的 test/no-only-tests 已通过原插件实验。perfectionist/sort-imports 被单独警告并省略，未出现在 skipped 中。

### unsupported

- `no-dupe-args`
- `no-octal`
- `no-octal-escape`
- `no-undef-init`

### not-implemented

- `no-restricted-syntax`
- `node/no-deprecated-api`
- `node/prefer-global/buffer`
- `node/prefer-global/process`
- `node/process-exit-as-throw`
- `jsdoc/check-param-names`
- `jsdoc/check-types`
- `jsdoc/no-multi-asterisks`
- `jsdoc/require-returns-check`
- `jsdoc/require-yields-check`
- `jsdoc/check-alignment`
- `jsdoc/multiline-blocks`
- `vitest/no-only-tests`
