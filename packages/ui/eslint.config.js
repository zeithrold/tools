import ztd from '@ztd-me/eslint'

export default ztd({
  typescript: { tsconfigPath: 'tsconfig.build.json' },
  react: true,
  // The private harness uses the public package until the reviewed type-alias policy is released.
  rules: { 'ts/consistent-type-definitions': ['error', 'type'] },
  ignores: [
    'dist/**',
    '.artifacts/**',
    'test/consumer/**',
  ],
})
