import config, { createConfig, type ConfigOptions } from '../../dist/index.js'
import type { Linter } from 'eslint'

const options: ConfigOptions = {
  typescript: { tsconfigPath: 'tsconfig.json', tsconfigRootDir: '/project' },
  react: { files: ['react/**/*.tsx'], compiler: true, experimental: false },
  vue: { files: ['vue/**/*.vue'] },
  test: true,
  ignores: ['dist/**'],
  rules: { complexity: ['error', { max: 8 }] },
}
const pending: Promise<Linter.Config[]> = config(options, { files: ['cli/**'], rules: { 'no-console': 'off' } })
void pending
void createConfig({ typescript: false })
void config({ react: { framework: 'vinext', appDir: 'apps/web/app' } })
void config({ react: { framework: 'next' } })
// @ts-expect-error -- Only the explicitly supported App Router integrations are accepted.
void config({ react: { framework: 'remix' } })
// @ts-expect-error -- JSX a11y is deliberately absent from this public API while incompatible with ESLint 10.
void config({ jsx: { a11y: true } })
// @ts-expect-error -- TypeScript cannot be silently downgraded to a syntax-only strictTypes:false profile.
void config({ strictTypes: false })
// @ts-expect-error -- Vue 2 semantics are outside the supported Vue 3 profile.
void config({ vue: { vueVersion: 2 } })

// @ts-expect-error -- Stable formatting is required for strict line width and array layout.
void config({ stylistic: false })
