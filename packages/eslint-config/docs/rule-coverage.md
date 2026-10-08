# Research rule coverage

The starting points are `eslint-rules-research.zh-CN.md` (Library `libfile_54b1fabff9dc8191964118f1efc89e81`) and `eslint-rules-exact-catalog.zh-CN.md` (`libfile_25c98595e1148191a893667c0a105b0b`), researched on 2026-10-01. The cloud executor retrieved complete 372-line and 242-line Library reads; byte materialization failed including its single supported retry. The implementation used retained read text rather than assuming another executor’s paths.

All 198 catalog entries have an explicit disposition: 157 enabled in their applicable profiles, 32 JSX accessibility entries deferred by user instruction, four transitional array entries replaced by `ztd/array-layout`, four Compiler entries opt-in, and one fetch entry experimental opt-in. The effective-config tests verify every enabled severity and explicit option against the selected dependency versions. Retained baseline entries are counted, not presented as new rules.

| Dependency | Selected version |
| --- | --- |
| Node / pnpm (cloud validation) | 24.19.0 / 11.19.0 |
| ESLint / TypeScript | 10.11.0 / 6.0.3 |
| @antfu/eslint-config | 9.5.1 |
| @typescript-eslint parser / plugin | 8.71.1 / 8.71.1 |
| @stylistic/eslint-plugin | 5.10.0 |
| @eslint-react/eslint-plugin / react-hooks / react-refresh | 5.23.3 / 7.1.1 / 0.5.7 |
| eslint-plugin-vue / vue-eslint-parser | 10.11.1 / 10.4.1 |
| eslint-plugin-vuejs-accessibility | 2.6.0 |
| eslint-plugin-sonarjs | 4.2.2 |
| @vitest/eslint-plugin | 1.6.27 |

Core/TS extensions, unused-variable owners, React Hooks owners, SFC line-width owners and array layout owners are deduplicated. Handwritten `.d.ts`, configuration and script files retain strict safety and size rules after antfu disables. JS-specific setter diagnostics remain with the TS compiler in TS files. Typed constant-expression checks use `ts/no-unnecessary-condition`; JS keeps `no-constant-binary-expression`. The public factory fixes editor behavior so selected severities do not vary between editor and CI.

Since 0.1.5, the researched unused-variable options apply through native `no-unused-vars` for JS/JSX
and through the existing typed wrapper for TypeScript/TS-parsed Vue. The catalog's historical
`unused-imports/no-unused-vars` entry is checked against the equivalent native JS owner without
changing the 198-entry inventory or its strict options. Application import rejection/autofix remains
enabled, including underscore-prefixed imports; ordinary unused imports may have two valid findings.
Extracted Markdown JS/JSX examples retain Antfu's partial-example unused exceptions. See
[the parser boundary](unused-variables.md).

The implementation does not claim a Vue index-key check: current stable Vue rules ensure a key exists but cannot ensure it is not a loop index. Vue template typing and dynamic component inference also remain outside this package’s static proof boundary.

Version 0.1.1 adds a scoped App Router adapter for the inherited Fast Refresh export rule without changing the 198-entry catalog. [Framework compatibility](framework-compatibility.md) explains the explicit API, scope, upstream contracts and runtime evidence. Antfu also inherits package-manager policy checks outside the research catalog; [pnpm policy](pnpm-policy.md) describes their effects and version requirements.

The frontend foundation policy additionally sets `ts/consistent-type-definitions: ['error', 'type']`
explicitly, replacing the inherited interface preference without changing the historical 198-entry
research inventory. Object contracts compose with type intersections/unions; consistent type-only
imports and all existing typed safety remain active. Import/path aliases are a separate application
source-layout concern.

