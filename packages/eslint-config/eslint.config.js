// eslint-disable-next-line antfu/no-import-dist -- Self-lint exercises the built package shipped to consumers.
import config from './dist/index.js'

export default config({
  typescript: { tsconfigPath: 'tsconfig.build.json' },
  ignores: [
    'dist/**',
    'test/fixtures/**',
    'test/types/**',
  ],
})
