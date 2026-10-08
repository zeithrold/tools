# Unused-variable parser boundary

Version 0.1.5 assigns JavaScript value-use analysis to ESLint's native `no-unused-vars`, retaining
all strict options: all variables, arguments and caught errors, the explicit `^_` exemptions, rest
siblings and errors for used ignored names. TypeScript/TSX and TS-parsed Vue retain the existing
`unused-imports/no-unused-vars` typed implementation. Vue with TypeScript disabled uses the native
JS rule. No parser, scope/reference objects, upstream package files or consumer rules are patched.

The author lock exercises TypeScript ESLint 8.71.1. Its `isTypeOnlyReference` helper assumes a
TypeScript `isValueReference` flag for variable definitions. Espree references do not expose that
flag, so the typed rule used by unused-imports can classify a runtime JS read as type-only. The
minimal failing example is `const number = 1; console.log(number)`. Separating rule owners by the
native parser scope fixes that mismatch without downgrading the dependency or replacing its code.
TypeScript values genuinely used only for types remain rejected, as documented by the upstream
[TypeScript rule](https://typescript-eslint.io/rules/no-unused-vars/).

Application JS retains `unused-imports/no-unused-imports`, including its autofix and rejection of
unused underscore-prefixed imports. Core JS analysis also sees import bindings, so an ordinary
unused import can have both a native and import-checker finding. This duplication preserves import
rejection rather than accepting an ignored import. The native used-ignore-name option applies to
imports as well as local variables. TypeScript owners and settings remain unchanged.

## Markdown examples

Antfu 9.5.1's installed `dist/index.mjs`, under `antfu/markdown/disables/code`, explicitly turns off
`no-unused-vars`, `ts/no-unused-vars`, `unused-imports/no-unused-vars` and
`unused-imports/no-unused-imports` for extracted code fragments. These may contain illustrative
imports or variables without the rest of an application. The JS/JSX fragment scope preserves that
native partial-example policy while leaving syntax, React and array checks active. It does not
apply to actual JS application files or change the existing TS/TSX example scope.

Regression tests pair allowed incomplete JS/JSX examples with rejected unused real application
bindings, and cover JS/JSX suffixes, named/default/namespace imports, parameters, catches, ignored
names, Vue template references, typed Vue and type-only use. Packed and registry smoke checks repeat
value-use and deliberate unused checks against the actual installed official npm package. Release-age,
trust, integrity and stage-only owner promotion controls remain active.
