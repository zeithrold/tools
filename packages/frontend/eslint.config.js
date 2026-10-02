import ztd from '@ztd-me/eslint'

export default ztd({
  typescript: { tsconfigPath: 'tsconfig.build.json' },
  react: true,
  ignores: [
    'dist/**',
    '.artifacts/**',
    'test/consumer/**',
  ],
})
