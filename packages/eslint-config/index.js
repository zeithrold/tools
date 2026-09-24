import antfu from '@antfu/eslint-config'

const strictTypeRules = {
  'ts/no-floating-promises': 'error',
  'ts/no-misused-promises': 'error',
  'ts/no-unsafe-assignment': 'error',
  'ts/no-unsafe-argument': 'error',
  'ts/no-unsafe-call': 'error',
  'ts/no-unsafe-member-access': 'error',
  'ts/no-unsafe-return': 'error',
}

// Consumers retain their own ignores, framework choices, and final overrides.
export function createConfig(options = {}, ...localConfigs) {
  const {
    typescript = true,
    react = false,
    svelte = false,
    ignores = [],
    strictTypes = false,
    rules = {},
    ...antfuOptions
  } = options

  if (strictTypes && (!typescript || typeof typescript !== 'object' || !typescript.tsconfigPath))
    throw new Error('strictTypes requires typescript.tsconfigPath')

  const typeRules = strictTypes ? strictTypeRules : {}
  return antfu({
    typescript,
    react,
    svelte,
    ignores,
    ...antfuOptions,
    rules: {
      'no-console': ['error', { allow: ['warn', 'error'] }],
      ...(typescript ? { 'ts/no-explicit-any': 'error' } : {}),
      ...typeRules,
      ...rules,
    },
  }, ...localConfigs)
}

export default createConfig
