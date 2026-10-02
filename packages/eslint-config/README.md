# @ztd-me/eslint

Strict, typed ESLint flat configs built on `@antfu/eslint-config`. The standard is defined independently of existing projects. Every selected research rule is an error; existing antfu opinions keep their upstream severities.

```sh
pnpm add -D @ztd-me/eslint eslint@^10.11.0 typescript@~6.0.3
```

Requires Node >=22.14, ESLint 10.11 or later in major 10, and TypeScript >=5.4 <6.1. The tested versions and all 198 research entries are documented in [rule coverage](docs/rule-coverage.md) and the [machine-readable catalog](docs/rule-catalog.json).

```js
// eslint.config.js (ESM)
import ztd from '@ztd-me/eslint'

export default ztd({
  typescript: { tsconfigPath: 'tsconfig.json' },
  react: true,
  ignores: ['dist/**'],
})
```

The default export and named `createConfig` are the same asynchronous function and return a Promise of flat configs. The declarations export `ConfigOptions`, `TypeScriptOptions`, `ReactOptions`, `VueOptions`, and `LocalConfig`. This API accepts explicit final overrides:

```js
import { createConfig } from '@ztd-me/eslint'

export default createConfig({
  vue: true,
  typescript: {
    tsconfigPath: 'tsconfig.json',
    tsconfigRootDir: import.meta.dirname,
  },
}, {
  name: 'project/cli-output',
  files: ['cli/**'],
  rules: { 'no-console': 'off' },
})
```

Framework dependencies used by this package are included: enabling a supported profile does not prompt to install a plugin. React and Vue default to `false`; enable only the frameworks used by the project. Both can be enabled with independent `files` globs in a mixed repository. Vue targets Vue 3; Vue 2 is outside this profile.

## TypeScript

Type-aware checking is on by default, using `tsconfig.json`. Supply an existing project config with `strict: true` (including `strictNullChecks`) and `noUncheckedIndexedAccess: true`. Inherited compiler options are resolved; missing configs and unsafe compiler options fail explicitly. The selected TS config must include every linted TS/TSX file, including handwritten configuration and test files. For solution-style project references, point to the actual linted project's config.

The parser uses the explicit TS project, with `.vue` as an extra file extension and `vue-eslint-parser` as the SFC outer parser. Typed rules cover Vue scripts as well as TS and TSX. Vue template expressions use Vue rules and the custom array visitor; TypeScript rules do not type-check template expressions.

For a JavaScript-only scope, use `typescript: false`. This turns off TypeScript parsing and rules; it does not lower JS complexity, length, or correctness standards. There is no syntax-only TypeScript or `strictTypes: false` mode. Run `tsc --noEmit` as a separate compiler gate; `exactOptionalPropertyTypes`, `noPropertyAccessFromIndexSignature`, and `useUnknownInCatchVariables` are also recommended compiler settings.

## Framework options

| Option | Default | Behavior |
| --- | --- | --- |
| `react` | `false` | `true`, or `{ files, compiler, experimental }`; includes hooks in `.ts` as well as JSX |
| `react.compiler` | `false` | Adds four official React Compiler checks |
| `react.experimental` | `false` | Adds experimental fetch cleanup checking |
| `vue` | `false` | `true`, or `{ files }`; Vue 3 correctness and accessibility are strict |
| `test` | `false` | Explicitly enable the researched Vitest profile; do not enable for Node test or Jest |
| `rules` | `{}` | Deliberate final overrides; subsequent flat configs run last |

JSX accessibility is disabled and `eslint-plugin-jsx-a11y` is not installed: its 6.10.2 peer range excludes ESLint 10. This is the user-authorized compatibility exception, not an accessibility certification. Vue accessibility remains enabled. JSX syntax and React correctness remain enabled when React is selected.

The selected React Hooks checks use one owner per behavior: `react/*` for hooks and effects, `react-hooks/*` only for globals, immutability, refs, and optional Compiler checks. The experimental fetch rule is off until explicitly selected. React version migration opinions are inherited from antfu rather than strengthened as correctness rules.

## Limits and arrays

| Measure | Inclusive maximum / threshold |
| --- | --- |
| Classic cyclomatic complexity | 10 |
| Cognitive complexity | 15 |
| Control-flow nesting | 4 |
| Function length | 60 effective lines, including IIFEs and nested function text |
| File length | 300 effective lines; SFCs include script, template and style |
| Code and comments per line | 120 characters, tab width 2 |
| Function parameters | 4; TS `this: void` is not counted |
| Vue block lines | script 200, template 200, style 150; blank lines excluded, comments counted |
| Vue template nesting | 5 |
| Array literals | 3 or more elements expand, including exactly 3 |

These are researched defaults, not empirically optimal constants. Function/file counts exclude blank lines and comments. Line-width rules only exempt lines containing URLs; strings, regexes, templates and comments remain checked. Width rules report violations without unsafe automatic wrapping.

`ztd/array-layout` implements the complete array policy in JS, TS, JSX, Vue scripts and Vue templates:

- Empty arrays and simple arrays of 1–2 elements use one line when the complete surrounding line fits 120 characters.
- Arrays with at least 3 elements, complex/nested elements, multiline elements, or over-wide simple pairs expand: bracket boundaries and each element have separate lines. Antfu supplies indentation and multiline trailing commas.
- Simple elements are primitives, identifiers, unary operations on primitives, simple noncomputed property chains, and their TS/chain wrappers. Objects, arrays, calls, functions, spreads, and conditional expressions are complex.
- Comments retain their association. A fix that would cross a comment is withheld for manual editing; comments prevent short-array collapsing. CRLF is preserved.
- Spreads count as one syntax element; holes remain prohibited by sparse-array rules. Destructuring and tuple types use their existing consistency policy rather than the literal threshold. Tuple values and `as const` arrays use the literal policy.

```ts
const pair = [first, second]
const ids = [
  first,
  second,
  third,
]
const users = [
  { id: 1 },
  { id: 2 },
]
const [red, green, blue] = rgb
```

The custom rule replaces the four transitional array rules from the research. Antfu's ArrayExpression consistency visitor is disabled; object/import/parameter consistency stays enabled. Stable stylistic formatting stays active. Disabling it or enabling its experimental list formatter is rejected; explicit final rule overrides remain available.

## Validation and release

```sh
pnpm install --frozen-lockfile
pnpm run check
```

The checks build JS/declarations, type-check the public API, self-lint source/tests/scripts, exercise real TS/React/Vue fixtures, check all numeric boundaries and all selected catalog options, and install/import/type-check an independently packed artifact with pnpm. Negative fixtures are checked by the test harness, not included in ordinary self-lint. The test modules have reasoned local exceptions for Node's asynchronous ESM setup and imports of the distribution being tested; safety and size limits remain active.

[Publishing instructions](docs/publishing.md) describe the stage-only GitHub workflow and the user-controlled promotion. Staging is not a public release. After promotion, run `node scripts/registry-smoke.mjs 0.1.0` to install the exact public version in a fresh pnpm project and verify imports, declarations and lint behavior. Registry installation follows pnpm's supply-chain policies; a policy rejection is a blocker, not permission to disable the policy.

The repository has not granted an open-source license; package metadata is `UNLICENSED` pending that separate decision.