## shared

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [complexity](https://eslint.org/docs/latest/rules/complexity) | `["error",{"max":10,"variant":"classic"}]` | enabled |
| [sonarjs/cognitive-complexity](https://sonarsource.github.io/rspec/#/rspec/S3776/javascript) | `["error",15]` | enabled |
| [max-depth](https://eslint.org/docs/latest/rules/max-depth) | `["error",{"max":4}]` | enabled |
| [max-lines-per-function](https://eslint.org/docs/latest/rules/max-lines-per-function) | `["error",{"max":60,"skipBlankLines":true,"skipComments":true,"IIFEs":true}]` | enabled |
| [max-lines](https://eslint.org/docs/latest/rules/max-lines) | `["error",{"max":300,"skipBlankLines":true,"skipComments":true}]` | enabled |
| [max-params](https://eslint.org/docs/latest/rules/max-params) | `["error",{"max":4}]` | enabled |
| [style/max-len](https://eslint.style/rules/max-len) | `["error",{"code":120,"comments":120,"tabWidth":2,"ignoreUrls":true,"ignoreStrings":false,"ignoreTemplateLiterals":false,"ignoreRegExpLiterals":false,"ignoreComments":false,"ignoreTrailingComments":false}]` | enabled |
| [eqeqeq](https://eslint.org/docs/latest/rules/eqeqeq) | `["error","always"]` | enabled |
| [curly](https://eslint.org/docs/latest/rules/curly) | `["error","all"]` | enabled |
| [array-callback-return](https://eslint.org/docs/latest/rules/array-callback-return) | `["error",{"allowImplicit":false,"checkForEach":true,"allowVoid":false}]` | enabled |
| [no-empty](https://eslint.org/docs/latest/rules/no-empty) | `["error",{"allowEmptyCatch":false}]` | enabled |
| [no-shadow](https://eslint.org/docs/latest/rules/no-shadow) | `["error",{"builtinGlobals":false,"hoist":"all","ignoreOnInitialization":false}]` | enabled |
| [no-nested-ternary](https://eslint.org/docs/latest/rules/no-nested-ternary) | `"error"` | enabled |
| [no-unused-expressions](https://eslint.org/docs/latest/rules/no-unused-expressions) | `["error",{"allowShortCircuit":false,"allowTernary":false,"allowTaggedTemplates":true,"enforceForJSX":true}]` | enabled |
| [eslint-comments/require-description](https://eslint-community.github.io/eslint-plugin-eslint-comments/rules/require-description.html) | `["error",{"ignore":["eslint-enable"]}]` | enabled |
| [unused-imports/no-unused-vars](https://github.com/sweepline/eslint-plugin-unused-imports/blob/master/docs/rules/no-unused-vars.md) | `["error",{"args":"all","argsIgnorePattern":"^_","caughtErrors":"all","caughtErrorsIgnorePattern":"^_","vars":"all","varsIgnorePattern":"^_","ignoreRestSiblings":true,"reportUsedIgnorePattern":true}]` | enabled |
| [style/array-bracket-newline](https://eslint.style/rules/array-bracket-newline) | `["error",{"multiline":true}]` | replaced: ztd/array-layout (full researched policy) |
| [style/array-element-newline](https://eslint.style/rules/array-element-newline) | `["error",{"ArrayExpression":{"minItems":3,"multiline":true,"consistent":true},"ArrayPattern":"consistent"}]` | replaced: ztd/array-layout (full researched policy) |
| [no-constant-binary-expression](https://eslint.org/docs/latest/rules/no-constant-binary-expression) | `"error"` | enabled |
| [no-unsafe-optional-chaining](https://eslint.org/docs/latest/rules/no-unsafe-optional-chaining) | `["error",{"disallowArithmeticOperators":true}]` | enabled |
| [no-promise-executor-return](https://eslint.org/docs/latest/rules/no-promise-executor-return) | `["error",{"allowVoid":false}]` | enabled |
| [no-setter-return](https://eslint.org/docs/latest/rules/no-setter-return) | `"error"` | enabled |
## ts-syntax

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [ts/no-explicit-any](https://typescript-eslint.io/rules/no-explicit-any) | `["error",{"fixToUnknown":false,"ignoreRestArgs":false}]` | enabled |
| [ts/no-non-null-assertion](https://typescript-eslint.io/rules/no-non-null-assertion) | `"error"` | enabled |
| [ts/no-empty-object-type](https://typescript-eslint.io/rules/no-empty-object-type) | `["error",{"allowInterfaces":"never","allowObjectTypes":"never"}]` | enabled |
| [ts/ban-ts-comment](https://typescript-eslint.io/rules/ban-ts-comment) | `["error",{"ts-ignore":true,"ts-nocheck":true,"ts-check":false,"ts-expect-error":"allow-with-description","minimumDescriptionLength":10}]` | enabled |
| [ts/consistent-type-assertions](https://typescript-eslint.io/rules/consistent-type-assertions) | `["error",{"assertionStyle":"as","objectLiteralTypeAssertions":"never","arrayLiteralTypeAssertions":"never"}]` | enabled |
| [ts/explicit-module-boundary-types](https://typescript-eslint.io/rules/explicit-module-boundary-types) | `["error",{"allowArgumentsExplicitlyTypedAsAny":false,"allowDirectConstAssertionInArrowFunctions":false,"allowHigherOrderFunctions":false,"allowOverloadFunctions":false,"allowTypedFunctionExpressions":true}]` | enabled |
| [ts/no-useless-constructor](https://typescript-eslint.io/rules/no-useless-constructor) | `"error"` | enabled |
| [ts/no-invalid-void-type](https://typescript-eslint.io/rules/no-invalid-void-type) | `["error",{"allowAsThisParameter":true,"allowInGenericTypeArguments":true}]` | enabled |
| [ts/no-extraneous-class](https://typescript-eslint.io/rules/no-extraneous-class) | `["error",{"allowConstructorOnly":false,"allowEmpty":false,"allowStaticOnly":false,"allowWithDecorator":true}]` | enabled |
| [ts/max-params](https://typescript-eslint.io/rules/max-params) | `["error",{"max":4,"countVoidThis":false}]` | enabled |
| [ts/no-shadow](https://typescript-eslint.io/rules/no-shadow) | `["error",{"builtinGlobals":false,"hoist":"functions-and-types","ignoreOnInitialization":false,"ignoreTypeValueShadow":true,"ignoreFunctionTypeParameterNameValueShadow":true}]` | enabled |
| [ts/no-unused-expressions](https://typescript-eslint.io/rules/no-unused-expressions) | `["error",{"allowShortCircuit":false,"allowTernary":false,"allowTaggedTemplates":true,"enforceForJSX":true}]` | enabled |
## typed

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [ts/strict-boolean-expressions](https://typescript-eslint.io/rules/strict-boolean-expressions) | `["error",{"allowAny":false,"allowNumber":false,"allowString":false,"allowNullableBoolean":false,"allowNullableEnum":false,"allowNullableNumber":false,"allowNullableObject":false,"allowNullableString":false}]` | enabled |
| [ts/no-floating-promises](https://typescript-eslint.io/rules/no-floating-promises) | `["error",{"checkThenables":true,"ignoreVoid":false,"ignoreIIFE":false,"allowForKnownSafeCalls":[],"allowForKnownSafePromises":[]}]` | enabled |
| [ts/no-misused-promises](https://typescript-eslint.io/rules/no-misused-promises) | `["error",{"checksConditionals":{"flagUnions":"all"},"checksSpreads":true,"checksVoidReturn":true}]` | enabled |
| [ts/restrict-plus-operands](https://typescript-eslint.io/rules/restrict-plus-operands) | `["error",{"allowAny":false,"allowBoolean":false,"allowNullish":false,"allowNumberAndString":false,"allowRegExp":false,"skipCompoundAssignments":false}]` | enabled |
| [ts/restrict-template-expressions](https://typescript-eslint.io/rules/restrict-template-expressions) | `["error",{"allow":[],"allowAny":false,"allowArray":false,"allowBoolean":false,"allowNullish":false,"allowNumber":true,"allowRegExp":false,"allowNever":false}]` | enabled |
| [ts/return-await](https://typescript-eslint.io/rules/return-await) | `["error","always"]` | enabled |
| [ts/switch-exhaustiveness-check](https://typescript-eslint.io/rules/switch-exhaustiveness-check) | `["error",{"allowDefaultCaseForExhaustiveSwitch":true,"considerDefaultExhaustiveForUnions":false,"requireDefaultForNonUnion":true}]` | enabled |
| [ts/only-throw-error](https://typescript-eslint.io/rules/only-throw-error) | `["error",{"allow":[],"allowRethrowing":true,"allowThrowingAny":false,"allowThrowingUnknown":false}]` | enabled |
| [ts/no-unsafe-type-assertion](https://typescript-eslint.io/rules/no-unsafe-type-assertion) | `"error"` | enabled |
| [ts/use-unknown-in-catch-callback-variable](https://typescript-eslint.io/rules/use-unknown-in-catch-callback-variable) | `"error"` | enabled |
| [ts/no-unnecessary-condition](https://typescript-eslint.io/rules/no-unnecessary-condition) | `["error",{"allowConstantLoopConditions":"only-allowed-literals","checkTypePredicates":true}]` | enabled |
| [ts/no-base-to-string](https://typescript-eslint.io/rules/no-base-to-string) | `["error",{"checkUnknown":true,"ignoredTypeNames":["Error","RegExp","URL","URLSearchParams"]}]` | enabled |
| [ts/no-misused-spread](https://typescript-eslint.io/rules/no-misused-spread) | `["error",{"allow":[]}]` | enabled |
| [ts/no-unsafe-enum-comparison](https://typescript-eslint.io/rules/no-unsafe-enum-comparison) | `"error"` | enabled |
| [ts/no-deprecated](https://typescript-eslint.io/rules/no-deprecated) | `["error",{"allow":[]}]` | enabled |
| [ts/consistent-type-exports](https://typescript-eslint.io/rules/consistent-type-exports) | `["error",{"fixMixedExportsWithInlineTypeSpecifier":false}]` | enabled |
| [ts/prefer-readonly](https://typescript-eslint.io/rules/prefer-readonly) | `["error",{"onlyInlineLambdas":false}]` | enabled |
| [ts/require-array-sort-compare](https://typescript-eslint.io/rules/require-array-sort-compare) | `["error",{"ignoreStringArrays":true}]` | enabled |
| [ts/require-await](https://typescript-eslint.io/rules/require-await) | `"error"` | enabled |
| [ts/prefer-nullish-coalescing](https://typescript-eslint.io/rules/prefer-nullish-coalescing) | `["error",{"ignoreConditionalTests":true,"ignoreMixedLogicalExpressions":false,"ignorePrimitives":{"bigint":false,"boolean":false,"number":false,"string":false}}]` | enabled |
| [ts/prefer-optional-chain](https://typescript-eslint.io/rules/prefer-optional-chain) | `["error",{"requireNullish":true,"allowPotentiallyUnsafeFixesThatModifyTheReturnTypeIKnowWhatImDoing":false}]` | enabled |
| [ts/no-array-delete](https://typescript-eslint.io/rules/no-array-delete) | `"error"` | enabled |
| [ts/no-unsafe-unary-minus](https://typescript-eslint.io/rules/no-unsafe-unary-minus) | `"error"` | enabled |
| [ts/prefer-promise-reject-errors](https://typescript-eslint.io/rules/prefer-promise-reject-errors) | `["error",{"allowEmptyReject":false,"allowThrowingAny":false,"allowThrowingUnknown":false}]` | enabled |
| [ts/await-thenable](https://typescript-eslint.io/rules/await-thenable) | `"error"` | enabled |
| [ts/no-for-in-array](https://typescript-eslint.io/rules/no-for-in-array) | `"error"` | enabled |
| [ts/no-implied-eval](https://typescript-eslint.io/rules/no-implied-eval) | `"error"` | enabled |
| [ts/no-unnecessary-type-assertion](https://typescript-eslint.io/rules/no-unnecessary-type-assertion) | `"error"` | enabled |
| [ts/no-unsafe-argument](https://typescript-eslint.io/rules/no-unsafe-argument) | `"error"` | enabled |
| [ts/no-unsafe-assignment](https://typescript-eslint.io/rules/no-unsafe-assignment) | `"error"` | enabled |
| [ts/no-unsafe-call](https://typescript-eslint.io/rules/no-unsafe-call) | `"error"` | enabled |
| [ts/no-unsafe-member-access](https://typescript-eslint.io/rules/no-unsafe-member-access) | `"error"` | enabled |
| [ts/no-unsafe-return](https://typescript-eslint.io/rules/no-unsafe-return) | `"error"` | enabled |
| [ts/unbound-method](https://typescript-eslint.io/rules/unbound-method) | `"error"` | enabled |
## react

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [react/exhaustive-deps](https://eslint-react.xyz/docs/rules/exhaustive-deps) | `["error",{"enableDangerousAutofixThisMayCauseInfiniteLoops":false,"requireExplicitEffectDeps":false}]` | enabled |
| [react/no-array-index-key](https://eslint-react.xyz/docs/rules/no-array-index-key) | `"error"` | enabled |
| [react/purity](https://eslint-react.xyz/docs/rules/purity) | `"error"` | enabled |
| [react/set-state-in-effect](https://eslint-react.xyz/docs/rules/set-state-in-effect) | `"error"` | enabled |
| [react/web-api-no-leaked-event-listener](https://eslint-react.xyz/docs/rules/web-api-no-leaked-event-listener) | `"error"` | enabled |
| [react/web-api-no-leaked-intersection-observer](https://eslint-react.xyz/docs/rules/web-api-no-leaked-intersection-observer) | `"error"` | enabled |
| [react/web-api-no-leaked-interval](https://eslint-react.xyz/docs/rules/web-api-no-leaked-interval) | `"error"` | enabled |
| [react/web-api-no-leaked-resize-observer](https://eslint-react.xyz/docs/rules/web-api-no-leaked-resize-observer) | `"error"` | enabled |
| [react/web-api-no-leaked-timeout](https://eslint-react.xyz/docs/rules/web-api-no-leaked-timeout) | `"error"` | enabled |
| [react/jsx-no-comment-textnodes](https://eslint-react.xyz/docs/rules/jsx-no-comment-textnodes) | `"error"` | enabled |
| [react/jsx-no-leaked-dollar](https://eslint-react.xyz/docs/rules/jsx-no-leaked-dollar) | `"error"` | enabled |
| [react/jsx-no-leaked-semicolon](https://eslint-react.xyz/docs/rules/jsx-no-leaked-semicolon) | `"error"` | enabled |
| [react/dom-no-dangerously-set-innerhtml](https://eslint-react.xyz/docs/rules/dom-no-dangerously-set-innerhtml) | `"error"` | enabled |
| [react/dom-no-script-url](https://eslint-react.xyz/docs/rules/dom-no-script-url) | `"error"` | enabled |
| [react/dom-no-unsafe-iframe-sandbox](https://eslint-react.xyz/docs/rules/dom-no-unsafe-iframe-sandbox) | `"error"` | enabled |
| [react/web-api-no-leaked-fetch](https://eslint-react.xyz/docs/rules/web-api-no-leaked-fetch) | `"error"` | opt-in: react.experimental |
| [react/use-state](https://eslint-react.xyz/docs/rules/use-state) | `["error",{"enforceAssignment":true,"enforceLazyInitialization":true,"enforceSetterName":true}]` | enabled |
| [react/no-unstable-context-value](https://eslint-react.xyz/docs/rules/no-unstable-context-value) | `"error"` | enabled |
| [react/no-unstable-default-props](https://eslint-react.xyz/docs/rules/no-unstable-default-props) | `["error",{"safeDefaultProps":[]}]` | enabled |
| [react/dom-no-missing-button-type](https://eslint-react.xyz/docs/rules/dom-no-missing-button-type) | `"error"` | enabled |
| [react/dom-no-missing-iframe-sandbox](https://eslint-react.xyz/docs/rules/dom-no-missing-iframe-sandbox) | `"error"` | enabled |
| [react/dom-no-unsafe-target-blank](https://eslint-react.xyz/docs/rules/dom-no-unsafe-target-blank) | `"error"` | enabled |
| [react/rules-of-hooks](https://eslint-react.xyz/docs/rules/rules-of-hooks) | `"error"` | enabled |
| [react/error-boundaries](https://eslint-react.xyz/docs/rules/error-boundaries) | `"error"` | enabled |
| [react/no-missing-key](https://eslint-react.xyz/docs/rules/no-missing-key) | `"error"` | enabled |
| [react/no-nested-component-definitions](https://eslint-react.xyz/docs/rules/no-nested-component-definitions) | `"error"` | enabled |
| [react/no-nested-lazy-component-declarations](https://eslint-react.xyz/docs/rules/no-nested-lazy-component-declarations) | `"error"` | enabled |
| [react/set-state-in-render](https://eslint-react.xyz/docs/rules/set-state-in-render) | `"error"` | enabled |
| [react/static-components](https://eslint-react.xyz/docs/rules/static-components) | `"error"` | enabled |
| [react/use-memo](https://eslint-react.xyz/docs/rules/use-memo) | `"error"` | enabled |
| [react/dom-no-dangerously-set-innerhtml-with-children](https://eslint-react.xyz/docs/rules/dom-no-dangerously-set-innerhtml-with-children) | `"error"` | enabled |
| [react/dom-no-void-elements-with-children](https://eslint-react.xyz/docs/rules/dom-no-void-elements-with-children) | `"error"` | enabled |
| [react/no-leaked-conditional-rendering](https://eslint-react.xyz/docs/rules/no-leaked-conditional-rendering) | `"error"` | enabled: typed React files only |
| [react-hooks/globals](https://react.dev/reference/eslint-plugin-react-hooks/lints/globals) | `"error"` | enabled |
| [react-hooks/immutability](https://react.dev/reference/eslint-plugin-react-hooks/lints/immutability) | `"error"` | enabled |
| [react-hooks/refs](https://react.dev/reference/eslint-plugin-react-hooks/lints/refs) | `"error"` | enabled |
| [react-hooks/config](https://react.dev/reference/eslint-plugin-react-hooks/lints/config) | `"error"` | opt-in: react.compiler |
| [react-hooks/gating](https://react.dev/reference/eslint-plugin-react-hooks/lints/gating) | `"error"` | opt-in: react.compiler |
| [react-hooks/incompatible-library](https://react.dev/reference/eslint-plugin-react-hooks/lints/incompatible-library) | `"error"` | opt-in: react.compiler |
| [react-hooks/preserve-manual-memoization](https://react.dev/reference/eslint-plugin-react-hooks/lints/preserve-manual-memoization) | `"error"` | opt-in: react.compiler |
## react-a11y

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [jsx-a11y/alt-text](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/alt-text.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/anchor-has-content](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/anchor-has-content.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/anchor-is-valid](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/anchor-is-valid.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/aria-activedescendant-has-tabindex](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/aria-activedescendant-has-tabindex.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/aria-props](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/aria-props.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/aria-proptypes](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/aria-proptypes.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/aria-role](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/aria-role.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/aria-unsupported-elements](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/aria-unsupported-elements.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/autocomplete-valid](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/autocomplete-valid.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/click-events-have-key-events](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/click-events-have-key-events.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/heading-has-content](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/heading-has-content.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/html-has-lang](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/html-has-lang.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/iframe-has-title](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/iframe-has-title.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/img-redundant-alt](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/img-redundant-alt.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/interactive-supports-focus](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/interactive-supports-focus.md) | `["error",{"tabbable":["button","checkbox","link","progressbar","searchbox","slider","spinbutton","switch","textbox"]}]` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/label-has-associated-control](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/blob/main/docs/rules/label-has-associated-control.md) | `["error",{"assert":"either","depth":3}]` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/media-has-caption](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/media-has-caption.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/mouse-events-have-key-events](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/mouse-events-have-key-events.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-access-key](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-access-key.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-autofocus](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-autofocus.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-distracting-elements](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-distracting-elements.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-interactive-element-to-noninteractive-role](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-interactive-element-to-noninteractive-role.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-noninteractive-element-interactions](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-noninteractive-element-interactions.md) | `["error",{"body":["onError","onLoad"],"iframe":["onError","onLoad"],"img":["onError","onLoad"]}]` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-noninteractive-element-to-interactive-role](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-noninteractive-element-to-interactive-role.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-noninteractive-tabindex](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-noninteractive-tabindex.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-redundant-roles](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-redundant-roles.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/no-static-element-interactions](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/no-static-element-interactions.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/role-has-required-aria-props](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/role-has-required-aria-props.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/role-supports-aria-props](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/role-supports-aria-props.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/scope](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/scope.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/tabindex-no-positive](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/tree/HEAD/docs/rules/tabindex-no-positive.md) | `"error"` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
| [jsx-a11y/control-has-associated-label](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/blob/main/docs/rules/control-has-associated-label.md) | `["error",{"depth":3,"ignoreElements":["input","select","textarea"]}]` | disabled: JSX a11y ESLint 10 peer incompatibility (user instruction) |
## vue

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [vue/no-dupe-keys](https://eslint.vuejs.org/rules/no-dupe-keys.html) | `"error"` | enabled |
| [vue/no-v-html](https://eslint.vuejs.org/rules/no-v-html.html) | `"error"` | enabled |
| [vue/no-setup-props-reactivity-loss](https://eslint.vuejs.org/rules/no-setup-props-reactivity-loss.html) | `"error"` | enabled |
| [vue/no-ref-object-reactivity-loss](https://eslint.vuejs.org/rules/no-ref-object-reactivity-loss.html) | `"error"` | enabled |
| [vue/no-mutating-props](https://eslint.vuejs.org/rules/no-mutating-props.html) | `["error",{"shallowOnly":false}]` | enabled |
| [vue/require-explicit-emits](https://eslint.vuejs.org/rules/require-explicit-emits.html) | `["error",{"allowProps":false}]` | enabled |
| [vue/require-explicit-slots](https://eslint.vuejs.org/rules/require-explicit-slots.html) | `"error"` | enabled |
| [vue/require-prop-types](https://eslint.vuejs.org/rules/require-prop-types.html) | `"error"` | enabled |
| [vue/no-undef-components](https://eslint.vuejs.org/rules/no-undef-components.html) | `["error",{"ignorePatterns":[]}]` | enabled |
| [vue/no-undef-directives](https://eslint.vuejs.org/rules/no-undef-directives.html) | `["error",{"ignore":[]}]` | enabled |
| [vue/no-unused-emit-declarations](https://eslint.vuejs.org/rules/no-unused-emit-declarations.html) | `"error"` | enabled |
| [vue/html-button-has-type](https://eslint.vuejs.org/rules/html-button-has-type.html) | `["error",{"button":true,"submit":true,"reset":true}]` | enabled |
| [vue/no-template-target-blank](https://eslint.vuejs.org/rules/no-template-target-blank.html) | `["error",{"allowReferrer":false,"enforceDynamicLinks":"always"}]` | enabled |
| [vue/eqeqeq](https://eslint.vuejs.org/rules/eqeqeq.html) | `["error","always"]` | enabled |
| [vue/max-lines-per-block](https://eslint.vuejs.org/rules/max-lines-per-block.html) | `["error",{"script":200,"template":200,"style":150,"skipBlankLines":true}]` | enabled |
| [vue/max-template-depth](https://eslint.vuejs.org/rules/max-template-depth.html) | `["error",{"maxDepth":5}]` | enabled |
| [vue/max-len](https://eslint.vuejs.org/rules/max-len.html) | `["error",{"code":120,"template":120,"comments":120,"tabWidth":2,"ignoreUrls":true,"ignoreStrings":false,"ignoreTemplateLiterals":false,"ignoreRegExpLiterals":false,"ignoreHTMLAttributeValues":false,"ignoreHTMLTextContents":false}]` | enabled |
| [vue/array-bracket-newline](https://eslint.vuejs.org/rules/array-bracket-newline.html) | `["error",{"multiline":true}]` | replaced: ztd/array-layout (full researched policy) |
| [vue/array-element-newline](https://eslint.vuejs.org/rules/array-element-newline.html) | `["error",{"ArrayExpression":{"minItems":3,"multiline":true,"consistent":true},"ArrayPattern":"consistent"}]` | replaced: ztd/array-layout (full researched policy) |
| [vue/no-template-shadow](https://eslint.vuejs.org/rules/no-template-shadow.html) | `"error"` | enabled |
| [vue/one-component-per-file](https://eslint.vuejs.org/rules/one-component-per-file.html) | `"error"` | enabled |
| [vue/no-required-prop-with-default](https://eslint.vuejs.org/rules/no-required-prop-with-default.html) | `"error"` | enabled |
| [vue/html-end-tags](https://eslint.vuejs.org/rules/html-end-tags.html) | `"error"` | enabled |
| [vue/no-async-in-computed-properties](https://eslint.vuejs.org/rules/no-async-in-computed-properties.html) | `"error"` | enabled |
| [vue/no-side-effects-in-computed-properties](https://eslint.vuejs.org/rules/no-side-effects-in-computed-properties.html) | `"error"` | enabled |
| [vue/no-use-v-if-with-v-for](https://eslint.vuejs.org/rules/no-use-v-if-with-v-for.html) | `"error"` | enabled |
| [vue/require-v-for-key](https://eslint.vuejs.org/rules/require-v-for-key.html) | `"error"` | enabled |
| [vue/no-watch-after-await](https://eslint.vuejs.org/rules/no-watch-after-await.html) | `"error"` | enabled |
| [vue/no-lifecycle-after-await](https://eslint.vuejs.org/rules/no-lifecycle-after-await.html) | `"error"` | enabled |
| [vue/no-expose-after-await](https://eslint.vuejs.org/rules/no-expose-after-await.html) | `"error"` | enabled |
## vue-a11y

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [vue-a11y/alt-text](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/alt-text.html) | `"error"` | enabled |
| [vue-a11y/anchor-has-content](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/anchor-has-content.html) | `"error"` | enabled |
| [vue-a11y/aria-props](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/aria-props.html) | `"error"` | enabled |
| [vue-a11y/aria-role](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/aria-role.html) | `"error"` | enabled |
| [vue-a11y/aria-unsupported-elements](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/aria-unsupported-elements.html) | `"error"` | enabled |
| [vue-a11y/click-events-have-key-events](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/click-events-have-key-events.html) | `"error"` | enabled |
| [vue-a11y/form-control-has-label](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/form-control-has-label.html) | `"error"` | enabled |
| [vue-a11y/heading-has-content](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/heading-has-content.html) | `"error"` | enabled |
| [vue-a11y/iframe-has-title](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/iframe-has-title.html) | `"error"` | enabled |
| [vue-a11y/interactive-supports-focus](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/interactive-supports-focus.html) | `"error"` | enabled |
| [vue-a11y/label-has-for](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/label-has-for.html) | `["error",{"required":{"some":["nesting","id"]},"allowChildren":false}]` | enabled |
| [vue-a11y/media-has-caption](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/media-has-caption.html) | `"error"` | enabled |
| [vue-a11y/mouse-events-have-key-events](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/mouse-events-have-key-events.html) | `"error"` | enabled |
| [vue-a11y/no-access-key](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-access-key.html) | `"error"` | enabled |
| [vue-a11y/no-aria-hidden-on-focusable](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-aria-hidden-on-focusable.html) | `"error"` | enabled |
| [vue-a11y/no-autofocus](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-autofocus.html) | `"error"` | enabled |
| [vue-a11y/no-distracting-elements](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-distracting-elements.html) | `"error"` | enabled |
| [vue-a11y/no-redundant-roles](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-redundant-roles.html) | `"error"` | enabled |
| [vue-a11y/no-role-presentation-on-focusable](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-role-presentation-on-focusable.html) | `"error"` | enabled |
| [vue-a11y/no-static-element-interactions](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/no-static-element-interactions.html) | `"error"` | enabled |
| [vue-a11y/role-has-required-aria-props](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/role-has-required-aria-props.html) | `"error"` | enabled |
| [vue-a11y/tabindex-no-positive](https://vue-a11y.github.io/eslint-plugin-vuejs-accessibility/rules/tabindex-no-positive.html) | `"error"` | enabled |
## tests

| Rule | Exact researched value | Disposition |
| --- | --- | --- |
| [test/valid-expect](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/valid-expect.md) | `["error",{"alwaysAwait":true,"minArgs":1,"maxArgs":1}]` | enabled |
| [test/no-conditional-expect](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/no-conditional-expect.md) | `["error",{"expectAssertions":false}]` | enabled |
| [test/no-conditional-tests](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/no-conditional-tests.md) | `"error"` | enabled |
| [test/no-disabled-tests](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/no-disabled-tests.md) | `"error"` | enabled |
| [test/expect-expect](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/expect-expect.md) | `["error",{"assertFunctionNames":["expect","assert"],"additionalTestBlockFunctions":[]}]` | enabled |
| [test/unbound-method](https://github.com/vitest-dev/eslint-plugin-vitest/blob/main/docs/rules/unbound-method.md) | `["error",{"ignoreStatic":false}]` | enabled |
