# `@zeithrold/eslint-config`

Draft flat ESLint config package. It composes `@antfu/eslint-config`; each project keeps its own framework options, ignores, and final overrides. This package contains no Workbench-specific i18n rule.

```js
import zeithrold from '@zeithrold/eslint-config'

export default zeithrold({
  react: true,
  typescript: { tsconfigPath: 'tsconfig.json' },
  strictTypes: true,
  ignores: ['dist/**'],
}, {
  files: ['app/**/*.tsx'],
  rules: {
    'react-refresh/only-export-components': 'off',
  },
})
```

Install this package, ESLint, and `@antfu/eslint-config` in the consuming project with its own package manager and lockfile. The package remains private until tested against both memory and another JS/TS frontend.
