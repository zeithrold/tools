import antfu from '@antfu/eslint-config'

export default antfu({
  gitignore: false,
  isInEditor: false,
  typescript: false,
  vue: false,
  test: false,
  markdown: false,
  rules: { 'no-console': 'off' },
})
