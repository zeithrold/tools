import ztd from '@ztd-me/eslint'

export default ztd({
  typescript: { tsconfigPath: 'tsconfig.json', tsconfigRootDir: import.meta.dirname },
  react: true,
  gitignore: false,
})
